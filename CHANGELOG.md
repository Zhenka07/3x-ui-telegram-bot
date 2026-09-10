# Changelog

## 1.1.0 — Multi-server, SSH Tunneling, Protocol Expansion & Diagnostics

- **Multi-server management**: Added support for managing multiple 3x-ui servers via SQLite `servers` table and Telegram switcher (`/servers`).
- **SSH Port Forwarding tunnel**: Added secure in-memory TCP tunneling (`internal/sshtunnel`) via `golang.org/x/crypto/ssh` to connect to panels on private networks without exposing panel ports.
- **Universal protocol links & subLinks**: Added automatic fetching of native 3x-ui `subLinks` and universal link generation for VLESS (Reality & TLS), VMess (base64 JSON), Trojan, and Shadowsocks.
- **Real-time online status indicators**: Integrated `/panel/api/clients/onlines` with visual status badges (`🟢` online, `⚪` offline, `⏸` disabled) across client buttons, inbound lists, and client cards.
- **Global client search**: Added instant search across all inbounds by email or UUID fragment with direct navigation (`/search`, `🔍 Поиск клиента`).
- **Server logs inspection**: Added `/logs` command and `📋 Логи сервера` button to inspect `journalctl -u x-ui` / `/var/log/x-ui.log` via SSH with API fallback and document attachment for large logs.
- **Release update checker**: Added GitHub API version comparator (`internal/updater`) with 1-hour cache to notify about new 3x-ui releases in `/status`.
- **Enhanced test suite**: Added unit tests for multi-server repository, version comparator, universal links, and logs retrieval.

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
