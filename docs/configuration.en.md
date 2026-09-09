# Configuration

[Russian version](configuration.md)

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `BOT_TOKEN` | Telegram bot token from @BotFather | required |
| `ADMIN_IDS` | Comma-separated list of administrator Telegram IDs | required |
| `XUI_BASE_URL` | Full base URL of the 3x-ui web panel | required |
| `XUI_USERNAME` | Administrator username for 3x-ui authentication | `admin` |
| `XUI_PASSWORD` | Administrator password for 3x-ui authentication | required |
| `XUI_INSECURE_SKIP_VERIFY` | Disable TLS certificate verification (for self-signed certs) | `false` |
| `SERVER_HOST` | Public IP address or domain name for VLESS links | required |
| `DB_PATH` | Path to the SQLite audit database file | `data/admin.db` |
| `XUI_TIMEOUT` | HTTP request timeout for 3x-ui panel API | `20s` |
| `TELEGRAM_POLLER_TIMEOUT` | Telegram long polling timeout | `10s` |
| `LOG_LEVEL` | Application logging verbosity (`debug`, `info`, `warn`, `error`) | `info` |

## Configuration File Security

- `.env` files contain sensitive secrets and must be restricted to mode `0600`;
- database directory `data/` is created automatically by the app with mode `0700`;
- never share Telegram bot tokens or 3x-ui panel credentials over unencrypted channels.
