# Pokemon Vega Wiki Search

ポケモンベガ（ファンメイドROM）の図鑑データを高速に検索・閲覧できるWebサイト。

**データソース:** [ポケモンベガ攻略Wiki - 図鑑一覧](https://w.atwiki.jp/altair1/pages/19.html)（No.001〜No.386 / 386匹）

## 技術スタック

| 役割 | 技術 |
|---|---|
| 言語 | Go 1.23 |
| スクレイピング | [colly](https://github.com/gocolly/colly) |
| Webサーバー | net/http or chi |
| フロントエンド | Go Template + [htmx](https://htmx.org/) + [Alpine.js](https://alpinejs.dev/) |
| DB | [Turso](https://turso.tech/)（LibSQL / SQLite互換） |
| 画像ストレージ | Google Cloud Storage |
| デプロイ | Cloud Run（サーバー）/ Cloud Run Jobs（スクレイパー） |

## プロジェクト構成

```
├── cmd/
│   ├── server/          # Webサーバーエントリーポイント
│   └── scraper/         # スクレイパーエントリーポイント
├── internal/
│   ├── db/              # Turso接続・クエリ・マイグレーション
│   ├── scraper/         # スクレイピングロジック
│   ├── model/           # 構造体定義（Pokemon, Move等）
│   ├── handler/         # HTTPハンドラー（htmxレスポンス含む）
│   └── storage/         # Cloud Storage画像アップロード
├── templates/           # Go Template（base, index, detail, partials）
├── static/              # CSS
├── migrations/          # SQLマイグレーションファイル
├── docs/                # 設計書
├── Dockerfile.server    # Cloud Run用
└── Dockerfile.scraper   # Cloud Run Jobs用
```

## セットアップ

### 前提条件

- Go 1.23+
- Tursoアカウント（本番）またはローカルSQLite（開発）

### 環境変数

```bash
cp .env.example .env
```

| 変数名 | 説明 |
|---|---|
| `TURSO_URL` | TursoデータベースURL |
| `TURSO_AUTH_TOKEN` | Turso認証トークン |
| `GCS_BUCKET_NAME` | GCSバケット名（画像保存先） |
| `GOOGLE_APPLICATION_CREDENTIALS` | GCPサービスアカウントキーのパス |

ローカル開発時はTursoの代わりにファイルベースSQLite（`file:pokemon.db`）を使用できます。

## ビルド・実行

```bash
# Webサーバー
go run ./cmd/server

# スクレイパー
go run ./cmd/scraper
```

### プロダクションビルド

```bash
CGO_ENABLED=0 go build -o server ./cmd/server
CGO_ENABLED=0 go build -o scraper ./cmd/scraper
```

### テスト

```bash
go test ./...                                       # 全テスト
go test ./internal/scraper/                         # 単一パッケージ
go test -run TestFunctionName ./internal/scraper/   # 単一テスト
```

## 機能

### 検索

- FTS5による名前あいまい検索
- タイプ絞り込み（タイプ1 / タイプ2）
- 種族値の範囲フィルター（HP・攻撃・防御・特攻・特防・素早さ）
- htmxによるページリロードなしの部分更新

### API

```
GET /                            # トップ・検索ページ
GET /pokemon/{id}                # ポケモン詳細
GET /search?q=&type=&speed_min=  # htmx部分レスポンス
```

### スクレイパー

1. 図鑑一覧ページから全386匹のリンクを取得
2. 各ポケモンページを巡回（1〜2秒間隔）
3. 基本情報・種族値・進化・技・入手方法をパース
4. 画像をGCSにアップロード
5. 全データをDBに挿入

## DB構成

SQLite/LibSQL。主要テーブル:

- `pokemon` — 基本情報（名前・タイプ・特性等）
- `base_stats` — 種族値
- `ev_yield` — 努力値
- `evolution` — 進化チェーン
- `move` — 技マスタ
- `learnset_level` / `learnset_tm` / `learnset_tutor` / `learnset_egg` — 習得技4種
- `encounter` — 入手方法
- `pokemon_fts` — FTS5全文検索用仮想テーブル

詳細なスキーマは [`docs/architecture.md`](docs/architecture.md) を参照。

## デプロイ

Docker マルチステージビルド（`CGO_ENABLED=0` + `scratch`ベース）で軽量イメージを生成し、Cloud Runにデプロイ。

```bash
# サーバーイメージのビルド例
docker build -f Dockerfile.server -t pokemon-vega-server .

# スクレイパーイメージのビルド例
docker build -f Dockerfile.scraper -t pokemon-vega-scraper .
```

## ライセンス

本プロジェクトはファンメイドの非公式ツールです。ポケモンおよびポケモンベガの権利はそれぞれの権利者に帰属します。
