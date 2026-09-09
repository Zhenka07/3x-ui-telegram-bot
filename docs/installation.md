# Установка

[English version](installation.en.md)

## 1. Системные требования

- Linux-сервер под управлением Ubuntu 22.04+, Debian 12+ или RHEL/AlmaLinux 9+;
- установленная панель управления MHSanaei/3x-ui;
- токен отдельного Telegram-бота для администратора от @BotFather;
- доступ по SSH с правами `root` или `sudo`.

## 2. Установка панели 3x-ui

Официальный скрипт установки актуальной версии панели MHSanaei/3x-ui:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
```

В процессе установки укажите порт веб-панели (по умолчанию 2053), имя пользователя и пароль.

## 3. Вариант 1: Развёртывание через Docker

Клонируйте репозиторий проекта на сервер:

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

Создайте файл конфигурации `.env` на основе шаблона и заполните параметры панели и токен:

```bash
cp .env.example .env
nano .env
```

Запустите контейнер бота с автоматической сборкой образа:

```bash
docker compose up -d --build
```

Просмотр журнала работы бота в реальном времени:

```bash
docker compose logs -f admin-bot
```

## 4. Вариант 2: Перенос бинарника и файла .env

Автоматическая сборка и загрузка через скрипт `deploy.sh`:

```bash
chmod +x deploy.sh
./deploy.sh <SERVER_IP>
```

Ручной способ: соберите статический бинарник без CGO на локальном компьютере:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o admin-bot ./cmd/admin
```

Загрузите бинарник и файл конфигурации `.env` на сервер и выставьте права доступа:

```bash
ssh root@SERVER_IP "mkdir -p /opt/3x-ui-admin"
scp admin-bot root@SERVER_IP:/opt/3x-ui-admin/admin-bot
scp .env root@SERVER_IP:/opt/3x-ui-admin/.env
ssh root@SERVER_IP "chmod 755 /opt/3x-ui-admin/admin-bot && chmod 600 /opt/3x-ui-admin/.env"
```

Настройте файл службы `/etc/systemd/system/admin-bot.service`:

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

Активируйте и запустите службу через systemd:

```bash
sudo cp admin-bot.service /etc/systemd/system/admin-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now admin-bot.service
```

## 5. Проверка статуса и журналов

Убедитесь, что служба успешно запущена и обрабатывает входящие обновления:

```bash
sudo systemctl status admin-bot.service
sudo journalctl -u admin-bot.service -f
```
