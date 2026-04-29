# Distributed MD5 Cracker

## How to Run
1. **Start the Manager:**
   `go run .` (Starts on port 8080)

2. **Start a Worker:**
   `BIND_PORT=7947 JOIN_ADDR=localhost:7946 go run .` (Starts on port 8081)

3. **Access the UI:**
   Open `http://localhost:8080` (or the port of whichever node is the MANAGER could be 81).

## Tech Stack
- *Hashicorp Memberlist:* Gossip protocol for node discovery.
- *gRPC:* For high-speed task dispatching.
- *Gin:* REST API and Web UI hosting.
- *Go:* Core logic.