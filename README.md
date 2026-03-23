# Backup Agent for VK Cloud S3

Агент резервного копирования для VK Cloud S3, разработанный на Go.

## Зачем нужен Backup Agent?

Backup Agent решает задачу автоматизации резервного копирования данных в облачное хранилище VK Cloud S3. Основные преимущества:

- **Автоматизация**: Программное управление бэкапами без ручного вмешательства
- **Надежность**: Асинхронная обработка с отслеживанием статуса операций
- **Масштабируемость**: Пул воркеров для параллельной обработки нескольких задач
- **Интеграция**: Полная совместимость с VK Cloud S3 через стандартный S3 API
- **Безопасность**: Аутентификация через токены, поддержка TLS
- **Мониторинг**: Детальное логирование и health checks

## Как работает Backup Agent?

### Архитектура

Агент построен по принципам чистой архитектуры с разделением на слои:

```
HTTP Handlers (API) → Task Manager → Worker Pool → Storage (S3) + Archiver
```

### Основные компоненты:

1. **HTTP API**: RESTful интерфейс для управления бэкапами
2. **Task Manager**: Управление жизненным циклом задач (создание, отслеживание, очистка)
3. **Worker Pool**: Пул горутин для асинхронной обработки задач
4. **S3 Storage**: Клиент для работы с VK Cloud S3 (на базе MinIO)
5. **Archiver**: Создание/распаковка tar.gz архивов
6. **Logger**: Структурированное логирование с уровнями

### Поток операций бэкапа:

1. Получение запроса через HTTP API
2. Создание задачи в Task Manager
3. Передача в Worker Pool для асинхронной обработки
4. Архивация исходных данных в tar.gz
5. Загрузка архива в VK Cloud S3
6. Обновление статуса задачи

### Поток операций восстановления:

1. Получение запроса на восстановление
2. Скачивание архива из S3
3. Распаковка в целевую директорию
4. Обновление статуса задачи

## Переход на реальное S3‑хранилище VK Cloud

Для перехода на production с VK Cloud S3:

1. **Замените переменные окружения агента** в `docker-compose.yml`:
   ```yaml
   S3_ENDPOINT: "hb.ru-msk.vkcloud-storage.ru"
   S3_REGION: "ru-msk"
   S3_ACCESS_KEY: "ваш_access_key"
   S3_SECRET_KEY: "ваш_secret_key"
   S3_USE_SSL: "true"
   S3_BUCKET: "название_вашего_бакета"
   ```

2. **Перезапустите контейнеры**:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

## Развертывание на VM/Компьютере

Backup Agent можно развернуть на любой Linux-совместимой VM или компьютере:

### Системные требования:

- Linux (Ubuntu, CentOS, Debian и т.д.)
- Go 1.21+ (для сборки) или готовый бинарный файл
- Доступ к VK Cloud S3 (credentials)

### Шаги развертывания:

1. **Сборка приложения:**
   ```bash
   git clone <repository-url>
   cd backup-agent
   go mod download
   CGO_ENABLED=0 GOOS=linux go build -o agent ./cmd/agent
   ```

2. **Настройка переменных окружения:**
   ```bash
   export AGENT_PORT=8080
   export AGENT_TOKEN="your-secure-token"
   export S3_ENDPOINT="https://hb.bizmrg.com"
   export S3_REGION="ru-msk"
   export S3_ACCESS_KEY="your-access-key"
   export S3_SECRET_KEY="your-secret-key"
   export S3_BUCKET="your-bucket-name"
   export LOG_LEVEL="info"
   ```

3. **Запуск:**
   ```bash
   ./agent
   ```

### Docker развертывание:

```bash
docker build -t backup-agent .
docker run -p 8080:8080 \
  -e AGENT_TOKEN="your-token" \
  -e S3_ENDPOINT="https://hb.bizmrg.com" \
  -e S3_ACCESS_KEY="your-key" \
  -e S3_SECRET_KEY="your-secret" \
  backup-agent
```

## Интеграция с Karboii

Backup Agent разработан для интеграции с сервисом резервного копирования Karboii VK Cloud. Karboii предоставляет высокоуровневое API для управления бэкапами, а агент выполняет фактические операции с данными.

### Настройка интеграции в Karboii

1. **Разверните агента** на VM в VK Cloud или Codespaces
2. **Получите URL агента** (в Codespaces - через вкладку Ports, порт 8080)
3. **В Karboii настройте адрес агента** как endpoint для выполнения операций:
   - В интерфейсе Karboii укажите URL агента (например: `http://your-agent-url:8080`)
   - Настройте токен аутентификации: `test-token-secret-123`
4. **Создайте план бэкапа** через Karboii API или интерфейс
5. **Настройте триггеры** для автоматического запуска бэкапов

### Тестирование интеграции

Для тестирования интеграции можно использовать curl команды:

```bash
# Проверка доступности агента
curl http://localhost:8080/health

# Создание бэкапа (имитация вызова от Karboii)
curl -X POST http://localhost:8080/api/v1/backup \
  -H "Authorization: Bearer test-token-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/data",
    "s3_key": "backup-test.tar.gz"
  }'
```

## Архитектура интеграции с Karboii

### Идеальная картина мира

1. **Karboii управляет планами и стратегиями**: Пользователь через Karboii задает планы резервного копирования, триггеры и стратегии хранения
2. **Karboii вызывает агента для выполнения**: Агент получает конкретные задания на бэкап/восстановление
3. **Агент выполняет операции**: Создает архивы и загружает в S3 VK Cloud
4. **Karboii управляет жизненным циклом**: Удаляет старые бэкапы согласно стратегиям хранения

### Текущая реализация

