# 3x-ui Telegram Admin Bot

Telegram бот для управления панелью [3x-ui](https://github.com/MHSanaei/3x-ui) через Telegram.

## Возможности

- 📋 **Просмотр Inbound (входящих подключений)**:
  - Просмотр списка активных inbound-подключений и статистики трафика.
  - Управление клиентами в существующих подключениях.
- 👥 **Управление клиентами**:
  - Создание новых клиентов с автоматической генерацией UUID.
  - Получение VLESS-ссылок и QR-кодов для подключения.
  - Просмотр статистики и лимитов трафика.
  - Включение, отключение, сброс трафика и удаление клиентов.
- 🔒 **Безопасность и контроль доступа**:
  - Whitelist Telegram ID администраторов (тихий игнор неавторизованных пользователей).
  - Аудит действий в локальной SQLite базе данных.
- ⚡ **Удобный FSM интерфейс**:
  - Интерактивное inline-меню.
  - Пошаговые сценарии ввода данных с возможностью отмены на любом этапе.

## Стек технологий

- **Go 1.25**
- [telebot.v3](https://github.com/tucnak/telebot) — Telegram Bot API фреймворк
- [modernc.org/sqlite](https://modernc.org/sqlite) — CGo-free SQLite драйвер
- [go-qrcode](https://github.com/skip2/go-qrcode) — генерация QR-кодов

## Установка и запуск

### 1. Клонирование репозитория

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

### 2. Настройка переменных окружения

Скопируйте файл `.env.example` в `.env` и укажите ваши параметры:

```bash
cp .env.example .env
```

Параметры:
- `BOT_TOKEN` — токен Telegram бота от [@BotFather](https://t.me/BotFather)
- `ADMIN_IDS` — Telegram ID администраторов через запятую (например, `123456789,987654321`)
- `XUI_BASE_URL` — URL панели 3x-ui (например, `https://panel.example.com:2053`)
- `XUI_USERNAME` — логин администратора 3x-ui
- `XUI_PASSWORD` — пароль администратора 3x-ui
- `XUI_INSECURE_SKIP_VERIFY` — пропуск проверки TLS-сертификата (`true`/`false`)
- `SERVER_HOST` — IP или домен сервера для VLESS ссылок

### 3. Сборка и запуск

```bash
# Сборка бинарного файла
go build -o admin-bot ./cmd/admin

# Запуск
./admin-bot
```

## Тестирование

```bash
go test ./...
```

## Лицензия

MIT
