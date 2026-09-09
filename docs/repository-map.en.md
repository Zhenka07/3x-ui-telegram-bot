# Repository Map

[Russian version](repository-map.md)

| Path | Purpose |
|---|---|
| `cmd/admin/main.go` | Entry point of the administrator Telegram bot |
| `internal/bot/bot.go` | Telebot initialization, menu, middleware, and route registration |
| `internal/bot/handlers_menu.go` | Main menu `/start`, `/help`, and system monitoring `/server` |
| `internal/bot/handlers_inbounds.go` | Inbound list viewing, details inspection, and traffic reset |
| `internal/bot/handlers_inbound_create.go` | Step-by-step FSM wizard for creating new VLESS-Reality inbounds |
| `internal/bot/handlers_wizard.go` | Step-by-step FSM wizard for adding new clients to inbounds |
| `internal/bot/handlers_clients.go` | Client search, details card, enable/disable toggle, reset, and deletion |
| `internal/bot/fsm.go` | In-memory FSM conversation state storage for inbound and client wizards |
| `internal/bot/messages.go` | Message templates, HTML escaping, and output formatting |
| `internal/bot/middleware.go` | Panic recovery, structured logging, and `ADMIN_IDS` allowlist filtering |
| `internal/config/` | Configuration loading and validation from `.env` and environment |
| `internal/storage/` | SQLite repository for administrative action audit logging (`audit_logs`) |
| `internal/vless/` | VLESS URI builder, in-memory QR code generator, UUIDv4, and Reality parser |
| `internal/xui/` | Dual-format 3x-ui panel API client (sessions, inbounds, clients, certs, status) |
| `docs/` | Complete operational and technical documentation set |
| `admin-bot.service` | Systemd unit file for continuous bot execution on the server |
| `deploy.sh` | Automated compilation and deployment script for remote servers |
| `Dockerfile` | Multi-stage build definition for a minimal Docker image |
| `docker-compose.yml` | Docker Compose stack configuration for running the service |
| `VERSION` | Current project release version file |
| `LICENSE` | Project license file |
| `CHANGELOG.md` | Release and version history |

Runtime files `.env`, SQLite databases `*.db`, and compiled binaries are excluded from Git.
