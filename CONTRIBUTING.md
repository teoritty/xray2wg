# Contributing to xray2wg

Thank you for your interest in contributing. This document explains how to get involved, what we expect from contributors, and how the project is structured.

---

## Table of contents

1. [Code of conduct](#code-of-conduct)
2. [Reporting bugs](#reporting-bugs)
3. [Requesting features](#requesting-features)
4. [Development setup](#development-setup)
5. [Making changes](#making-changes)
6. [Pull request process](#pull-request-process)
7. [Commit style](#commit-style)
8. [Testing](#testing)
9. [Project layout](#project-layout)

---

## Code of conduct

Be respectful. Harassment, personal attacks, and discrimination of any kind will not be tolerated. Treat everyone as a collaborator, not a competitor.

---

## Reporting bugs

Use the **Bug Report** issue template. Before you open a new issue:

- Search [existing issues](https://github.com/teoritty/xray2wg/issues) (including closed ones).
- Reproduce the problem on the latest image or commit.
- Collect debug logs: set `LOG_LEVEL=debug` and `XRAY_LOG_LEVEL=debug` in `docker-compose.yml`, then `docker compose logs --tail 500 xray2wg`.

A good bug report includes:
- A minimal, reproducible set of steps.
- Exact version / git SHA.
- Host OS and kernel (`uname -sr`).
- Relevant log output and a sanitised `docker-compose.yml`.

---

## Requesting features

Use the **Feature Request** issue template. For large architectural changes — new proxying backends, significant API changes, new balancing strategies — **open an issue first** and discuss the design before writing code. This saves everyone time.

---

## Development setup

### Prerequisites

| Tool | Minimum version |
|------|----------------|
| Go | 1.23 |
| Node.js | 22 |
| Docker + Compose v2 | 24 |
| Linux host or VM | kernel ≥ 5.10 (for iptables TPROXY) |

> **macOS / Windows:** You can build the frontend and run unit tests, but a Linux environment (VM, WSL 2 with a real kernel, or a remote host) is required for end-to-end testing because iptables and WireGuard are Linux-only.

### Clone and build

```bash
git clone https://github.com/teoritty/xray2wg.git
cd xray2wg

# Build the React frontend
cd frontend
npm ci
npm run build
cd ..

# Build the Go backend (embeds the frontend)
cd backend
cp -r ../frontend/dist ./staticfs/
go build -o ../xray2wg ./cmd/server
cd ..

# Or build the Docker image (recommended for testing the full stack)
docker build -f docker/Dockerfile -t xray2wg:dev .
```

### Running locally (Linux)

```bash
export CORS_ALLOWED_ORIGINS=https://localhost:8080
export DATA_DIR=./data
mkdir -p data
sudo ./xray2wg   # NET_ADMIN required
```

---

## Making changes

1. **Fork** the repository and create a branch from `main`:
   ```bash
   git checkout -b fix/icmp-tproxy
   ```

2. **Keep changes focused.** One logical change per PR. Refactors, formatting fixes, and feature work should be in separate PRs.

3. **Follow existing patterns.** Read a few nearby files before writing new code. The project uses:
   - Go standard library conventions (`errors.Is`, `context`-threaded calls, zerolog for structured logging).
   - React 18 with TypeScript; no class components.
   - SQLite via `modernc.org/sqlite` — no CGO dependency.

4. **No speculative changes.** Do not add error handling for cases that cannot occur, config flags for features that do not exist yet, or abstractions for hypothetical future requirements.

5. **Run the linter before committing:**
   ```bash
   cd backend && go vet ./...
   cd frontend && npm run lint
   ```

---

## Pull request process

1. Ensure all tests pass (see [Testing](#testing)).
2. Rebase on the latest `main` before opening the PR.
3. Write a clear PR description:
   - **What** changed.
   - **Why** (link the relevant issue, e.g. `Fixes #42`).
   - **How to test** it.
4. Keep the diff small and reviewable. Huge PRs with mixed concerns will be asked to be split.
5. A maintainer will review within a reasonable time. Be patient and respond to feedback promptly.
6. PRs are merged by the maintainer using **squash merge** unless the commit history is particularly clean and meaningful.

---

## Commit style

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

[optional body]

[optional footer: Fixes #<issue>]
```

| Type | When to use |
|------|-------------|
| `fix` | Bug fix |
| `feat` | New feature |
| `refactor` | Code change that is neither a fix nor a feature |
| `docs` | Documentation only |
| `test` | Adding or fixing tests |
| `chore` | Build, CI, dependency updates |

Examples:
```
fix(tproxy): bypass ICMP in mangle chain so ping works
feat(balancer): add weighted round-robin strategy
docs: add CONTRIBUTING guide and issue templates
```

---

## Testing

### Backend unit tests

```bash
cd backend
go test ./...

# With race detector (requires CGO + gcc)
go test -race ./...
```

### Frontend type-check and lint

```bash
cd frontend
npm run build   # type-checks via tsc
npm run lint
```

### Integration / end-to-end

There is currently no automated end-to-end test suite. When making changes to the iptables rules, WireGuard lifecycle, or xray-core configuration, test manually:

1. Build the Docker image with your changes.
2. Start a tunnel with a real VLESS upstream.
3. Connect a WireGuard peer and verify TCP, UDP, and ICMP traffic.
4. Restart the container and confirm the tunnel restores cleanly.

---

## Project layout

```
xray2wg/
├── backend/
│   ├── cmd/server/             # Entry point and dependency wiring
│   └── internal/
│       ├── domain/             # Entities and repository interfaces
│       ├── service/            # Business logic (tunnel, peer, subscription)
│       ├── infrastructure/     # xray-core adapter, wireguard-go, SQLite, iptables
│       └── api/                # HTTP handlers, middleware, WebSocket
│
├── frontend/                   # React 18 + Vite + TypeScript
│   └── src/
│       ├── pages/              # Route-level components
│       ├── components/         # Shared UI components
│       ├── services/api.ts     # Typed REST client
│       └── hooks/              # WebSocket stats hook
│
├── docker/
│   ├── Dockerfile              # Multi-stage build (Node → Go → runtime)
│   └── entrypoint.sh
│
├── .github/
│   └── ISSUE_TEMPLATE/         # Bug report and feature request forms
│
├── docker-compose.yml          # Reference deployment (all env vars documented here)
└── README.md
```

---

## Questions?

Open a [Discussion](https://github.com/teoritty/xray2wg/discussions) rather than an issue for general questions. Issues are for confirmed bugs and feature proposals.
