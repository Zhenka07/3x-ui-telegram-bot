#!/usr/bin/env bash
set -e

# Цвета для вывода в терминал
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=== Скрипт сборки и обновления 3x-ui Admin Bot ===${NC}"

# 1. Определение параметров сервера
SERVER_IP="${1:-$SERVER_IP}"
SERVER_USER="${SERVER_USER:-root}"
SERVER_PORT="${SERVER_PORT:-22}"
REMOTE_DIR="${REMOTE_DIR:-/opt/3x-ui-admin}"
SERVICE_NAME="${SERVICE_NAME:-admin-bot}"

if [ -z "$SERVER_IP" ]; then
    # Если в локальном .env указан SERVER_HOST, берем его как значение по умолчанию
    if [ -f .env ]; then
        SERVER_IP=$(grep -E "^SERVER_HOST=" .env | cut -d '=' -f2 | tr -d ' "' || true)
    fi
fi

if [ -z "$SERVER_IP" ]; then
    echo -e "${YELLOW}Укажите IP-адрес сервера:${NC}"
    echo -e "Использование: $0 <SERVER_IP>"
    echo -e "Или задайте переменную окружения: SERVER_IP=1.2.3.4 $0"
    exit 1
fi

echo -e "Сервер:       ${GREEN}${SERVER_USER}@${SERVER_IP}:${SERVER_PORT}${NC}"
echo -e "Каталог:      ${GREEN}${REMOTE_DIR}${NC}"
echo -e "Служба:       ${GREEN}${SERVICE_NAME}${NC}"
echo ""

# 2. Локальная компиляция под Linux (amd64) без CGO
LOCAL_BIN="admin-bot-bin"
echo -e "${BLUE}[1/5] Сборка бинарного файла для Linux (amd64)...${NC}"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${LOCAL_BIN}" ./cmd/admin

# Убедимся, что временный бинарник удалится при завершении скрипта
trap 'rm -f "${LOCAL_BIN}"' EXIT

echo -e "${GREEN}✓ Бинарник собран: $(du -h ${LOCAL_BIN} | cut -f1)${NC}"

# 3. Проверка и создание удаленной директории
echo -e "${BLUE}[2/5] Подготовка удаленной директории на сервере...${NC}"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_IP}" "mkdir -p ${REMOTE_DIR}"

# 4. Проверка наличия .env на сервере
echo -e "${BLUE}[3/5] Проверка конфигурационного файла .env на сервере...${NC}"
if ! ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_IP}" "[ -f ${REMOTE_DIR}/.env ]"; then
    echo -e "${YELLOW}⚠️ Внимание: Файл ${REMOTE_DIR}/.env не найден на сервере!${NC}"
    if [ -f .env ]; then
        read -p "Хотите загрузить локальный файл .env на сервер? (y/n): " upload_env
        if [ "$upload_env" = "y" ] || [ "$upload_env" = "Y" ]; then
            scp -P "${SERVER_PORT}" .env "${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/.env"
            ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_IP}" "chmod 600 ${REMOTE_DIR}/.env"
            echo -e "${GREEN}✓ Файл .env успешно загружен с правами 600.${NC}"
        fi
    else
        echo -e "${RED}Не забудьте создать ${REMOTE_DIR}/.env на сервере на основе .env.example перед запуском!${NC}"
    fi
else
    echo -e "${GREEN}✓ Конфигурационный файл ${REMOTE_DIR}/.env присутствует.${NC}"
fi

# 5. Загрузка бинарника на сервер (атомарное обновление)
echo -e "${BLUE}[4/5] Загрузка нового бинарника на сервер...${NC}"
scp -P "${SERVER_PORT}" "${LOCAL_BIN}" "${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/admin-bot.new"

# 6. Замена бинарника и перезапуск systemd-сервиса
echo -e "${BLUE}[5/5] Атомарная замена бинарника и перезапуск сервиса...${NC}"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_IP}" "
    mv ${REMOTE_DIR}/admin-bot.new ${REMOTE_DIR}/admin-bot && \
    chmod +x ${REMOTE_DIR}/admin-bot && \
    if systemctl is-active --quiet ${SERVICE_NAME}; then
        echo 'Перезапуск ${SERVICE_NAME}...' && \
        systemctl restart ${SERVICE_NAME};
    else
        echo 'Запуск ${SERVICE_NAME}...' && \
        systemctl restart ${SERVICE_NAME} || true;
    fi && \
    systemctl is-active --quiet ${SERVICE_NAME} && echo 'Служба работает корректно!' || echo 'Внимание: проверьте логи через journalctl -u ${SERVICE_NAME} -n 20'
"

echo ""
echo -e "${GREEN}🎉 Бот успешно обновлен и работает на сервере!${NC}"
echo -e "Просмотр логов в реальном времени: ${YELLOW}ssh ${SERVER_USER}@${SERVER_IP} 'journalctl -u ${SERVICE_NAME} -f'${NC}"
