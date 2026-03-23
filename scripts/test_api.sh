#!/bin/bash
# test_api.sh

set -e

BASE_URL="http://localhost:8080/api/v1"
TOKEN="test-token-secret-123"
TEST_DIR="/tmp/test_data"
RESTORE_DIR="/tmp/restored_data"

echo "=== Backup Agent Test Script ==="
echo ""

# Создание тестовых данных
echo "[1] Creating test data..."
mkdir -p "$TEST_DIR"
echo "Test file 1" > "$TEST_DIR/file1.txt"
echo "Test file 2" > "$TEST_DIR/file2.txt"
mkdir -p "$TEST_DIR/subdir"
echo "Test file 3" > "$TEST_DIR/subdir/file3.txt"
echo "✓ Test data created"
echo ""

# Health check
echo "[2] Health check..."
curl -s http://localhost:8080/health | jq .
echo "✓ Server is healthy"
echo ""

# Создание бэкапа
echo "[3] Creating backup..."
BACKUP_RESPONSE=$(curl -s -X POST "$BASE_URL/backup" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"source_path\": \"$TEST_DIR\",
    \"s3_key\": \"test/backup-$(date +%s).tar.gz\"
  }")

TASK_ID=$(echo $BACKUP_RESPONSE | jq -r '.task_id')
echo "✓ Backup task created: $TASK_ID"
echo ""

# Ожидание завершения
echo "[4] Waiting for backup to complete..."
for i in {1..30}; do
    STATUS=$(curl -s -X GET "$BASE_URL/task/$TASK_ID" \
      -H "Authorization: Bearer $TOKEN" | jq -r '.status')
    
    if [ "$STATUS" == "completed" ]; then
        echo "✓ Backup completed"
        break
    elif [ "$STATUS" == "failed" ]; then
        echo "✗ Backup failed"
        curl -s -X GET "$BASE_URL/task/$TASK_ID" \
          -H "Authorization: Bearer $TOKEN" | jq .
        exit 1
    fi
    
    echo "  Status: $STATUS (attempt $i/30)"
    sleep 1
done
echo ""

# Список бэкапов
echo "[5] Listing backups..."
curl -s -X GET "$BASE_URL/backups?prefix=test/" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✓ Backups listed"
echo ""

# Получение ключа бэкапа
BACKUP_KEY=$(curl -s -X GET "$BASE_URL/backups?prefix=test/" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.[0].key')

echo "[6] Restoring backup..."
RESTORE_RESPONSE=$(curl -s -X POST "$BASE_URL/restore" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"s3_key\": \"$BACKUP_KEY\",
    \"destination_path\": \"$RESTORE_DIR\"
  }")

RESTORE_TASK_ID=$(echo $RESTORE_RESPONSE | jq -r '.task_id')
echo "✓ Restore task created: $RESTORE_TASK_ID"
echo ""

# Ожидание завершения восстановления
echo "[7] Waiting for restore to complete..."
for i in {1..30}; do
    STATUS=$(curl -s -X GET "$BASE_URL/task/$RESTORE_TASK_ID" \
      -H "Authorization: Bearer $TOKEN" | jq -r '.status')
    
    if [ "$STATUS" == "completed" ]; then
        echo "✓ Restore completed"
        break
    elif [ "$STATUS" == "failed" ]; then
        echo "✗ Restore failed"
        curl -s -X GET "$BASE_URL/task/$RESTORE_TASK_ID" \
          -H "Authorization: Bearer $TOKEN" | jq .
        exit 1
    fi
    
    echo "  Status: $STATUS (attempt $i/30)"
    sleep 1
done
echo ""

# Проверка восстановленных файлов
echo "[8] Verifying restored files..."
if docker exec backup-agent-agent-1 test -f "/tmp/restored_data/file1.txt"; then
    echo "✓ file1.txt restored"
else
    echo "✗ file1.txt not found"
    exit 1
fi

if docker exec backup-agent-agent-1 test -f "/tmp/restored_data/subdir/file3.txt"; then
    echo "✓ file3.txt in subdir restored"
else
    echo "✗ file3.txt not found"
    exit 1
fi
echo ""

# Удаление бэкапа
echo "[9] Deleting backup..."
curl -s -X DELETE "$BASE_URL/backups/$BACKUP_KEY" \
  -H "Authorization: Bearer $TOKEN"
echo "✓ Backup deleted"
echo ""

echo "=== All tests passed! ==="