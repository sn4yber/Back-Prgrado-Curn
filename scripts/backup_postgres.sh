#!/bin/sh
set -eu

PROJECT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"

if [ -f "$PROJECT_DIR/.env" ]; then
  # Carga variables de entorno para DB_USER/DB_NAME.
  # shellcheck disable=SC1091
  . "$PROJECT_DIR/.env"
fi

if [ -z "${DB_USER:-}" ] || [ -z "${DB_NAME:-}" ]; then
  echo "DB_USER y DB_NAME son obligatorios (en .env o entorno)."
  exit 1
fi

mkdir -p "$BACKUP_DIR"
TS="$(date +%Y%m%d_%H%M%S)"
FILE="$BACKUP_DIR/${DB_NAME}_${TS}.sql.gz"

echo "[backup] creando dump en $FILE"
cd "$PROJECT_DIR"
docker compose -f "$COMPOSE_FILE" exec -T db pg_dump -U "$DB_USER" "$DB_NAME" | gzip -9 > "$FILE"

echo "[backup] limpiando backups de más de $RETENTION_DAYS días"
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime "+$RETENTION_DAYS" -delete

if [ "${UPLOAD_TO_SPACES:-false}" = "true" ]; then
  if ! command -v aws >/dev/null 2>&1; then
    echo "[backup] aws cli no disponible; se omite subida a Spaces"
    exit 0
  fi

  : "${SPACES_BUCKET:?Define SPACES_BUCKET para subir a Spaces}"
  : "${SPACES_REGION:?Define SPACES_REGION para subir a Spaces}"
  : "${SPACES_ENDPOINT:?Define SPACES_ENDPOINT para subir a Spaces}"

  echo "[backup] subiendo a Spaces s3://$SPACES_BUCKET/postgres/"
  aws s3 cp "$FILE" "s3://$SPACES_BUCKET/postgres/$(basename "$FILE")" \
    --region "$SPACES_REGION" \
    --endpoint-url "$SPACES_ENDPOINT"
fi

echo "[backup] listo"

