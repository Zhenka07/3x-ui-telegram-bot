# Установка

[English version](installation.en.md)

## 1. Системные требования

- Linux-сервер под управлением Ubuntu 22.04+, Debian 12+ или RHEL/AlmaLinux 9+;
- установленная панель управления MHSanaei/3x-ui;
- компилятор Go 1.22+ либо установленный Docker Compose;
- токен отдельного Telegram-бота для администратора от @BotFather;
- доступ по SSH с правами `root` или `sudo`.

## 2. Установка панели 3x-ui

Официальный скрипт установки актуальной версии панели MHSanaei/3x-ui:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
```

В процессе установки укажите порт веб-панели (по умолчанию 2053), имя пользователя и пароль.

## 3. Получение исходного кода

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

## 4. Быстрый деплой и обновление через deploy.sh

Скрипт `deploy.sh` компилирует бинарник под Linux, загружает его на сервер по SSH и перезапускает службу:

```bash
chmod +x deploy.sh
./deploy.sh <SERVER_IP>
```

Скрипт проверяет наличие `.env`, атомарно заменяет исполняемый файл и проверяет статус службы.

## 5. Развёртывание с помощью Docker

Для запуска через Docker Compose скопируйте `.env.example` в `.env`, укажите параметры и запустите контейнер:

```bash
cp .env.example .env
docker compose up -d --build
```

Просмотр журналов контейнера:

```bash
docker compose logs -f admin-bot
```

## 6. Ручная сборка и настройка Systemd

Сборка статического бинарника без CGO:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o admin-bot ./cmd/admin
```

Создайте рабочий каталог и скопируйте файлы:

```bash
sudo mkdir -p /opt/3x-ui-admin
sudo cp admin-bot /opt/3x-ui-admin/admin-bot
sudo chmod 755 /opt/3x-ui-admin/admin-bot
sudo cp .env.example /opt/3x-ui-admin/.env
sudo chmod 600 /opt/3x-ui-admin/.env
```

Настройте файл `/etc/systemd/system/admin-bot.service`:

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

Активируйте и запустите службу:

```bash
sudo cp admin-bot.service /etc/systemd/system/admin-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now admin-bot.service
```

## 7. Проверка статуса и журналов

Убедитесь, что служба успешно запущена и обрабатывает входящие обновления:

```bash
sudo systemctl status admin-bot.service
sudo journalctl -u admin-bot.service -f
```
