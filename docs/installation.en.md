# Installation

[Russian version](installation.md)

## 1. System Requirements

- Linux server running Ubuntu 22.04+, Debian 12+, or RHEL/AlmaLinux 9+;
- installed and configured MHSanaei/3x-ui management panel;
- dedicated Telegram bot token for the administrator from @BotFather;
- SSH access with `root` or `sudo` privileges.

## 2. Installing the 3x-ui Panel

Official installation script for the latest release of the MHSanaei/3x-ui panel:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
```

During installation, specify the web panel port (default 2053), admin username, and password.

## 3. Option 1: Deployment via Docker

Clone the project repository to the server:

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

Create the `.env` configuration file from the template and fill in panel credentials and bot token:

```bash
cp .env.example .env
nano .env
```

Start the bot container with automatic image building:

```bash
docker compose up -d --build
```

View bot runtime logs in real time:

```bash
docker compose logs -f admin-bot
```

## 4. Option 2: Transferring Binary and .env File

Automated build and upload using the `deploy.sh` script:

```bash
chmod +x deploy.sh
./deploy.sh <SERVER_IP>
```

Manual approach: compile a static standalone binary without CGO on the local machine:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o admin-bot ./cmd/admin
```

Upload the binary and `.env` configuration file to the server and set proper permissions:

```bash
ssh root@SERVER_IP "mkdir -p /opt/3x-ui-admin"
scp admin-bot root@SERVER_IP:/opt/3x-ui-admin/admin-bot
scp .env root@SERVER_IP:/opt/3x-ui-admin/.env
ssh root@SERVER_IP "chmod 755 /opt/3x-ui-admin/admin-bot && chmod 600 /opt/3x-ui-admin/.env"
```

Configure the systemd service file `/etc/systemd/system/admin-bot.service`:

```ini
[Unit]
Description=3x-ui Telegram Admin Bot
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/3x-ui-admin
ExecStart=/opt/3x-ui-admin/admin-bot
Restart=always
RestartSec=5s
EnvironmentFile=/opt/3x-ui-admin/.env
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal
SyslogIdentifier=admin-bot

[Install]
WantedBy=multi-user.target
```

Enable and start the service via systemd:

```bash
sudo cp admin-bot.service /etc/systemd/system/admin-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now admin-bot.service
```

## 5. Verifying Status and Logs

Verify that the service is running and processing incoming updates:

```bash
sudo systemctl status admin-bot.service
sudo journalctl -u admin-bot.service -f
```