**✅ Что работает:**
- Karboii может вызывать REST API агента для выполнения операций
- Агент асинхронно выполняет бэкап (архивирование + загрузка в S3)
- Поддержка всех CRUD операций: backup, restore, list, delete
- Асинхронное выполнение с отслеживанием статуса задач

**⚠️ Архитектурные решения:**
- **Агент как "executor"**: Не хранит логику планов/триггеров, только выполняет операции
- **Karboii как "orchestrator"**: Должен управлять всем жизненным циклом бэкапов
- **Разделение ответственности**: Karboii = мозг (планирование), Агент = руки (выполнение)

### Сценарий работы

```
1. Пользователь создает план в Karboii
   POST /{project_id}/plans

2. Karboii настраивает триггер
   POST /{project_id}/triggers

3. По расписанию Karboii вызывает агента
   POST /api/v1/backup (source_path, s3_key)

4. Агент выполняет бэкап асинхронно
   - Создает task
   - Архивирует данные
   - Загружает в S3
   - Обновляет статус

5. Karboii проверяет статус
   GET /api/v1/task/{task_id}

6. Karboii управляет хранением
   - Анализирует стратегии (GFS, retention)
   - Вызывает удаление старых бэкапов
   DELETE /api/v1/backups/{old_key}
```

### Рекомендации для production

- **Валидация запросов**: Добавить проверку, что запросы приходят от Karboii
- **Аутентификация**: Настроить взаимную аутентификацию (Keystone tokens ↔ Bearer tokens)
- **Мониторинг**: Логировать все операции для аудита
- **Отказоустойчивость**: Обработка сетевых ошибок и повторные попытки
- **Масштабирование**: Несколько инстансов агента за load balancer

### Масштабирование и высокая доступность

Karboii вызывает API агента для выполнения операций:

- **Создание бэкапа**: Karboii → POST /api/v1/backup → Агент архивирует и загружает в S3
- **Восстановление**: Karboii → POST /api/v1/restore → Агент скачивает и распаковывает
- **Список бэкапов**: Karboii → GET /api/v1/backups → Агент возвращает список из S3
- **Удаление**: Karboii → DELETE /api/v1/backups/{key} → Агент удаляет из S3

### API Karboii

Karboii предоставляет REST API для управления планами бэкапов, триггерами и чекпоинтами. Агент интегрируется как исполнитель операций с данными.

Основные endpoints Karboii:
- `/{project_id}/plans` - Управление планами бэкапов
- `/{project_id}/checkpoints` - Создание и управление чекпоинтами
- `/{project_id}/restores` - Восстановление из бэкапов

### Сценарий интеграции с Karboii

1. **Создание плана в Karboii**:
   - POST `/{project_id}/plans` - создает план бэкапа
   - В плане указываются ресурсы (VM, диски) и параметры

2. **Настройка триггера**:
   - POST `/{project_id}/triggers` - создает расписание бэкапов
   - Указывается cron-выражение и максимальное количество бэкапов

3. **Выполнение бэкапа**:
   - Karboii вызывает POST `/{project_id}/providers/{provider_id}/checkpoints`
   - Агент получает задачу и выполняет бэкап через свой API
   - Результат возвращается в Karboii как checkpoint

4. **Мониторинг и управление**:
   - GET `/{project_id}/checkpoints` - просмотр статуса бэкапов
   - DELETE `/{project_id}/checkpoints/{checkpoint_id}` - удаление бэкапа

### Аутентификация

- Karboii использует Keystone токены: `X-Auth-Token` в заголовках
- Агент использует Bearer токены: `Authorization: Bearer test-token-secret-123`
- Для интеграции необходимо настроить взаимную аутентификацию

## Требования

- Go 1.21+
- Docker и Docker Compose (для локального тестирования)

## Сборка

```bash
make build
```

## Запуск с реальным S3 VK Cloud

После настройки credentials в `docker-compose.yml`:

```bash
# Остановить MinIO (если запущен)
docker-compose down

# Запустить только агента с VK Cloud S3
docker-compose up agent -d

# Проверить здоровье
curl http://localhost:8080/health
```

### Тестирование с MinIO (для разработки)

Для локального тестирования используйте MinIO:

```bash
# Запустить MinIO и агента
docker-compose -f docker-compose.minio.yml up -d

# Запустить тесты
chmod +x scripts/test_api.sh
./scripts/test_api.sh
```

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

Для тестирования функциональности используйте предоставленный скрипт с MinIO:

```bash
# Запуск MinIO для тестирования
make docker-up-minio

# Создание тестовых данных
mkdir -p test_data
echo "test" > test_data/file.txt

# Запуск тест API
chmod +x scripts/test_api.sh
./scripts/test_api.sh

# Остановка
make docker-down
```

Скрипт выполняет полный цикл: создание бэкапа, проверка статуса, список, восстановление, удаление.

### Ручное тестирование API

Используйте curl команды из раздела API выше для ручного тестирования.

## Установка и запуск

### Production с VK Cloud S3

```bash
# 1. Клонируем репозиторий
git clone <repository-url>
cd backup-agent

# 2. Настраиваем credentials в docker-compose.yml
# Замените ваш_access_key, ваш_secret_key, название_вашего_бакета

# 3. Запускаем агента
make docker-up

# 4. Проверяем здоровье
curl http://localhost:8080/health
```

### Тестирование с MinIO

```bash
# 1. Клонируем репозиторий
git clone <repository-url>
cd backup-agent

# 2. Запускаем MinIO и агента
make docker-up-minio

# 3. Создаём тестовые данные
mkdir -p test_data
echo "test" > test_data/file.txt

# 4. Запускаем тесты
chmod +x scripts/test_api.sh
./scripts/test_api.sh

# 5. Останавливаем
make docker-down
```