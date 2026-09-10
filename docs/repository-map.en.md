# Repository Map

[Russian version](repository-map.md)

| Path | Purpose |
|---|---|
| `cmd/admin/main.go` | Entry point of the administrator Telegram bot |
| `internal/bot/bot.go` | Telebot initialization, menu, middleware, and route registration |
| `internal/bot/handlers_menu.go` | Main menu `/start`, `/help`, client search, and `/status` metrics |
| `internal/bot/handlers_inbounds.go` | Inbound list viewing, online status badges, and details inspection |
| `internal/bot/handlers_inbound_create.go` | Step-by-step FSM wizard for creating new VLESS-Reality inbounds |
| `internal/bot/handlers_wizard.go` | Step-by-step FSM wizard for adding new clients to inbounds |
| `internal/bot/handlers_clients.go` | Client card, universal links/subLinks, status toggle, reset, deletion |
| `internal/bot/handlers_servers.go` | Multi-server listing and active server context switcher (`/servers`) |
| `internal/bot/handlers_logs.go` | Inspection and document upload of system logs x-ui and Xray (`/logs`) |
| `internal/bot/fsm.go` | In-memory FSM conversation state storage for inbound and client wizards |
| `internal/bot/messages.go` | Message templates, HTML escaping, and output formatting |
| `internal/bot/middleware.go` | Panic recovery, structured logging, and `ADMIN_IDS` allowlist filtering |
| `internal/config/` | Configuration loading and validation from `.env` (Telegram, 3x-ui, SSH, DB) |
| `internal/sshtunnel/` | Secure in-memory SSH port forwarding tunnel and command execution |
| `internal/storage/` | SQLite repositories for audit logs (`audit_logs`) and servers registry (`servers`) |
| `internal/updater/` | Background GitHub API release polling and version comparison cache |
| `internal/vless/` | Universal link generator (VLESS/VMess/Trojan/SS), QR codes, UUIDv4 |
| `internal/xui/` | Dual-format 3x-ui panel API client (sessions, inbounds, clients, logs) |
| `docs/` | Complete operational and technical documentation set |
| `admin-bot.service` | Systemd unit file for continuous bot execution on the server |
| `deploy.sh` | Automated compilation and deployment script for remote servers |
| `Dockerfile` | Multi-stage build definition for a minimal Docker image |
| `docker-compose.yml` | Docker Compose stack configuration for running the service |
| `VERSION` | Current project release version file |
| `LICENSE` | Project license file |
| `CHANGELOG.md` | Release and version history |

Runtime files `.env`, SQLite databases `*.db`, and compiled binaries are excluded from Git.
