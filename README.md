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
Polling Service → Task Manager → Worker Pool → Storage (S3) + Archiver
Status Reporter ← Task Manager
HTTP Handlers (API для тестирования)
```

### Основные компоненты:

1. **Polling Service**: Опрашивает Karboii API за новыми задачами (новый компонент)
2. **HTTP API**: RESTful интерфейс для управления бэкапами (для тестирования и отладки)
3. **Task Manager**: Управление жизненным циклом задач (создание, отслеживание, очистка)
4. **Worker Pool**: Пул горутин для асинхронной обработки задач
5. **S3 Storage**: Клиент для работы с VK Cloud S3 (на базе MinIO)
6. **Archiver**: Создание/распаковка tar.gz архивов
7. **Status Reporter**: Отправляет статус выполненных задач обратно в Karboii (новый компонент)
8. **Logger**: Структурированное логирование с уровнями

### Поток операций бэкапа (интегрированная модель):

1. Polling Service получает задачу от Karboii API
2. Создание задачи в Task Manager
3. Передача в Worker Pool для асинхронной обработки
4. Архивация исходных данных в tar.gz
5. Загрузка архива в VK Cloud S3
6. Обновление статуса задачи в Task Manager
7. Status Reporter отправляет результат в Karboii

### Поток операций восстановления:

1. Polling Service получает задачу restore от Karboii
2. Скачивание архива из S3
3. Распаковка в целевую директорию
4. Status Reporter отправляет статус в Karboii

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

Backup Agent разработан для интеграции с сервисом резервного копирования Karboii VK Cloud. Karboii расположен во внутреннем контуре и не может передавать данные по сети наружу. Поэтому агент использует механизм polling: периодически запрашивает Karboii API за новыми задачами и выполняет их.

### Настройка интеграции в Karboii

1. **Разверните агента** на VM в VK Cloud или Codespaces с доступом к Karboii API
2. **Настройте переменные окружения агента** для подключения к Karboii:
   ```bash
   export KARBOII_ENDPOINT="https://karboii.internal.vkcloud.ru/api/v1"
   export KARBOII_TOKEN="your-karboii-token"
   export POLLING_INTERVAL="30s"  # Интервал опроса задач
   export PROJECT_ID="your-project-id"
   ```
3. **В Karboii настройте проект** и планы бэкапа через его интерфейс или API
4. **Агент автоматически начнет опрос** Karboii за задачами и их выполнение

### Компоненты интеграции

- **Polling Service**: Фоновая служба в агенте, которая периодически опрашивает Karboii API
- **Task Processor**: Обрабатывает полученные задачи и выполняет операции бэкапа/восстановления
- **Status Reporter**: Отправляет обновления статуса выполненных задач обратно в Karboii

### Тестирование интеграции

Для тестирования интеграции можно использовать curl команды к Karboii API (имитация):

```bash
# Проверка доступности агента
curl http://localhost:8080/health

# Имитация получения задач от Karboii (для тестирования)
# В реальности агент сам опрашивает Karboii
curl -X POST http://localhost:8080/api/v1/backup \
  -H "Authorization: Bearer test-token-secret-123" \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/data",
    "s3_key": "backup-test.tar.gz",
    "task_id": "karboii-task-123"
  }'
```

## Архитектура интеграции с Karboii

### Новая архитектура (с учетом ограничений сети)

Поскольку Karboii расположен во внутреннем контуре и не может передавать данные наружу, используется pull-модель:

1. **Агент как активный компонент**: Агент инициирует все коммуникации
2. **Polling за задачами**: Агент периодически запрашивает Karboii API за новыми задачами
3. **Выполнение и отчетность**: Агент выполняет задачи и отправляет статус обратно в Karboii

### Основные компоненты агента (обновлено):

1. **HTTP API**: RESTful интерфейс для управления бэкапами (для тестирования и отладки)
2. **Polling Service**: Новый компонент для опроса Karboii API
3. **Task Manager**: Управление жизненным циклом задач (теперь получает задачи от Polling Service)
4. **Worker Pool**: Пул горутин для асинхронной обработки задач
5. **S3 Storage**: Клиент для работы с VK Cloud S3
6. **Archiver**: Создание/распаковка tar.gz архивов
7. **Status Reporter**: Отправка статуса задач в Karboii
8. **Logger**: Структурированное логирование

### Поток операций (новая модель):

1. Polling Service опрашивает Karboii API за новыми задачами
2. При получении задачи создается локальная задача в Task Manager
3. Worker Pool асинхронно выполняет задачу (архивирование + загрузка в S3)
4. Status Reporter отправляет результат выполнения обратно в Karboii
5. Karboii обновляет свой статус чекпоинта

### Текущая реализация

**✅ Что работает:**
- Polling Service для получения задач от Karboii
- Асинхронное выполнение бэкапа (архивирование + загрузка в S3)
- Отправка статуса задач обратно в Karboii
- Поддержка всех CRUD операций: backup, restore, list, delete

**⚠️ Архитектурные решения:**
- **Агент как "pull-based executor"**: Активно запрашивает задачи вместо пассивного ожидания
- **Karboii как "task provider"**: Предоставляет задачи через API, получает статус
- **Разделение ответственности**: Karboii = планировщик, Агент = исполнитель + репортер

### Сценарий работы

```
1. Пользователь создает план в Karboii
   POST /{project_id}/plans

