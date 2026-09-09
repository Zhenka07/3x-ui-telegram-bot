# Troubleshooting

[Russian version](troubleshooting.md)

## Authorization Error: session expired or 401 Unauthorized

The bot attempts automatic re-login when a session expires. If the failure persists, check service logs:

```bash
journalctl -u admin-bot.service -n 50 --no-pager
```

Verify that credentials specified in `.env` match the active login and password of the 3x-ui panel.

## TLS Error: x509: certificate signed by unknown authority

When using a self-signed SSL certificate for the 3x-ui panel, add the following setting to `.env`:

```dotenv
XUI_INSECURE_SKIP_VERIFY=true
```

After updating the file, restart the service using `systemctl restart admin-bot.service`.

## Telegram Error: 401 Unauthorized / Invalid Token

Verify the validity of the bot token received from @BotFather using a direct HTTP request:

```bash
curl -s "https://api.telegram.org/bot<YOUR_TOKEN>/getMe"
```

The response should return a valid JSON object with `ok: true` and the username of your bot.

## Database Error: database is locked

This error occurs when multiple processes attempt concurrent write access to the same SQLite file:

```bash
sudo systemctl stop admin-bot.service
fuser -v /opt/3x-ui-admin/data/admin.db
```

Terminate lingering background processes and start the service cleanly.

## Permission Error: permission denied data/

Ensure that the service process user has appropriate read and write permissions on the data directory:

```bash
sudo chown -R root:root /opt/3x-ui-admin/data
sudo chmod 700 /opt/3x-ui-admin/data
```
