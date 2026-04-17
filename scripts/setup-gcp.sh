#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-vega-pokedex}"
REGION="${REGION:-asia-northeast1}"
REPO_NAME="${REPO_NAME:-vega-pokedex}"
USER_EMAIL="${USER_EMAIL:-$(gcloud config get-value account)}"
RUNTIME_SA_NAME="${RUNTIME_SA_NAME:-vega-pokedex-runtime}"
RUNTIME_SA_EMAIL="${RUNTIME_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "==> Project:    $PROJECT_ID"
echo "==> Region:     $REGION"
echo "==> User:       $USER_EMAIL"
echo "==> Runtime SA: $RUNTIME_SA_EMAIL"

echo "==> Switching gcloud project"
gcloud config set project "$PROJECT_ID"

echo "==> Granting IAM roles to $USER_EMAIL"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="user:${USER_EMAIL}" \
  --role="roles/cloudbuild.builds.editor" \
  --condition=None >/dev/null

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="user:${USER_EMAIL}" \
  --role="roles/artifactregistry.writer" \
  --condition=None >/dev/null

echo "==> Enabling required APIs"
gcloud services enable \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  iam.googleapis.com

echo "==> Creating Artifact Registry repository: $REPO_NAME"
if ! gcloud artifacts repositories describe "$REPO_NAME" --location="$REGION" >/dev/null 2>&1; then
  gcloud artifacts repositories create "$REPO_NAME" \
    --repository-format=docker \
    --location="$REGION"
else
  echo "    (already exists, skipped)"
fi

echo "==> Creating runtime service account: $RUNTIME_SA_NAME"
if ! gcloud iam service-accounts describe "$RUNTIME_SA_EMAIL" >/dev/null 2>&1; then
  gcloud iam service-accounts create "$RUNTIME_SA_NAME" \
    --display-name="Cloud Run runtime SA for vega-pokedex"
else
  echo "    (already exists, skipped)"
fi

echo "==> Granting minimal roles to runtime SA"
# ログ書き込みのみ（Cloud Runは起動時に自動で logging.logWriter が必要）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${RUNTIME_SA_EMAIL}" \
  --role="roles/logging.logWriter" \
  --condition=None >/dev/null

echo "==> Creating Secret Manager secrets"
create_secret_if_missing() {
  local name="$1"
  if ! gcloud secrets describe "$name" >/dev/null 2>&1; then
    gcloud secrets create "$name" --replication-policy="automatic"
    echo "    created: $name (add a version with: gcloud secrets versions add $name --data-file=-)"
  else
    echo "    (secret $name already exists, skipped)"
  fi
}

create_secret_if_missing "turso-url"
create_secret_if_missing "turso-auth-token"

echo "==> Granting secret accessor to runtime SA (resource-level)"
for secret in turso-url turso-auth-token; do
  gcloud secrets add-iam-policy-binding "$secret" \
    --member="serviceAccount:${RUNTIME_SA_EMAIL}" \
    --role="roles/secretmanager.secretAccessor" \
    --condition=None >/dev/null
done

echo "==> Granting Cloud Build service account permissions"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
CB_SA="${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${CB_SA}" \
  --role="roles/run.admin" \
  --condition=None >/dev/null

# Cloud Build が Cloud Run サービスをランタイムSAとしてデプロイするために必要
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA_EMAIL" \
  --member="serviceAccount:${CB_SA}" \
  --role="roles/iam.serviceAccountUser" >/dev/null

# Cloud Build が自身のSAを Cloud Run の管理操作で使うためにも必要
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${CB_SA}" \
  --role="roles/iam.serviceAccountUser" \
  --condition=None >/dev/null

cat <<EOF

==> Setup completed.

次にシークレット値をアップロード:
  echo -n "libsql://your-db.turso.io" | gcloud secrets versions add turso-url --data-file=-
  echo -n "eyJ..." | gcloud secrets versions add turso-auth-token --data-file=-

または同梱のヘルパーで .env から自動アップロード:
  ./scripts/upload-secrets.sh

その後デプロイ:
  make deploy
EOF
