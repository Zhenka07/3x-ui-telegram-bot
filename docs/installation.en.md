# Installation

[Russian version](installation.md)

## 1. System Requirements

- Linux server running Ubuntu 22.04+, Debian 12+, or RHEL/AlmaLinux 9+;
- an installed and running MHSanaei/3x-ui panel instance;
- Go 1.22+ compiler or installed Docker Compose;
- dedicated Telegram bot token for the administrator from @BotFather;
- SSH access with `root` or `sudo` privileges.

## 2. Installing the 3x-ui Panel

Official installation script for the latest stable release of MHSanaei/3x-ui:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
```

During installation, configure the web panel port (default 2053), admin username, and password.

## 3. Obtaining the Source Code

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

## 4. Quick Deployment and Updates via deploy.sh

The `deploy.sh` script compiles the Linux binary, uploads it to the server over SSH, and restarts the service:

```bash
chmod +x deploy.sh
./deploy.sh <SERVER_IP>
```

The script verifies `.env` presence, atomically replaces the executable file, and checks service status.

## 5. Deployment with Docker

To deploy using Docker Compose, copy `.env.example` to `.env`, configure parameters, and launch the container:

```bash
cp .env.example .env
docker compose up -d --build
```

View container logs in real time:

```bash
docker compose logs -f admin-bot
```

## 6. Manual Build and Systemd Setup

Build a static standalone binary without CGO:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o admin-bot ./cmd/admin
```

Create the working directory and copy required files:

```bash
sudo mkdir -p /opt/3x-ui-admin
sudo cp admin-bot /opt/3x-ui-admin/admin-bot
sudo chmod 755 /opt/3x-ui-admin/admin-bot
sudo cp .env.example /opt/3x-ui-admin/.env
sudo chmod 600 /opt/3x-ui-admin/.env
```

Configure the `/etc/systemd/system/admin-bot.service` file:

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

Enable and start the service:

```bash
sudo cp admin-bot.service /etc/systemd/system/admin-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now admin-bot.service
```

## 7. Verifying Status and Logs

Verify that the service is running and processing incoming updates:

```bash
sudo systemctl status admin-bot.service
sudo journalctl -u admin-bot.service -f
```
