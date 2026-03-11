# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

ポケモンベガ（ファンメイドROM）の図鑑データをWikiからスクレイピングし、高速に検索・閲覧できるWebサイト。
データソースは `https://w.atwiki.jp/altair1/pages/19.html`（No.001〜No.386、386匹）。

## 技術スタック

- **言語:** Go 1.23（toolchain go1.23.6）
- **スクレイピング:** colly (`github.com/gocolly/colly`)
- **Webサーバー:** net/http or chi
- **フロントエンド:** Go Template + htmx + Alpine.js
- **DB:** Turso（LibSQL / SQLite互換）、ローカル開発時はファイルベースSQLite (`file:pokemon.db`)
- **画像ストレージ:** Google Cloud Storage
- **デプロイ:** Cloud Run（サーバー）/ Cloud Run Jobs（スクレイパー）

## ビルド・実行コマンド

ローカル実行は `.env` を自動読み込みするMakefile経由で行う。

```bash
# ローカル実行（.envから環境変数を読み込み）
make server          # Webサーバー起動
make scraper         # スクレイパー実行

# ビルド
make build           # server + scraper 両方
make build-server    # サーバーのみ
make build-scraper   # スクレイパーのみ

# テスト
make test

# クリーン
make clean
```

本番環境ではシェル環境変数を直接設定する。`go run` を直接使う場合は事前に環境変数のexportが必要。

## アーキテクチャ

2つのエントリーポイント（`cmd/server`、`cmd/scraper`）を持つモノリスGoプロジェクト。

- **`cmd/server/`** — Webサーバー。Go Template + htmxでSSR、検索結果は部分HTMLとして返す
- **`cmd/scraper/`** — Wikiスクレイパー。collyでページ巡回→データパース→DB挿入→画像をGCSアップロード
- **`internal/db/`** — Turso/LibSQL接続、クエリ、マイグレーション
- **`internal/scraper/`** — スクレイピングロジック（テーブル構造ベースでパース）
- **`internal/model/`** — 構造体定義（Pokemon, Move等）
- **`internal/handler/`** — HTTPハンドラー（htmxレスポンス含む）
- **`internal/storage/`** — Cloud Storage画像アップロード
- **`templates/`** — Go Template（base, index, detail, partials）
- **`migrations/`** — SQLマイグレーションファイル

### ルーティング

```
GET /                          # トップ・検索ページ
GET /pokemon/{id}              # ポケモン詳細
GET /search?q=&type=&speed_min= # htmx部分レスポンス
```

### DB構成

SQLite/LibSQL。主要テーブル: `pokemon`（基本情報）、`base_stats`（種族値）、`ev_yield`（努力値）、`evolution`（進化チェーン）、`move`（技マスタ）、`learnset_level/tm/tutor/egg`（習得技4種）、`encounter`（入手方法）、`pokemon_fts`（FTS5全文検索）。

### 検索機能

FTS5による名前あいまい検索、タイプ絞り込み、種族値範囲フィルター。htmxで部分更新。

## 環境変数

```
TURSO_URL=
TURSO_AUTH_TOKEN=
GCS_BUCKET_NAME=
GOOGLE_APPLICATION_CREDENTIALS=
```

## 実装ログ

実装・修正を行った際は `docs/implementation_logs/YYYYMMDD/` 配下にログを残すこと。

- ディレクトリ: `docs/implementation_logs/YYYYMMDD/`（実施日ベース）
- ファイル名: `01_<やったこと>.md`, `02_<やったこと>.md`, ...（連番 + 内容の要約）
- 内容: 背景、変更内容（対象ファイルと何をしたか）、関連情報

## 注意事項

- スクレイピング時はリクエスト間隔を必ず設ける（1〜2秒のランダムスリープ）
- LibSQLドライバは `github.com/tursodatabase/libsql-client-go` を使用
- collyのパーサーはテーブル構造をベースに実装（各セクションのテーブルヘッダーで判定）
- 画像URLはスクレイピング時にWikiのURLを一時保存し、GCSアップロード後に上書き更新
- Dockerイメージは `CGO_ENABLED=0` + scratch ベースのマルチステージビルド