2. Karboii настраивает триггер
   POST /{project_id}/triggers

3. По расписанию Karboii подготавливает задачу
   (внутренне в Karboii, без сетевых вызовов)

4. Агент опрашивает Karboii за задачами
   GET /{project_id}/tasks?status=pending
   Получает: [{"task_id": "123", "type": "backup", "source_path": "/data", "s3_key": "backup.tar.gz"}]

5. Агент выполняет задачу асинхронно
   - Создает локальную задачу в Task Manager
   - Архивирует данные в tar.gz
   - Загружает архив в VK Cloud S3
   - Обновляет локальный статус

6. Агент отправляет статус в Karboii
   POST /{project_id}/tasks/{task_id}/status
   Body: {"status": "completed", "result": {"s3_key": "backup.tar.gz", "size": 123456}}

7. Karboii обновляет чекпоинт
   (внутренне обрабатывает результат)

8. Karboii управляет хранением
   - Анализирует стратегии (GFS, retention)
   - Создает задачи на удаление старых бэкапов
   - Агент опрашивает и выполняет задачи удаления
```

### Рекомендации для production

- **Валидация запросов**: Добавить проверку аутентификации при опросе Karboii API
- **Аутентификация**: Настроить токены для доступа к Karboii API (Keystone tokens)
- **Мониторинг**: Логировать все операции polling и выполнения задач
- **Отказоустойчивость**: Обработка сетевых ошибок при опросе Karboii и повторные попытки
- **Масштабирование**: Настроить интервал polling в зависимости от нагрузки (короткий интервал для высокой частоты задач)
- **Безопасность**: Хранить KARBOII_TOKEN securely, использовать HTTPS для коммуникации

### Масштабирование и высокая доступность

Агент активно опрашивает Karboii API для получения задач:

- **Получение задач**: Агент → GET /{project_id}/tasks → Karboii возвращает pending задачи
- **Выполнение бэкапа**: Агент архивирует и загружает в S3 локально
- **Отправка статуса**: Агент → POST /{project_id}/tasks/{task_id}/status → Karboii обновляет статус
- **Восстановление**: Аналогично, агент получает задачу restore и выполняет
- **Список/удаление**: Агент получает задачи на list/delete и выполняет операции с S3

### API Karboii для polling

Агент опрашивает следующие endpoints Karboii:
- `GET /{project_id}/tasks?status=pending` - Получение новых задач
- `POST /{project_id}/tasks/{task_id}/status` - Отправка статуса выполнения

Задачи могут быть типов: backup, restore, list, delete с соответствующими параметрами.

### Сценарий интеграции с Karboii

1. **Создание плана в Karboii**:
   - POST `/{project_id}/plans` - создает план бэкапа
   - В плане указываются ресурсы (VM, диски) и параметры

2. **Настройка триггера**:
   - POST `/{project_id}/triggers` - создает расписание бэкапов
   - Указывается cron-выражение и максимальное количество бэкапов

3. **Агент опрашивает задачи**:
   - Агент периодически вызывает GET `/{project_id}/tasks?status=pending`
   - Karboii возвращает список задач для выполнения

4. **Выполнение бэкапа**:
   - Агент получает задачу типа "backup" с параметрами source_path, s3_key
   - Выполняет архивирование и загрузку в S3
   - Отправляет статус POST `/{project_id}/tasks/{task_id}/status`

5. **Karboii создает чекпоинт**:
   - На основе успешного статуса от агента создается чекпоинт

6. **Мониторинг и управление**:
   - GET `/{project_id}/checkpoints` - просмотр статуса бэкапов
   - DELETE `/{project_id}/checkpoints/{checkpoint_id}` - удаление бэкапа (создает задачу для агента)

### Аутентификация

- Karboii использует Keystone токены для аутентификации агента
- Агент отправляет токен в заголовке `X-Auth-Token` при запросах к Karboii
- Для интеграции необходимо получить токен от Karboii и настроить его в переменной `KARBOII_TOKEN`

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
- `AGENT_TOKEN` - API токен для аутентификации (для тестового API)
- `AGENT_TLS_CERT` - Путь к TLS сертификату
- `AGENT_TLS_KEY` - Путь к TLS ключу
- `KARBOII_ENDPOINT` - URL Karboii API (например: https://karboii.internal.vkcloud.ru/api/v1)
- `KARBOII_TOKEN` - Токен для аутентификации в Karboii API
- `PROJECT_ID` - ID проекта в Karboii
- `POLLING_INTERVAL` - Интервал опроса задач от Karboii (default: 30s)
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