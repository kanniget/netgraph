# Multi-stage build for netgraph

# Stage 1: build frontend
FROM node:20 AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend .
RUN npm run build

# Stage 2: build backend
FROM golang:1.21 AS backend-builder
WORKDIR /app
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download
COPY server ./server
RUN CGO_ENABLED=0 go build -o netgraph ./server

# Final stage
FROM alpine
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /app/netgraph ./netgraph
COPY --from=frontend-builder /app/frontend/public ./frontend/public
COPY data ./data
EXPOSE 8080
CMD ["./netgraph"]
