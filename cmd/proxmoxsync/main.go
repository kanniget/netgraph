package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var verbose bool

func logf(format string, a ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", a...)
	}
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Name string         `json:"name"`
	Info map[string]any `json:"info,omitempty"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ticketResponse struct {
	Data struct {
		Ticket string `json:"ticket"`
	} `json:"data"`
}

type vmInfo struct {
	Node string
	VMID string
	Name string
}

func main() {
	host := flag.String("host", "", "Proxmox host (e.g. https://pve:8006)")
	user := flag.String("user", "root@pam", "API user")
	pass := flag.String("pass", "", "API password")
	out := flag.String("out", filepath.Join("data", "graph.json"), "output graph json (deprecated)")
	outfile := flag.String("file", "", "output graph json file")
	insecure := flag.Bool("insecure", false, "ignore TLS certificate errors")
	verboseFlag := flag.Bool("verbose", false, "enable verbose output")
	ignoreTypesFlag := flag.String("ignore", "", "comma-separated node types to ignore")
	flag.StringVar(ignoreTypesFlag, "i", "", "comma-separated node types to ignore (shorthand)")
	flag.BoolVar(verboseFlag, "v", false, "enable verbose output (shorthand)")
	flag.Parse()

	verbose = *verboseFlag
	if verbose {
		fmt.Println("verbose output enabled")
	}
	ignore := make(map[string]bool)
	if *ignoreTypesFlag != "" {
		for _, t := range strings.Split(*ignoreTypesFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				ignore[strings.ToLower(t)] = true
			}
		}
	}

	if *host == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "host and pass are required")
		os.Exit(1)
	}

	var client *http.Client
	if *insecure {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}

	logf("logging in to %s as %s", *host, *user)
	ticket, err := login(client, *host, *user, *pass)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login error:", err)
		os.Exit(1)
	}

	graph := Graph{}
	nodeMap := make(map[string]*Node)
	addNode := func(n Node) {
		if existing, ok := nodeMap[n.ID]; ok {
			if n.Info != nil {
				if existing.Info == nil {
					existing.Info = make(map[string]any)
				}
				for k, v := range n.Info {
					existing.Info[k] = v
				}
			}
		} else {
			graph.Nodes = append(graph.Nodes, n)
			nodeMap[n.ID] = &graph.Nodes[len(graph.Nodes)-1]
		}
	}

	if !ignore["zone"] {
		logf("retrieving zones")
		zones, err := getZones(client, *host, ticket)
		if err != nil {
			fmt.Fprintln(os.Stderr, "get zones error:", err)
		}
		logf("retrieved %d zones", len(zones))

		for _, z := range zones {
			addNode(Node{ID: z, Type: "zone", Name: z})
		}
	}

	if !ignore["net"] {
		logf("retrieving networks")
		networks, err := getNetworks(client, *host, ticket)
		if err != nil {
			fmt.Fprintln(os.Stderr, "get networks error:", err)
		}
		logf("retrieved %d networks", len(networks))

		for _, n := range networks {
			info := map[string]any{}
			if n.Zone != "" {
				info["zone"] = n.Zone
			}
			if n.Bridge != "" {
				info["bridge"] = n.Bridge
			}
			addNode(Node{ID: n.ID, Type: "net", Name: n.ID, Info: info})
			if n.Zone != "" && !ignore["zone"] {
				addNode(Node{ID: n.Zone, Type: "zone", Name: n.Zone})
				graph.Links = append(graph.Links, Link{Source: n.ID, Target: n.Zone})
			}
			if n.Bridge != "" && !ignore["bridge"] {
				addNode(Node{ID: n.Bridge, Type: "bridge", Name: n.Bridge})
				graph.Links = append(graph.Links, Link{Source: n.ID, Target: n.Bridge})
			}
		}
	}

	if !ignore["host"] {
		logf("retrieving hosts")
		hosts, err := getHosts(client, *host, ticket)
		if err != nil {
			fmt.Fprintln(os.Stderr, "get hosts error:", err)
		}
		logf("retrieved %d hosts", len(hosts))

		for _, h := range hosts {
			logf("retrieving interfaces for host %s", h)
			ifaces, err := getHostIfaces(client, *host, ticket, h)
			if err != nil {
				fmt.Fprintln(os.Stderr, "get host interfaces error:", err)
				continue
			}
			logf("host %s has %d interfaces", h, len(ifaces))
			ifaceList := []map[string]string{}
			for _, iface := range ifaces {
				ifaceList = append(ifaceList, map[string]string{"name": iface.Name, "kind": iface.Kind})
			}
			addNode(Node{ID: h, Type: "host", Name: h, Info: map[string]any{"interfaces": ifaceList}})

			for _, iface := range ifaces {
				nodeType := iface.Kind
				if nodeType == "bridge" || nodeType == "OVSBridge" {
					nodeType = "bridge"
				} else {
					nodeType = "nic"
				}
				if ignore[nodeType] {
					continue
				}
				addNode(Node{ID: iface.Name, Type: nodeType, Name: iface.Name, Info: map[string]any{"host": h, "kind": iface.Kind}})
				graph.Links = append(graph.Links, Link{Source: iface.Name, Target: h})
			}
		}
	}

	if !ignore["vm"] {
		logf("retrieving VMs")
		vms, err := getVMs(client, *host, ticket)
		if err != nil {
			fmt.Fprintln(os.Stderr, "get vms error:", err)
		}
		logf("retrieved %d VMs", len(vms))

		for _, v := range vms {
			logf("retrieving interfaces for VM %s", v.Name)
			ifaces, err := getVMIfaces(client, *host, ticket, v)
			if err != nil {
				fmt.Fprintln(os.Stderr, "get vm interfaces error:", err)
				continue
			}
			logf("VM %s has %d interfaces", v.Name, len(ifaces))
			disks, err := getVMDisks(client, *host, ticket, v)
			if err != nil {
				fmt.Fprintln(os.Stderr, "get vm disks error:", err)
				continue
			}
			logf("VM %s has %d disks", v.Name, len(disks))
			addNode(Node{ID: v.Name, Type: "vm", Name: v.Name, Info: map[string]any{"node": v.Node, "vmid": v.VMID, "interfaces": ifaces, "disks": disks}})
			for _, iface := range ifaces {
				if ignore["bridge"] {
					continue
				}
				addNode(Node{ID: iface, Type: "bridge", Name: iface})
				graph.Links = append(graph.Links, Link{Source: iface, Target: v.Name})
			}
			for _, disk := range disks {
				if ignore["disk"] {
					continue
				}
				addNode(Node{ID: disk, Type: "disk", Name: disk, Info: map[string]any{"vm": v.Name}})
				graph.Links = append(graph.Links, Link{Source: disk, Target: v.Name})
			}
		}
	}

	b, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := *out
	if *outfile != "" {
		path = *outfile
	}
	logf("writing graph to %s", path)
	if err := os.WriteFile(path, b, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func login(client *http.Client, host, user, pass string) (string, error) {
	data := fmt.Sprintf("username=%s&password=%s", user, pass)
	req, err := http.NewRequest("POST", host+"/api2/json/access/ticket", bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s: %s", resp.Status, string(body))
	}

	var t ticketResponse
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}
	return t.Data.Ticket, nil
}

type listResponse struct {
	Data []struct {
		ID     string `json:"id"`
		Node   string `json:"node"`
		Iface  string `json:"iface"`
		Zone   string `json:"zone"`
		Bridge string `json:"bridge"`
		VNet   string `json:"vnet"`
		Type   string `json:"type"`
	} `json:"data"`
}

type ifaceInfo struct {
	Name string
	Kind string
}

type networkInfo struct {
	ID     string
	Zone   string
	Bridge string
}

func getNetworks(client *http.Client, host, ticket string) ([]networkInfo, error) {
	req, _ := http.NewRequest("GET", host+"/api2/json/cluster/sdn/vnets", nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var nets []networkInfo
	for _, d := range lr.Data {
		id := d.ID
		if id == "" {
			id = d.VNet
		}
		if id != "" {
			nets = append(nets, networkInfo{ID: id, Zone: d.Zone, Bridge: d.Bridge})
		}
	}
	return nets, nil
}

func getZones(client *http.Client, host, ticket string) ([]string, error) {
	req, _ := http.NewRequest("GET", host+"/api2/json/cluster/sdn/zones", nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var zones []string
	for _, d := range lr.Data {
		switch {
		case d.ID != "":
			zones = append(zones, d.ID)
		case d.Zone != "":
			zones = append(zones, d.Zone)
		}
	}
	return zones, nil
}

func getHosts(client *http.Client, host, ticket string) ([]string, error) {
	req, _ := http.NewRequest("GET", host+"/api2/json/nodes", nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var hosts []string
	for _, d := range lr.Data {
		if d.Node != "" {
			hosts = append(hosts, d.Node)
		}
	}
	return hosts, nil
}

func getHostIfaces(client *http.Client, host, ticket, node string) ([]ifaceInfo, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api2/json/nodes/%s/network", host, node), nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var ifaces []ifaceInfo
	for _, d := range lr.Data {
		if d.Iface != "" {
			kind := d.Type
			if kind == "" {
				kind = "nic"
			}
			ifaces = append(ifaces, ifaceInfo{Name: d.Iface, Kind: kind})
		}
	}
	return ifaces, nil
}

func getVMs(client *http.Client, host, ticket string) ([]vmInfo, error) {
	req, _ := http.NewRequest("GET", host+"/api2/json/cluster/resources?type=vm", nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var lr struct {
		Data []struct {
			VMID json.Number `json:"vmid"`
			Name string      `json:"name"`
			Node string      `json:"node"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var vms []vmInfo
	for _, d := range lr.Data {
		if d.VMID.String() != "" && d.Name != "" {
			vms = append(vms, vmInfo{Node: d.Node, VMID: d.VMID.String(), Name: d.Name})
		}
	}
	return vms, nil
}

