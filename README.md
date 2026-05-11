# xray2wg

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](https://hub.docker.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**xray2wg** is a self-hosted VPN gateway that bridges [VLESS+Reality](https://github.com/XTLS/Xray-core) and [WireGuard](https://www.wireguard.com). It lets routers and devices that only speak WireGuard transparently route traffic through VLESS tunnels — without any client-side changes.

---

## How it works

Most consumer routers (MikroTik, Keenetic) support WireGuard natively but cannot speak the VLESS protocol. xray2wg runs as a Docker container and acts as a transparent bridge:

```
┌─────────────────────────────────────────────────────┐
│   Your Router (MikroTik / OpenWrt / any WG client)  │
└────────────────────┬────────────────────────────────┘
                     │ WireGuard (UDP, encrypted)
                     ▼
┌─────────────────────────────────────────────────────┐
│   xray2wg container                                 │
│                                                     │
│   wireguard-go ──► iptables TPROXY ──► xray-core   │
│   (userspace WG)                    (VLESS outbound)│
└────────────────────┬────────────────────────────────┘
                     │ VLESS+Reality (TCP/TLS, encrypted)
                     ▼
┌─────────────────────────────────────────────────────┐
│   Your VLESS upstream server                        │
└─────────────────────────────────────────────────────┘
```

Traffic from WG clients is intercepted transparently via `TPROXY` and forwarded through one or more VLESS nodes. The client never needs to know about VLESS at all.

### Key features

| Feature | Details |
|---------|---------|
| **Multiple VLESS nodes per tunnel** | Assign several nodes and balance traffic between them |
| **Round Robin balancing** | Distribute connections evenly across all nodes |
| **Least Ping balancing** | Route to the fastest node, with automatic health probes every 10 s |
| **WireGuard config export** | Download `.conf` or copy MikroTik RouterOS commands |
| **QR code generation** | Scan from any WG client app |
| **Real-time traffic stats** | Per-tunnel and per-peer RX/TX rates via WebSocket |
| **VLESS subscriptions** | Paste a subscription URL; nodes refresh automatically |
| **Encryption at rest** | Private keys stored AES-256-GCM encrypted |
| **Single binary** | Frontend embedded; no external dependencies |

---

## Quick start

### Prerequisites

- Docker ≥ 24 with Compose v2
- Linux host with `NET_ADMIN` capability (bare metal or VM — **not** Docker Desktop on macOS/Windows)
- A working VLESS+Reality server (or a subscription URL)

### 1. Clone and configure

```bash
git clone https://github.com/your-username/xray2wg.git
cd xray2wg
mkdir -p data
```

Edit `docker-compose.yml` — the only required change is your host address for CORS:

```yaml
environment:
  - CORS_ALLOWED_ORIGINS=https://192.168.1.100:8080   # your server's LAN IP
  - PUBLIC_HOST=192.168.1.100                         # used in WG peer exports
```

### 2. Start

```bash
docker compose up -d
```

The container exposes:
- **`8080/tcp`** — Web UI (HTTPS with auto-generated certificate)
- **`51820–51830/udp`** — WireGuard ports (one per tunnel)

### 3. Open the Web UI

Navigate to `https://<your-server-ip>:8080`.

On first launch you will be prompted to set an admin password. Your browser will warn about the self-signed certificate — accept it, or provide your own cert via `TLS_CERT_FILE`/`TLS_KEY_FILE`.

### 4. Create a tunnel

1. **Subscriptions → Add** — paste your VLESS subscription URL
2. **Tunnels → New tunnel** — pick a subscription, select one or more nodes, choose a balancing strategy
3. **Start** the tunnel
4. **Add a peer** and download the WireGuard config or scan the QR code

---

## Load balancing

xray2wg supports assigning multiple VLESS nodes to a single WireGuard tunnel. Traffic is distributed by xray-core's built-in balancer — no additional iptables rules needed.

### Strategies

| Strategy | Behaviour | Best for |
|----------|-----------|---------|
| **Round Robin** | Each new connection goes to the next node in order | Equal-spec nodes, no latency preference |
| **Least Ping** | Routes to the node with lowest measured round-trip time; probes every 10 s | Geographically diverse nodes |

### Configuring via UI

When creating or editing a tunnel, the **Nodes** section replaces the old single-node dropdown:

- Add nodes from any subscription (use the search box)
- Drag-reorder with the ▲/▼ buttons to set priority
- Choose a strategy with the radio buttons

The **Nodes** tab on the tunnel detail page shows live health status for each node (latency and availability) when running with Least Ping.

### Configuring via API

```bash
# Assign nodes and set strategy
curl -X PUT https://host:8080/api/v1/tunnels/1/nodes \
  -H "Content-Type: application/json" \
  -b "access_token=<jwt>" \
  -d '{"node_ids": [3, 7, 12], "strategy": "least_ping"}'

# Read current nodes with health data
curl https://host:8080/api/v1/tunnels/1/nodes -b "access_token=<jwt>"
```

---

## Configuration reference

All configuration is done via environment variables.

### Required

| Variable | Description |
|----------|-------------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of exact browser origins allowed for CORS and WebSocket (`scheme://host[:port]`). Example: `https://192.168.1.100:8080` |

### TLS

| Variable | Default | Description |
|----------|---------|-------------|
| `TLS_OFF` / `HTTP_PLAIN` | `false` | Set to `true` to disable TLS (plain HTTP on `PORT`) |
| `TLS_AUTO_CERT` | `true` | Auto-generate RSA-4096 cert under `$DATA_DIR/tls/` if cert files are missing |
| `TLS_CERT_FILE` | — | Path to PEM certificate (skips auto-generation when both cert+key are provided) |
| `TLS_KEY_FILE` | — | Path to PEM private key |
| `HTTP_REDIRECT_ADDR` | — | Optional bind address (e.g. `:80`) for HTTP→HTTPS redirect |
| `BEHIND_HTTPS` | `false` | Set to `true` when behind a TLS-terminating reverse proxy; marks auth cookies `Secure` |

### General

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_DIR` | `./data` | Persistent state: SQLite DB, master key, JWT key, TLS certs |
| `PORT` | `8080` | Listen port for the web server |
| `PUBLIC_HOST` | — | Server IP/hostname used in WireGuard peer exports when not set via Settings |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `XRAY_LOG_LEVEL` | `warning` | xray-core log verbosity |
| `XRAY2WG_TUNNEL_TRACE` | — | Set to `1` to enable verbose tunnel lifecycle logging |
| `METRICS_API_KEY` | — | If set, `GET /metrics` requires `X-Metrics-Key` header; otherwise loopback-only |
| `CSP_OFF` | `false` | Disable Content-Security-Policy (development only) |

---

## Building from source

### Prerequisites

- Go 1.23+
- Node.js 22+
- Linux (for iptables / WireGuard tests) or Docker

```bash
# Build frontend
cd frontend
npm ci
npm run build
cd ..

# Build backend (embeds frontend dist)
cd backend
cp -r ../frontend/dist ./staticfs/
go build -o ../xray2wg ./cmd/server

# Or build the Docker image
docker build -f docker/Dockerfile -t xray2wg .
```

### Running locally (Linux only)

```bash
export CORS_ALLOWED_ORIGINS=https://localhost:8080
export DATA_DIR=./data
mkdir -p data
sudo ./xray2wg   # NET_ADMIN required for iptables / WireGuard
```

### Running tests

```bash
cd backend
go test ./...

# With race detector (requires CGO + gcc)
go test -race ./...
```

---

## WireGuard peer setup

### Any WG client

1. In the tunnel detail page, go to **Peers → Add peer**
2. Download the `.conf` file or scan the QR code
3. Import into any WireGuard client (mobile, desktop, router)

### MikroTik RouterOS

1. Open the peer config page, switch to the **MikroTik** tab
2. Copy the RouterOS commands
3. Paste into a MikroTik terminal (`/ip address`, `/interface wireguard`, etc. are pre-generated)

> **Note:** Routing rules (which traffic to send through the WG interface) must be configured manually on the router based on your network topology.

---

## API reference

The REST API is available at `/api/v1`. All endpoints except auth probes require a valid session cookie.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/auth/login` | Authenticate with password |
| `GET` | `/auth/me` | Check session |
| `GET` | `/tunnels` | List all tunnels |
| `POST` | `/tunnels` | Create tunnel |
| `GET` | `/tunnels/:id` | Get tunnel |
| `PUT` | `/tunnels/:id` | Update tunnel |
| `DELETE` | `/tunnels/:id` | Delete tunnel |
| `POST` | `/tunnels/:id/start` | Start tunnel |
| `POST` | `/tunnels/:id/stop` | Stop tunnel |
| `GET` | `/tunnels/:id/nodes` | List VLESS nodes with health |
| `PUT` | `/tunnels/:id/nodes` | Set VLESS nodes and strategy |
| `GET` | `/tunnels/:id/peers` | List WireGuard peers |
| `POST` | `/tunnels/:id/peers` | Add peer |
| `GET` | `/tunnels/:id/peers/:pid/config` | WireGuard `.conf` for peer |
| `GET` | `/tunnels/:id/peers/:pid/mikrotik` | MikroTik RouterOS commands |
| `GET` | `/tunnels/:id/stats` | Traffic history (`?window=1h\|6h\|24h`) |
| `GET` | `/subscriptions` | List subscriptions |
| `POST` | `/subscriptions` | Add subscription |
| `POST` | `/subscriptions/:id/refresh` | Refresh nodes now |
| `GET` | `/ws/stats` | WebSocket real-time stats stream |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/health` | Liveness probe |
| `GET` | `/ready` | Readiness probe |

---

## Project structure

```
xray2wg/
├── backend/                    # Go application
│   ├── cmd/server/             # Entry point, DI wiring
│   └── internal/
│       ├── domain/             # Entities and repository interfaces
│       ├── service/            # Business logic
│       ├── infrastructure/     # xray-core, wireguard-go, SQLite, iptables
│       ├── infra/              # Adapter layer
│       └── api/                # HTTP handlers, middleware, WebSocket
│
├── frontend/                   # React 18 + Vite + TypeScript
│   └── src/
│       ├── pages/              # Route components
│       ├── components/         # Shared UI components
│       ├── services/api.ts     # Typed API client
│       └── hooks/              # WebSocket stats hook
│
├── docker/
│   ├── Dockerfile              # Multi-stage build
│   └── entrypoint.sh
│
└── docker-compose.yml
```

---

## Security notes

- **Private keys** (WireGuard and PSK) are stored AES-256-GCM encrypted with a master key in `data/master.key`. Back this file up.
- **JWT signing key** is auto-generated RSA-4096 at first start (`data/jwt_private.pem`).
- **TLS certificate** is auto-generated RSA-4096 if no cert is provided. For production, provide your own cert via `TLS_CERT_FILE`/`TLS_KEY_FILE`.
- The container requires `CAP_NET_ADMIN` and `/dev/net/tun`. Run it on a dedicated host or a properly isolated VM.
- `NET_ADMIN` is needed only for iptables TPROXY rules and the WireGuard userspace device. The web server process runs as the container user.

---

## Contributing

1. Fork the repository and create a feature branch
2. Make your changes — run `go test ./...` and `npm run build` before submitting
3. Open a pull request with a clear description of what changes and why

Please keep PRs focused. For large features, open an issue first to discuss the approach.

---

## License

[MIT](LICENSE) — © 2026 xray2wg contributors
