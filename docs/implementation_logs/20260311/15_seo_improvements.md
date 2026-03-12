# SEO対応

## 背景

全ページでtitle/descriptionが同一、sitemap/robots.txt/構造化データが未実装だったためSEO対策を実施。

## 変更内容

### `templates/base.html`
- title, meta description, canonical, og:title, og:description, og:type, og:url, og:imageをテンプレートデータから動的に設定
- データ未設定時はデフォルト値にフォールバック
- JSON-LD構造化データの出力枠を追加

### `internal/handler/handler.go`
- `Handler` 構造体に `BaseURL` フィールドを追加（`BASE_URL` 環境変数 or デフォルト値）
- `Index`: 空mapにcanonicalURLのみ設定（DB不要のまま）
- `PokemonDetail`: 以下のSEOデータをテンプレートに渡すよう変更
  - `PageTitle`: 「{名前} (No.{ID}) - ポケモンベガ図鑑 | 種族値・技・入手方法」
  - `PageDescription`: タイプ情報を含む個別説明文
  - `CanonicalURL`: `{BASE_URL}/pokemon/{id}`
  - `OGType`: `article`
  - `OGImage`: コメントアウトで枠のみ（将来実装）
  - `JSONLD`: WebPage + BreadcrumbList構造化データ
- `Sitemap` ハンドラー追加: トップ + 386ポケモンページのsitemap.xml動的生成
- `RobotsTxt` ハンドラー追加: `/search` をDisallow、sitemap.xmlを通知
- `safeJS` テンプレート関数を追加（JS安全出力用）

### `cmd/server/main.go`
- `/sitemap.xml` と `/robots.txt` のルートを追加

## 環境変数

- `BASE_URL`: サイトのベースURL（例: `https://pokemon-vega.example.com`）。未設定時はデフォルト値。

## 未実装（枠のみ）

- `og:image`: コメントアウトで枠を用意済み。OGP画像生成の仕組みが出来次第有効化。

## SEO対応状況

| 項目 | 状態 |
|------|------|
| ページ別title/description | 対応済み |
| canonical URL | 対応済み |
| sitemap.xml | 対応済み（動的生成） |
| robots.txt | 対応済み |
| JSON-LD構造化データ | 対応済み（WebPage + Breadcrumb） |
| og:image | 枠のみ |
