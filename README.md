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
