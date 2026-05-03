# Distributed MD5 Cracker

## Category
Distributed Systems

## How to Run
1. **Start the Manager:**
   `go run .` (Starts on port 8080)

2. **Start a Worker:**
   `BIND_PORT=7947 JOIN_ADDR=localhost:7946 go run .` (Starts on port 8081)

3. **Access the UI:**
   Open `http://localhost:8080` (or the port of whichever node is the MANAGER).

## Tech Stack
- *Hashicorp Memberlist:* Gossip protocol for node discovery.
- *gRPC:* For high-speed task dispatching.
- *Gin:* REST API and Web UI hosting.
- *Go:* Core logic.

## Containerization & Results
*   **Docker Orchestration:** Fully containerized using `docker-compose` for automated cluster setup and scaling.
*   **Task Distribution:** Manager partitions alphabet ranges (e.g., `a-j`, `k-z`) across workers via gRPC streams.
*   **Verified Success:** Successfully cracks 4-character hashes (e.g., `pass`) with real-time task cancellation once found.
*   **Persistence:** Saves results to `/root/result.json` using Docker volumes to persist data across container restarts.
