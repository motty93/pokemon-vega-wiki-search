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

```bash
# ビルド
CGO_ENABLED=0 go build -o server ./cmd/server
CGO_ENABLED=0 go build -o scraper ./cmd/scraper

# 実行
go run ./cmd/server
go run ./cmd/scraper

# テスト
go test ./...
go test ./internal/scraper/  # 単一パッケージのみ
go test -run TestFunctionName ./internal/scraper/  # 単一テストのみ
```

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

## 注意事項

- スクレイピング時はリクエスト間隔を必ず設ける（1〜2秒のランダムスリープ）
- LibSQLドライバは `github.com/tursodatabase/libsql-client-go` を使用
- collyのパーサーはテーブル構造をベースに実装（各セクションのテーブルヘッダーで判定）
- 画像URLはスクレイピング時にWikiのURLを一時保存し、GCSアップロード後に上書き更新
- Dockerイメージは `CGO_ENABLED=0` + scratch ベースのマルチステージビルド
