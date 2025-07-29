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

type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
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
	flag.Parse()

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

	ticket, err := login(client, *host, *user, *pass)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login error:", err)
		os.Exit(1)
	}

	graph := Graph{}

	networks, err := getNetworks(client, *host, ticket)
	if err != nil {
		fmt.Fprintln(os.Stderr, "get networks error:", err)
	}

	for _, n := range networks {
		graph.Nodes = append(graph.Nodes, Node{ID: n, Type: "net", Name: n})
	}

	vms, err := getVMs(client, *host, ticket)
	if err != nil {
		fmt.Fprintln(os.Stderr, "get vms error:", err)
	}

	for _, v := range vms {
		graph.Nodes = append(graph.Nodes, Node{ID: v.Name, Type: "host", Name: v.Name})
	}

	for _, v := range vms {
		ifaces, err := getVMIfaces(client, *host, ticket, v)
		if err != nil {
			fmt.Fprintln(os.Stderr, "get vm interfaces error:", err)
			continue
		}
		for _, iface := range ifaces {
			graph.Links = append(graph.Links, Link{Source: iface, Target: v.Name})
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
		ID    string `json:"id"`
		Node  string `json:"node"`
		Iface string `json:"iface"`
	} `json:"data"`
}

func getNetworks(client *http.Client, host, ticket string) ([]string, error) {
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
	var nets []string
	for _, d := range lr.Data {
		if d.ID != "" {
			nets = append(nets, d.ID)
		}
	}
	return nets, nil
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

func getHostIfaces(client *http.Client, host, ticket, node string) ([]string, error) {
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
	var ifaces []string
	for _, d := range lr.Data {
		if d.Iface != "" {
			ifaces = append(ifaces, d.Iface)
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
			VMID string `json:"vmid"`
			Name string `json:"name"`
			Node string `json:"node"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	var vms []vmInfo
	for _, d := range lr.Data {
		if d.VMID != "" && d.Name != "" {
			vms = append(vms, vmInfo{Node: d.Node, VMID: d.VMID, Name: d.Name})
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
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	var ifaces []string
	for k, v := range cfg.Data {
		if strings.HasPrefix(k, "net") {
			parts := strings.Split(v, ",")
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
	return ifaces, nil
}
