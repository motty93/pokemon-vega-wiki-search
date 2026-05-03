# robots.txt にAIクローラー拒否を追加

## 背景

AI学習・LLM用クローラー（GPTBot, ClaudeBot, PerplexityBot 等）による大量アクセスを抑止するため、
仕様準拠のクローラーに対して `Disallow: /` を伝える。
強制力はないが、お行儀の良いbotには効くため、無料・低リスクの第一防衛線として実施。

通常検索（Googlebot, Bingbot）と SNS プレビュー系（facebookexternalhit, Twitterbot 等）は引き続き許可し、
検索流入と OGP プレビューは維持する。

## 変更内容

### 対象ファイル

- `internal/handler/handler.go` の `RobotsTxt` ハンドラ

### 変更前

```go
fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /search\n\nSitemap: %s/sitemap.xml\n", h.BaseURL)
```

`User-agent: *` ブロックのみで、AIクローラーを個別拒否していなかった。

### 変更後

raw string literal で robots.txt の本文を保持し、AIクローラー19種を個別に `Disallow: /` 指定。
最後に `User-agent: *` で一般botを許可（`/search` は DB 負荷対策で従来どおり Disallow）。

拒否対象のクローラー:

- OpenAI: `GPTBot`, `ChatGPT-User`, `OAI-SearchBot`
- Anthropic: `ClaudeBot`, `Claude-Web`, `anthropic-ai`
- Google: `Google-Extended`（Googlebot は通常検索のため除外）
- Common Crawl: `CCBot`
- Perplexity: `PerplexityBot`
- Meta: `Meta-ExternalAgent`, `Meta-ExternalFetcher`
- ByteDance: `Bytespider`
- Amazon: `Amazonbot`
- Apple: `Applebot-Extended`（Applebot は Siri 検索用のため除外）
- Cohere: `cohere-ai`
- Diffbot: `Diffbot`
- TheHive AI: `ImagesiftBot`
- Webz.io: `Omgilibot`
- Timpi: `Timpibot`

### 配信方法

既存の chi ルーター (`cmd/server/main.go`) で `/robots.txt` → `h.RobotsTxt` のマッピングが存在しているため追加コード不要。
ルートパス (`/robots.txt`) で配信される。

`Sitemap:` ディレクティブは `h.BaseURL` 経由で動的に組み立てるため、ハンドラ実装を維持（静的ファイル化しない）。

### Content-Type / Cache-Control

既存設定をそのまま維持:

- `Content-Type: text/plain; charset=utf-8`
- `Cache-Control: public, max-age=86400`（1日キャッシュ）

## 確認

ローカルビルド成功:

```sh
go build ./...
# => エラーなし
```

デプロイ後の確認コマンド:

```sh
curl https://vega-pokedex-838747766225.asia-northeast1.run.app/robots.txt
curl -I https://vega-pokedex-838747766225.asia-northeast1.run.app/robots.txt
```

期待される結果:
- `HTTP/2 200`
- `content-type: text/plain; charset=utf-8`
- AIクローラー19種 + `User-agent: *` ブロック + `Sitemap:` 行が返る

## 関連情報

- robots.txt は強制力がないため、お行儀の悪いbotは無視する。次タスク（User-Agent判別ミドルウェア）と組み合わせる前提
- `Google-Extended` は Google の AI 学習用クローラ。Googlebot とは別物のため拒否しても通常検索には影響しない
- `Applebot-Extended` も同様で、Apple Intelligence の学習用。Applebot（Siri 検索用）には影響しない
- `User-agent: *` ブロックの `Disallow: /search` は既存仕様を維持（DB負荷対策）
