# デプロイ手順

Cloud Run へのデプロイおよびシークレット運用の手順。

## アーキテクチャ前提

- **読み取りDB:** SQLite（Dockerイメージに同梱、`/data/pokemon.db`、起動時にread-only化）
- **書き込みDB:** Turso（アナリティクス専用、Secret Manager経由で注入）
- **認証:** 公開サイト（`--allow-unauthenticated`）
- **ランタイムSA:** `vega-pokedex-runtime@<PROJECT_ID>.iam.gserviceaccount.com`
- **シークレット:** Secret Manager の `turso-url` / `turso-auth-token`

## 初回セットアップ

GCPプロジェクト側のリソース（SA、Secret Manager、IAM、Artifact Registry）を一括構築する。スクリプトは冪等なので何度実行しても安全。

```bash
./scripts/setup-gcp.sh
```

環境変数で上書き可：

| 変数 | デフォルト |
|---|---|
| `PROJECT_ID` | `vega-pokedex` |
| `REGION` | `asia-northeast1` |
| `REPO_NAME` | `vega-pokedex` |
| `RUNTIME_SA_NAME` | `vega-pokedex-runtime` |
| `USER_EMAIL` | `gcloud config` のアカウント |

このスクリプトが行うこと：

1. 必要なGCP APIを有効化（cloudbuild / artifactregistry / run / secretmanager / iam）
2. Artifact Registry リポジトリ作成
3. ランタイムSA `vega-pokedex-runtime` 作成
4. ランタイムSAに `roles/logging.logWriter` のみ付与（最小権限）
5. Secret Manager シークレット `turso-url` / `turso-auth-token` を空で作成
6. 各シークレットに**リソースレベル**で `roles/secretmanager.secretAccessor` をランタイムSAに付与（プロジェクト全体ではない）
7. Cloud Build SA に `roles/run.admin` と `roles/iam.serviceAccountUser`（ランタイムSAへのimpersonate用）付与
8. ユーザに `roles/cloudbuild.builds.editor` / `roles/artifactregistry.writer` 付与

## シークレットの投入・更新

`.env` の `TURSO_URL` / `TURSO_AUTH_TOKEN` を Secret Manager にアップロードする。

```bash
./scripts/upload-secrets.sh
```

- 既存値と同じなら新バージョンを追加しない（冪等）
- 値が空ならスキップ
- シークレット自体が未作成なら事前に `setup-gcp.sh` を実行すること

## デプロイ

```bash
make deploy
```

内部では `gcloud builds submit --config=cloudbuild.yml --substitutions=SHORT_SHA=<HEAD短SHA>` が走り、以下の順で実行される：

1. Docker build
2. Artifact Registry へ push
3. Cloud Run にデプロイ（ランタイムSA指定・Secret Manager注入付き）

**注意:** `SHORT_SHA` は `git rev-parse --short HEAD` で取るため、未コミットの変更はイメージタグに反映されない。コミットしてから `make deploy` する。

## デプロイ後の動作確認

```bash
SERVICE_URL=$(gcloud run services describe vega-pokedex --region=asia-northeast1 --format='value(status.url)')

# セキュリティヘッダ
curl -sI "$SERVICE_URL/" | grep -iE "content-security|x-frame|strict-transport|ratelimit"

# 検索エンドポイント
curl -s -o /dev/null -w "HTTP %{http_code}\n" "$SERVICE_URL/search?q=ピカ"

# ログ確認（"Analytics DB not configured" が出ていなければTurso接続成功）
gcloud run services logs read vega-pokedex --region=asia-northeast1 --limit=20
```

## Tursoトークンのローテーション

```bash
# 1. 新トークン発行
turso db tokens create pokemon-vega-search-stg-motty93

# 2. .env を差し替え
vim .env

# 3. Secret Manager に反映
./scripts/upload-secrets.sh

# 4. Cloud Run を再デプロイ（Secret Managerの新バージョンを読ませる）
make deploy

# 5. 動作確認後、古いトークンを無効化
turso db tokens invalidate pokemon-vega-search-stg-motty93
```

`--set-secrets=KEY=secret:latest` でデプロイしているので、シークレットの新バージョン追加後に Cloud Run を再デプロイするだけで切り替わる。

## トラブルシュート

### デプロイ時に `PERMISSION_DENIED` が出る
- Cloud Build SA の `roles/iam.serviceAccountUser` 付与が抜けている可能性。`setup-gcp.sh` を再実行。

### 起動ログに `Analytics DB not configured` が出続ける
- `TURSO_URL` 環境変数が注入されていない。Secret Manager に値があるか確認：
  ```bash
  gcloud secrets versions access latest --secret=turso-url
  ```

### レート制限が効いてない / 全リクエストが同一IPに見える
- `httprate.LimitByRealIP` を使っているので、Cloud Run 前段でのX-Forwarded-Forが必須。独自LBを前に挟んでいる場合はヘッダの伝播を確認。

### ローカル実行でTursoに書き込まれてしまう
- `.env` の `TURSO_URL` を空にすれば analytics は no-op になる。もしくは `unset TURSO_URL` してから `make server`。

## 関連ファイル

- `cloudbuild.yml` — ビルド・デプロイ設定
- `Dockerfile` — distroless/static:nonrootベース
- `scripts/setup-gcp.sh` — GCPリソース初期化
- `scripts/upload-secrets.sh` — シークレット投入
- `cmd/server/main.go` — HTTPサーバ（タイムアウト・レート制限・ヘッダ設定）
- `internal/handler/middleware.go` — セキュリティヘッダ/ボディ制限ミドルウェア
- `internal/db/db.go` — 読み書き分離のDB接続
