#!/usr/bin/env bash
set -euo pipefail

# .env から TURSO_URL / TURSO_AUTH_TOKEN を読み取り Secret Manager にアップロードする。
# 既存の値と同じなら新バージョンを追加しない（冪等）。

PROJECT_ID="${PROJECT_ID:-vega-pokedex}"
ENV_FILE="${ENV_FILE:-.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: $ENV_FILE not found" >&2
  exit 1
fi

gcloud config set project "$PROJECT_ID" >/dev/null

# set -a で export 付き読み込み、未定義変数も安全に扱うため set +u 一時解除
set -a
# shellcheck disable=SC1090
set +u
. "$ENV_FILE"
set -u
set +a

upload_if_changed() {
  local secret_name="$1"
  local value="$2"

  if [[ -z "$value" ]]; then
    echo "SKIP  $secret_name (value empty in $ENV_FILE)"
    return
  fi

  if ! gcloud secrets describe "$secret_name" >/dev/null 2>&1; then
    echo "ERROR: secret '$secret_name' does not exist. Run setup-gcp.sh first." >&2
    exit 1
  fi

  local current=""
  current=$(gcloud secrets versions access latest --secret="$secret_name" 2>/dev/null || true)
  if [[ "$current" == "$value" ]]; then
    echo "OK    $secret_name (unchanged)"
    return
  fi

  printf '%s' "$value" | gcloud secrets versions add "$secret_name" --data-file=- >/dev/null
  echo "ADDED $secret_name (new version)"
}

upload_if_changed "turso-url" "${TURSO_URL:-}"
upload_if_changed "turso-auth-token" "${TURSO_AUTH_TOKEN:-}"

echo "Done."
