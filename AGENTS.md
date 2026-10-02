# 3x-ui Telegram Admin Bot — Инструкция для AI-агентов (AGENTS.md)

Добро пожаловать в проект **3x-ui Telegram Admin Bot**!
Этот документ содержит архитектурные принципы, правила написания кода, стандарты безопасности и рабочие процессы для AI-агентов и ассистентов в текущих и будущих сессиях.

---

## 1. Роль и назначение проекта

Проект представляет собой производственный Telegram-бот на Go для полноценного удалённого администрирования серверов и VPN-панели [3x-ui](https://github.com/MHSanaei/3x-ui) (протоколы Xray: VLESS-Reality, VMess, Trojan, Shadowsocks).

Ключевые возможности системы:
- **Мультисерверность** (`/servers`): подключение и переключение между несколькими панелями 3x-ui прямо из чата.
- **Безопасный доступ** (`internal/sshtunnel`): работа через SSH-туннель без необходимости открывать порт веб-панели наружу.
- **Управление подключениями**: создание Inbounds с Reality-маскировкой, генерация ключей X25519, создание клиентов, генерация VLESS/VMess/Trojan ссылок и QR-кодов.
- **Мониторинг и логирование**: отслеживание онлайн-статусов клиентов, просмотр системных логов x-ui и Xray в чате (`/logs`).
- **Аудит**: сохранение всех действий администратора в SQLite без CGO.
- **Observability**: сквозное структурированное логирование (`log/slog`) и распределённый трейсинг (`OpenTelemetry`).

---

## 2. Стек технологий и обязательные ограничения

1. **Версия Go**: Go 1.22+ (актуальная версия среды разработки — Go 1.26).
2. **CGO**: **СТРОГО ЗАПРЕЩЁН** (`CGO_ENABLED=0`).
   - Используется pure-Go драйвер SQLite: `modernc.org/sqlite`.
   - Любая компиляция и запуск тестов производятся исключительно с `CGO_ENABLED=0`.
3. **Логирование**: Стандартная библиотека `log/slog` (никаких logrus/zap/zerolog).
   - Используется `observability.TracingHandler`.
   - При логировании обязательно передавать контекст: `slog.InfoContext(ctx, ...)` или логгер с контекстом.
   - Поддерживаются форматы вывода `json` и `text`, уровни `debug`, `info`, `warn`, `error`.
4. **Распределённый трейсинг**: Официальный SDK OpenTelemetry (`go.opentelemetry.io/otel`).
   - Экспорт спанов в `stdout` или по gRPC в OTLP коллектор (Jaeger / Tempo).
   - Поддержка полного отключения трейсинга через флаг `OTEL_ENABLED=false` с no-op провайдером.
   - Корректная обработка завершения (`Shutdown(ctx)`) с таймаутом сброса буфера телеметрии.
5. **Безопасность (PII & Secrets)**:
   - **СТРОГО ЗАПРЕЩЕНО** выводить в логи или спаны в открытом виде токены бота (`BOT_TOKEN`), пароли от панели (`XUI_PASSWORD`), SSH-ключи, пароли и сессионные куки.
   - Чувствительные поля автоматически маскируются значением `[REDACTED]`.
6. **Telegram-библиотека**: `gopkg.in/telebot.v3`.

---

## 3. Архитектура пакетов

```text
admin/
├── cmd/
│   └── admin/
│       └── main.go                 # Точка входа: graceful shutdown, OTel, slog, bot.Start
├── internal/
│   ├── bot/                        # Telegram-бот на telebot.v3
│   │   ├── bot.go                  # Инициализация бота, регистрация роутов
│   │   ├── middleware.go           # Whitelist администраторов, трейсинг, panic recovery
│   │   ├── fsm.go                  # Машина состояний (FSM) для пошаговых сценариев
│   │   ├── handlers_*.go           # Хендлеры меню, клиентов, серверов, логов, мастера inbounds
│   │   └── messages.go             # Шаблоны сообщений и клавиатуры
│   ├── config/                     # Парсинг переменных окружения (.env) и валидация
│   ├── observability/              # Логирование и OpenTelemetry трейсинг
│   │   ├── config.go               # Конфигурация уровней логов и OTel параметров
│   │   ├── context.go              # Извлечение TraceID/SpanID, мост с telebot.Context
│   │   ├── logger.go               # TracingHandler с санитизацией секретов
│   │   ├── middleware.go           # TelegramTraceMiddleware: создание Root Span
│   │   └── tracer.go               # InitTracer (stdout, otlp, disabled)
│   ├── sshtunnel/                  # SSH-туннель (DialContext, ExecuteCommand с OTel спанами)
│   ├── storage/                    # SQLite: audit_logs, servers (инструментированы OTel)
│   ├── updater/                    # Проверка релизов панели 3x-ui на GitHub
│   ├── vless/                      # Генерация universal link (VLESS/VMess/Trojan), QR-кодов, UUID
│   └── xui/                        # Клиент к 3x-ui API (otelhttp.NewTransport, авто-перелогин)
├── docs/                           # Подробная документация проекта
└── .env.example                    # Пример конфигурации со всеми переменными
```

---

## 4. Паттерны и правила реализации

### 4.1. Контекст и трейсинг в Telegram-хендлерах
- `telebot.Context` связывается с `context.Context` через `observability.TelegramTraceMiddleware`.
- Для получения контекста с Root Span внутри хендлера используйте:
  ```go
  ctx := observability.ContextFromTele(c)
  ```
- Для фоновых асинхронных операций (`go func()`) отвязывайте контекст от отмены запроса Telegram:
  ```go
  asyncCtx := context.WithoutCancel(ctx)
  go func() {
      asyncCtx, span := tracer.Start(asyncCtx, "operation_name")
      defer span.End()
      // работа с asyncCtx
  }()
  ```

### 4.2. Dual Error Recording в OpenTelemetry
При возникновении ошибки внутри спана всегда вызывайте оба метода:
```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
```

### 4.3. HTTP-клиенты и внешние вызовы
- Все исходящие HTTP-запросы должны выполняться с контекстом: `http.NewRequestWithContext(ctx, ...)`.
- Все HTTP-клиенты к внешним API оборачиваются через `otelhttp.NewTransport(transport)`.

### 4.4. Хранилище SQLite
- SQLite настраивается с параметрами:
  `journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`, `MaxOpenConns(1)`.
- Все методы репозиториев принимают `context.Context` первым аргументом и создают дочерние спаны (`db.<repo>.<op>`).

---

## 5. Процедура тестирования и сдачи задач

Перед завершением любой задачи агент **обязан** выполнить верификацию:

```bash
# 1. Запуск всех юнит-тестов (строго без CGO)
CGO_ENABLED=0 go test -v ./...

# 2. Проверка успешной сборки бинарника
CGO_ENABLED=0 go build -o /dev/null -v ./cmd/admin
```

### Правила коммитов и веток:
- Ветки: `feat/<feature-name>`, `fix/<bug-name>`, `docs/<doc-name>`.
- Формат сообщений: **Conventional Commits**:
  - `feat(scope): краткое описание на английском`
  - `fix(scope): краткое описание`
  - `docs(scope): краткое описание`
- Никогда не коммитить: `.env`, скомпилированные бинарники (`admin-bot`), файлы БД (`admin.db`, `data/`), дампы и логи.
