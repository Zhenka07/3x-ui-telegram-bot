# Карта репозитория

[English version](repository-map.en.md)

| Путь | Назначение |
|---|---|
| `cmd/admin/main.go` | Точка входа Telegram-бота администратора |
| `internal/bot/bot.go` | Инициализация telebot, регистрация меню, middleware и маршрутов |
| `internal/bot/handlers_menu.go` | Главное меню `/start`, `/help` и системный мониторинг `/server` |
| `internal/bot/handlers_inbounds.go` | Просмотр списка inbounds, детальная информация и сброс трафика |
| `internal/bot/handlers_inbound_create.go` | Пошаговый FSM-мастер создания новых VLESS-Reality подключений |
| `internal/bot/handlers_wizard.go` | Пошаговый FSM-мастер добавления новых клиентов в inbounds |
| `internal/bot/handlers_clients.go` | Поиск, просмотр карточки клиента, включение, сброс и удаление |
| `internal/bot/fsm.go` | Хранилище состояний диалогов FSM (сессии создания inbound и клиента) |
| `internal/bot/messages.go` | Текстовые шаблоны сообщений, экранирование HTML и форматирование |
| `internal/bot/middleware.go` | Перехват паник, структурированное логирование и белый список `ADMIN_IDS` |
| `internal/config/` | Загрузка и валидация конфигурации из `.env` и переменных окружения |
| `internal/storage/` | SQLite-репозиторий журнала аудита административных действий (`audit_logs`) |
| `internal/vless/` | Сборщик VLESS URI, генератор QR-кодов в памяти, генератор UUIDv4 и парсер Reality |
| `internal/xui/` | Двухформатный API-клиент панели 3x-ui (сессии, inbounds, клиенты, сертификаты, статус) |
| `docs/` | Полный комплект эксплуатационной и технической документации |
| `admin-bot.service` | Systemd unit для круглосуточной работы бота на сервере |
| `deploy.sh` | Скрипт автоматической сборки и обновления бинарника на удалённом сервере |
| `Dockerfile` | Многоэтапная сборка минимального Docker-образа |
| `docker-compose.yml` | Конфигурация запуска сервиса в Docker Compose |
| `VERSION` | Файл с текущей версией проекта |
| `LICENSE` | Лицензия проекта |
| `CHANGELOG.md` | История изменений и версий проекта |

Файлы `.env`, базы данных SQLite `*.db` и скомпилированные бинарники исключены из Git.
