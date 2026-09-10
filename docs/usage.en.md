# Usage

[Russian version](usage.md)

## Main Menu and Commands

```text
/start    - interactive main management menu
/inbounds - inbound connections list and management
/servers  - server list and active server switcher
/search   - global client search by email or UUID
/status   - server health monitoring, Xray status, and updates
/logs     - view server diagnostic logs (e.g. /logs 100)
/help     - help reference on bot capabilities
```

## Multi-Server Management

- The `/servers` command or `🌐 Серверы` button displays all registered 3x-ui server endpoints.
- A green checkmark `✅` indicates the currently active server.
- Selecting another server immediately switches the bot's runtime context without restart.
- The default server configuration automatically synchronizes with `.env` on launch.

## Inbound Creation Wizard

1. Enter a clear inbound name (Remark, between 2 and 64 characters).
2. Choose a network port: random available port button (15000–55000), port 443, or custom port input.
3. Select a Reality masking domain: built-in presets (`apple.com`, `yahoo.com`, `gateway.icloud.com`, `dl.google.com`) or custom domain.
4. The bot automatically requests or locally generates an X25519 key pair and a 16-character Short ID.
5. Review the parameters on the confirmation card and click the create button.

## Client Management & Universal Protocols

- **Online status indicators**:
  - `🟢` — client is online (active traffic right now);
  - `⚪` — client is active but currently offline;
  - `⏸` — client is disabled by the administrator.
- **Global search**: `/search` command or `🔍 Поиск клиента` button searches across all inbounds by email or UUID substrings.
- **Protocol support & subscriptions**:
  - Automatically fetches native 3x-ui `subLinks` when available.
  - Universal link generation for VLESS (Reality and TLS), VMess, Trojan, and Shadowsocks.
  - In-memory QR code generation sent directly to Telegram.
- **Client actions**: enable/disable toggle, traffic counter reset, expiry extension, and confirmation-protected deletion.

## System Diagnostics, Logs & Audit

- **Server monitoring (`/status`)**: live CPU, RAM, disk space metrics, uptime, and Xray core status.
- **Update checking**: queries GitHub API for new 3x-ui releases and alerts if an update is available.
- **Log inspection (`/logs`)**: inspects `journalctl -u x-ui` via SSH tunnel or panel API fallback. For outputs exceeding 3,500 characters, the bot automatically uploads a `server-logs.txt` document.
- **Audit logging**: every administrative change is tracked in the local SQLite `audit_logs` table.
