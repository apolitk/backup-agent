# Backup Agent for VK Cloud S3

Агент резервного копирования для VK Cloud S3, разработанный на Go.

## Требования

- Go 1.21+
- Docker и Docker Compose (для локального тестирования)

## Сборка

```bash
make build
```

## Запуск локально

# Запуск MinIO и агента через Docker Compose
make docker-up

# Проверка здоровья сервиса
curl http://localhost:8080/health

## API

### Health Check
```bash
curl http://localhost:8080/health
```

### Создание бэкапа
```bash
curl -X POST http://localhost:8080/api/v1/backup \
  -H "Authorization: Bearer test-token-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/tmp/test_data",
    "s3_key": "backups/test-backup.tar.gz"
  }'
```

### Проверка статуса
```bash
curl -X GET http://localhost:8080/api/v1/task/TASK_ID \
  -H "Authorization: Bearer test-token-secret-123"
```

### Список бэкапов
```bash
curl -X GET "http://localhost:8080/api/v1/backups?prefix=backups/" \
  -H "Authorization: Bearer test-token-secret-123"
```

### Восстановление
```bash
curl -X POST http://localhost:8080/api/v1/restore \
  -H "Authorization: Bearer test-token-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "s3_key": "backups/test-backup.tar.gz",
    "destination_path": "/tmp/restored_data"
  }'
```

### Удаление бэкапа
```bash
curl -X DELETE http://localhost:8080/api/v1/backups/backups/test-backup.tar.gz \
  -H "Authorization: Bearer test-token-secret-123"
```

## Конфигурация

Агент конфигурируется через переменные окружения:

- `AGENT_PORT` - Порт сервера (default: 8080)
- `AGENT_TOKEN` - API токен для аутентификации
- `AGENT_TLS_CERT` - Путь к TLS сертификату
- `AGENT_TLS_KEY` - Путь к TLS ключу
- `S3_ENDPOINT` - S3 endpoint
- `S3_REGION` - S3 регион
- `S3_ACCESS_KEY` - S3 access key
- `S3_SECRET_KEY` - S3 secret key
- `S3_USE_SSL` - Использовать SSL (default: true)
- `S3_BUCKET` - S3 бакет
- `TEMP_DIR` - Временная директория (default: /tmp)
- `MAX_WORKERS` - Максимум воркеров (default: 4)
- `LOG_LEVEL` - Уровень логирования (default: info)

## Тестирование

```bash
make test
```

## Установка и запуск

# 1. Клонируем репозиторий
git clone <repo-url>
cd agent

# 2. Инициализируем модули
go mod download

# 3. Запускаем Docker Compose
docker-compose up -d

# 4. Ждём готовности сервисов
sleep 5

# 5. Создаём тестовые данные и выполняем тесты
mkdir -p test_data
echo "test" > test_data/file.txt

# 6. Запускаем тест API
chmod +x scripts/test_api.sh
./scripts/test_api.sh

# 7. Останавливаем сервисы
docker-compose down