func getVMIfaces(client *http.Client, host, ticket string, vm vmInfo) ([]string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%s/config", host, vm.Node, vm.VMID), nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var cfg struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	var ifaces []string
	for k, v := range cfg.Data {
		if strings.HasPrefix(k, "net") {
			if val, ok := v.(string); ok {
				parts := strings.Split(val, ",")
				for _, p := range parts {
					if strings.HasPrefix(p, "bridge=") {
						b := strings.TrimPrefix(p, "bridge=")
						if b != "" {
							ifaces = append(ifaces, b)
						}
					}
				}
			}
		}
	}
	return ifaces, nil
}

func getVMDisks(client *http.Client, host, ticket string, vm vmInfo) ([]string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%s/config", host, vm.Node, vm.VMID), nil)
	req.Header.Set("Cookie", "PVEAuthCookie="+ticket)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s: %s", resp.Status, string(body))
	}
	var cfg struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	var disks []string
	for k, v := range cfg.Data {
		// ignore scsi controller entries
		if k == "scsihw" {
			continue
		}
		if strings.HasPrefix(k, "scsi") || strings.HasPrefix(k, "sata") || strings.HasPrefix(k, "ide") || strings.HasPrefix(k, "virtio") {
			if val, ok := v.(string); ok {
				// skip cdrom devices
				if strings.Contains(val, "media=cdrom") {
					continue
				}
				parts := strings.Split(val, ",")
				if len(parts) > 0 {
					disk := parts[0]
					disk = strings.TrimSpace(disk)
					if disk != "" {
						disks = append(disks, disk)
					}
				}
			}
		}
	}
	return disks, nil
}
