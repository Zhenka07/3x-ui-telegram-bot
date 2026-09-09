# Changelog

## 1.0.0 — First public release

- Added complete 3x-ui infrastructure administration via Telegram bot written in Go.
- Added interactive FSM wizard for creating VLESS-Reality inbounds with port selection (random 15000–55000, 443, custom), SNI masking presets, X25519 key generation, and short IDs.
- Added client management wizard with UUID generation, traffic limits, expiry dates, and IP limits.
- Added client operations: search by email/UUID, enable/disable toggle, traffic counter reset, expiry extension, and deletion.
- Added real-time server diagnostics: CPU load, RAM usage, disk utilization, uptime, and Xray core status.
- Added SQLite audit logging for administrative actions without CGO dependencies.
- Added automated remote deployment script `deploy.sh` and systemd service unit `admin-bot.service`.
- Added dual-format 3x-ui API client with automatic session re-authentication and cookie persistence.
- Added strict Go-standard English doc comments across the entire codebase.
