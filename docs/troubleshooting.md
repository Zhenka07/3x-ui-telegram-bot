# Решение проблем

[English version](troubleshooting.en.md)

## Ошибка авторизации: session expired или 401 Unauthorized

Бот автоматически переавторизуется при устаревании сессии. Если ошибка повторяется, проверьте логи службы:

```bash
journalctl -u admin-bot.service -n 50 --no-pager
```

Убедитесь, что логин и пароль в файле `.env` совпадают с учётными данными веб-панели 3x-ui.

## Ошибка TLS: x509: certificate signed by unknown authority

При использовании самоподписанного SSL-сертификата в панели 3x-ui добавьте в файл `.env`:

```dotenv
XUI_INSECURE_SKIP_VERIFY=true
```

После изменения файла перезапустите службу командой `systemctl restart admin-bot.service`.

## Ошибка Telegram: 401 Unauthorized / Invalid Token

Проверьте корректность токена бота, отправленного @BotFather, с помощью прямого запроса:

```bash
curl -s "https://api.telegram.org/bot<YOUR_TOKEN>/getMe"
```

В ответе должен вернуться JSON со статусом `ok: true` и именем созданного бота.

## Ошибка базы данных: database is locked

Ошибка возникает при одновременной блокировке файла SQLite несколькими процессами:

```bash
sudo systemctl stop admin-bot.service
fuser -v /opt/3x-ui-admin/data/admin.db
```

Завершите зависшие фоновые процессы и запустите службу заново.

## Ошибка прав доступа: permission denied data/

Убедитесь, что пользователь процесса имеет права на чтение и запись в каталоге базы данных:

```bash
sudo chown -R root:root /opt/3x-ui-admin/data
sudo chmod 700 /opt/3x-ui-admin/data
```
