# Cloud Run デプロイ修正（ランタイムSA・Turso削除・distroless WORKDIR）

## 背景

SQL Playground（DuckDB-WASM）追加後に初めて `make deploy` を実行したところ、
Cloud Build のビルド・push は成功するが Cloud Run デプロイ（Step #2）で失敗した。
調査の結果、デプロイ環境のセットアップ不備と Dockerfile の WORKDIR 問題が重なっていた。

## 原因と対応

### 1. ランタイムSAが存在しない / actAs 権限不足

- エラー: `PERMISSION_DENIED: Permission 'iam.serviceaccounts.actAs' denied on
  service account vega-pokedex-runtime@... (or it may not exist)`
- `gcloud iam service-accounts list` で確認 → ランタイムSA `vega-pokedex-runtime` が
  **存在しなかった**（Compute デフォルトSAのみ）。
- さらに Cloud Build の実行SAが従来の `@cloudbuild` から
  **Compute デフォルトSA（`{番号}-compute@developer.gserviceaccount.com`）** に変わっており
  （Google の仕様変更）、`setup-gcp.sh` が `@cloudbuild` にだけ付与していた actAs が効いていなかった。
- 対応（手動 gcloud）:
  - `gcloud iam service-accounts create vega-pokedex-runtime` でランタイムSA作成
  - `roles/logging.logWriter` をプロジェクトに付与
  - Compute デフォルトSA → ランタイムSAへの `roles/iam.serviceAccountUser`（actAs）を付与
  - Compute SA はもともと `roles/editor` を持つため run デプロイ権限はあり、不足は actAs のみだった。
- 恒久対応: `scripts/setup-gcp.sh` を修正し、`@cloudbuild` と Compute デフォルトSA の
  **両方**に run.admin / actAs を付与するようループ化（再発防止）。

### 2. Secret Manager 未使用 / Turso シークレット不在

- `gcloud secrets list` → Secret Manager API が無効（"has not been used before"）。
  本番は SQLite 同梱で Turso（アナリティクス）は未使用。
- `cloudbuild.yml` の `--set-secrets TURSO_URL=...,TURSO_AUTH_TOKEN=...` を**削除**。
- アプリは `TURSO_URL` 未設定でアナリティクス無効・正常起動する設計（main.go）。

### 3. distroless の WORKDIR でテンプレートが読めずコンテナ起動失敗

- 上記を直すとデプロイは Step #2 のリビジョン作成まで到達したが、今度はコンテナが
  PORT=8080 で起動せず失敗。Cloud Run ログに:
  `Failed to create handler: parse base.html: open templates/base.html: no such file or directory`
- 原因: アプリは `templates/`・`static`（相対パス）でアセットを読む。
  Dockerfile は 4/17 に `FROM scratch` → `FROM gcr.io/distroless/static-debian12:nonroot`
  へ変更されていたが、**distroless:nonroot の WORKDIR は `/home/nonroot`**（`docker inspect` で確認）。
  scratch は WORKDIR 未設定（=`/`）だったため相対パスで `/templates` が読めていたが、
  distroless では `/home/nonroot/templates` を探して失敗。DBは絶対パス `/data/pokemon.db`
  なので読めており、「DBは読めるがテンプレートは読めない」症状になった。
  （新リビジョンが起動失敗したため、現在も scratch 版の旧リビジョン 00003 が稼働してトラフィックを保持していた）
- 対応: `Dockerfile` の 2nd stage に **`WORKDIR /`** を1行追加し、scratch 版と同じ CWD=`/` を再現。

## 結果

- 再デプロイ成功（Cloud Build STATUS: SUCCESS）。
- 本番で `/`・`/playground`・`/static/data/*.parquet` がすべて 200 を返すことを確認。

## 対象ファイル

- `scripts/setup-gcp.sh` — CB_SA を Compute デフォルトSAにも対応（両SAへ run.admin / actAs）
- `cloudbuild.yml` — Turso の `--set-secrets` 削除
- `Dockerfile` — 2nd stage に `WORKDIR /` 追加

## 関連メモ

- distroless:nonroot の WORKDIR は `/home/nonroot`。相対パスでアセットを読むGoアプリは
  `WORKDIR /` を明示するか、アセットを WORKDIR 配下に置く必要がある。
- 本番は SQLite 同梱・Turso 未使用（`docs/implementation_logs` 既存メモと整合）。
