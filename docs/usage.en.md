# Usage

[Russian version](usage.md)

## Main Menu and Commands

```text
/start
/inbounds
/clients
/server
/help
```

- `/start` — interactive main management menu;
- `/inbounds` — inbound connections list and management;
- `/clients` — search and step-by-step client provisioning;
- `/server` — server health monitoring and Xray status;
- `/help` — help reference on bot capabilities.

## Inbound Creation Wizard

1. Enter a clear inbound name (Remark, between 2 and 64 characters).
2. Choose a network port: random available port button (15000–55000), port 443, or custom port input.
3. Select a Reality masking domain: built-in presets (`apple.com`, `yahoo.com`, `gateway.icloud.com`, `dl.google.com`) or custom domain.
4. The bot automatically requests or locally generates an X25519 key pair and a 16-character Short ID.
5. Review the parameters on the confirmation card and click the create button.

## Client Management

- step-by-step client creation in the selected inbound with automatic UUIDv4 generation;
- searching existing clients by email or UUID;
- client card with actions: enable/disable toggle, traffic counter reset, and deletion;
- generating a monospace `vless://` link and sending a QR code image directly to Telegram.

## System Monitoring and Audit

- live server health metrics: CPU, RAM, disk space utilization, and system Uptime;
- checking status of the Xray core service (active/stopped);
- automated recording of every administrative action into the SQLite audit log (`audit_logs`).
