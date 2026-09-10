# Карта репозитория

[English version](repository-map.en.md)

| Путь | Назначение |
|---|---|
| `cmd/admin/main.go` | Точка входа Telegram-бота администратора |
| `internal/bot/bot.go` | Инициализация telebot, регистрация меню, middleware и маршрутов |
| `internal/bot/handlers_menu.go` | Главное меню `/start`, `/help`, поиск и системный статус `/status` |
| `internal/bot/handlers_inbounds.go` | Просмотр списка inbounds, онлайн-статусы, детальная информация |
| `internal/bot/handlers_inbound_create.go` | Пошаговый FSM-мастер создания новых VLESS-Reality подключений |
| `internal/bot/handlers_wizard.go` | Пошаговый FSM-мастер добавления новых клиентов в inbounds |
| `internal/bot/handlers_clients.go` | Карточка клиента, генерация universal/subLinks, сброс и удаление |
| `internal/bot/handlers_servers.go` | Список управляемых серверов и переключение активного сервера (`/servers`) |
| `internal/bot/handlers_logs.go` | Просмотр и выгрузка системных логов x-ui и Xray (`/logs`) |
| `internal/bot/fsm.go` | Хранилище состояний диалогов FSM (сессии создания inbound и клиента) |
| `internal/bot/messages.go` | Текстовые шаблоны сообщений, экранирование HTML и форматирование |
| `internal/bot/middleware.go` | Перехват паник, структурированное логирование и белый список `ADMIN_IDS` |
| `internal/config/` | Загрузка и валидация конфигурации из `.env` (Telegram, 3x-ui, SSH, DB) |
| `internal/sshtunnel/` | Модуль безопасного in-memory SSH-туннелирования и выполнения команд |
| `internal/storage/` | SQLite-репозитории журнала аудита (`audit_logs`) и реестра серверов (`servers`) |
| `internal/updater/` | Фоновая проверка и кэширование релизов 3x-ui через GitHub API |
| `internal/vless/` | Генератор универсальных ссылок (VLESS/VMess/Trojan/SS), QR-кодов, UUIDv4 |
| `internal/xui/` | Двухформатный API-клиент панели 3x-ui (сессии, inbounds, clients, certs, logs) |
| `docs/` | Полный комплект эксплуатационной и технической документации |
| `admin-bot.service` | Systemd unit для круглосуточной работы бота на сервере |
| `deploy.sh` | Скрипт автоматической сборки и обновления бинарника на удалённом сервере |
| `Dockerfile` | Многоэтапная сборка минимального Docker-образа |
| `docker-compose.yml` | Конфигурация запуска сервиса в Docker Compose |
| `VERSION` | Файл с текущей версией проекта |
| `LICENSE` | Лицензия проекта |
| `CHANGELOG.md` | История изменений и версий проекта |

Файлы `.env`, базы данных SQLite `*.db` и скомпилированные бинарники исключены из Git.
