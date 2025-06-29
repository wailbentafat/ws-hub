# Scalable and Vendor-Neutral WebSocket Pooler

This project is a complete implementation of a scalable, decoupled architecture designed to handle a large number of stateful WebSocket connections horizontally. It addresses the common challenge of scaling WebSocket servers by separating connection management from business logic using a central message bus and a shared state store.

## The Problem
Standard WebSocket applications are stateful, meaning the server that holds the connection must remember the client's session. This makes horizontal scaling (simply adding more servers) difficult, as there is no easy way to share session information or route messages between users connected to different server instances.

## The Solution & Architecture
This project implements a **WebSocket Pooler** architecture to solve this problem. The core principle is decoupling:

* **WebSocket Poolers:** Lightweight services whose only job is to manage physical WebSocket connections, authenticate clients, and relay messages. They are completely stateless regarding business logic.
* **Backend Service(s):** Services that contain the actual business logic (e.g., chat, presence). They never talk directly to clients.
* **Redis (Message Broker & Store):** Acts as the central communication hub (via Pub/Sub) and a shared state store (via Redis Sets) for presence information.

This design allows for the `pooler` instances to be scaled independently to handle massive connection loads, while the `backend` services can be scaled based on processing needs.

### System Diagram
   +--------------------------+
                 |                          |
                 |  Clients (Web Browsers)  |
                 |                          |
                 +-------------+------------+
                               | (ws://.../ws)
                               v
+----------------------------------+----------------------------------+
| Docker Swarm Cluster                                                |
|                                                                     |
|                  +--------------------------+                         |
|                  | Traefik Load Balancer    |                         |
|                  | (Port :8080)             |                         |
|                  +-----------+--------------+                         |
|                              |                                        |
|      +-----------------------+------------------------+               |
|      |                       |                        |               |
|      v                       v                        v               |
| +----+-----+           +----+-----+            +----+-----+          |
| | Pooler #1|           | Pooler #2|            | Pooler #3|          |
| +----------+           +----------+            +----------+          |
|      ^    |                  ^    |                   ^   |          |
|      |    | (Pub/Sub)        |    | (Pub/Sub)         |   | (Pub/Sub)|
|      |    +------------------+----|-------------------+   |          |
|      |                       |    |                       |          |
|      +-----------------------+----------------------------+----------+
|                              |                                        |
|                              v                                        |
|                  +-----------+--------------+                         |
|                  | Redis                    |                         |
|                  | (Message Bus & State)    |                         |
|                  +-----------+--------------+                         |
|                              ^                                        |
|                              | (Pub/Sub & R/W)                        |
|                              v                                        |
|                  +-----------+--------------+                         |
|                  | Backend Service(s)       |                         |
|                  | (Business & Presence Logic)|                         |
|                  +--------------------------+                         |
|                                                                     |
+---------------------------------------------------------------------+
## Features
* **Horizontal Scalability:** Pooler instances can be scaled on demand to handle increasing connection loads.
* **Decoupled Architecture:** Connection management is fully separated from business logic via Redis Pub/Sub.
* **Centralized State Management:** A real-time Presence System tracks online users using Redis Sets as a shared source of truth.
* **Secure Connections:** WebSocket connections are protected by JWT (JSON Web Token) authentication.
* **High Availability:** Deployed on Docker Swarm, the system can tolerate container crashes and automatically restart services.
* **Automated Load Balancing:** Traefik automatically discovers and load balances traffic across all available pooler instances.
* **Production-Ready Configuration:** Services are deployed with resource limits, restart policies, and rolling update configurations for zero-downtime deployments.

## Technology Stack
* **Backend Language:** Go (Golang)
* **Containerization:** Docker
* **Orchestration:** Docker Swarm
* **Load Balancer:** Traefik
* **Message Broker:** Redis (Pub/Sub)
* **State Store:** Redis (Sets)
* **Core Libraries:** `gorilla/websocket`, `go-redis`, `golang-jwt/jwt`

## Getting Started

### Prerequisites
* [Docker](https://www.docker.com/products/docker-desktop/) installed and running.
* A [Docker Hub](https://hub.docker.com/) account.

### 1. Clone the Repository
```bash
git clone <https://github.com/wailbentafat/ws-hub>
cd <ws-hub>
