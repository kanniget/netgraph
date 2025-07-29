# netgraph

This project contains a Go backend using `gorilla/mux` and a Svelte frontend. The frontend visualizes a simple network using D3.js.

## Build Frontend

```bash
cd frontend
npm install
npm run build
```

## Run Server

```bash
go run ./server
```

The server hosts the built frontend and provides the `/api/graph` endpoint serving a JSON graph dataset.

## Docker Compose

Build and run the stack using [Docker Compose](https://docs.docker.com/compose/):

```bash
docker compose up --build
```

## Proxmox Sync Tool

A helper CLI `proxmoxsync` queries a Proxmox host using the REST API and writes a graph definition to `data/graph.json`.

### Usage

```bash
# build the tool
go build ./cmd/proxmoxsync

# run against a Proxmox host
./proxmoxsync -host https://pve.example.com:8006 -user root@pam -pass secret
```

The tool retrieves SDN networks, hosts, and the network interfaces each host is attached to. Networks and hosts are added as nodes while links between them represent the attached interfaces.

