# Multi-stage build for netgraph

# Stage 1: build frontend
FROM node:20 AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend .
RUN npm run build

# Stage 2: build backend
FROM golang:1.24.3 AS backend-builder
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server .
RUN CGO_ENABLED=0 go build -o /app/netgraph

# build extraction tool
WORKDIR /app/proxmoxsync
COPY cmd/proxmoxsync/go.mod ./
RUN go mod download
COPY cmd/proxmoxsync .
RUN CGO_ENABLED=0 go build -o /app/proxmoxsync

# Final stage
FROM alpine
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /app/netgraph ./netgraph
COPY --from=backend-builder /app/proxmoxsync ./proxmoxsync
COPY --from=frontend-builder /app/frontend/public ./frontend/public
COPY data ./data
EXPOSE 8080
CMD ["./netgraph"]
