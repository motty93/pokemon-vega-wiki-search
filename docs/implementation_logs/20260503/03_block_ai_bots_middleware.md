# User-Agent判別ミドルウェアでAIクローラーを強制ブロック

## 背景

タスク2（robots.txt にAIクローラー拒否追加）はお行儀の良いbot向けの第一防衛線。
robots.txt を無視するbotには効かないため、リクエストレベルで `403 Forbidden` を返すミドルウェアを追加し、
ハンドラ到達前にブロックする。DB アクセスや画像配信処理を発生させない目的。

User-Agent ヘッダーは詐称可能だが、自分自身を `GPTBot/1.0` のように名乗ってくる主要AI crawlerには有効。
詐称対応（Cloudflare 等）はスコープ外。

## 変更内容

### 新規ファイル: `internal/handler/blockbots.go`

既存ミドルウェアが `internal/handler/middleware.go` に集約されている慣習に合わせ、同パッケージに配置。
`BlockAIBotsMiddleware` 関数を実装し、`User-Agent` 文字列にブロックリスト中のいずれかが部分一致したら `403` を返す。

ブロック対象（19種、robots.txt の Disallow と同じセット）:

- OpenAI: `GPTBot`, `ChatGPT-User`, `OAI-SearchBot`
- Anthropic: `ClaudeBot`, `Claude-Web`, `anthropic-ai`
- Google: `Google-Extended`（Googlebot は除外）
- Common Crawl: `CCBot`
- Perplexity: `PerplexityBot`
- Meta: `Meta-ExternalAgent`, `Meta-ExternalFetcher`
- ByteDance: `Bytespider`
- Amazon: `Amazonbot`
- Apple: `Applebot-Extended`（Applebot は除外）
- Cohere: `cohere-ai`
- Diffbot: `Diffbot`
- TheHive AI: `ImagesiftBot`
- Webz.io: `Omgilibot`
- Timpi: `Timpibot`

### `cmd/server/main.go` のミドルウェアチェーン更新

`LoggingMiddleware` の直後（`SecurityHeadersMiddleware` より前）に `BlockAIBotsMiddleware` を挿入。
理由: ブロックしたリクエストもログに残したい（監視・効果測定のため）。
`SecurityHeaders` より前にしたのは、403 で打ち返すリクエストに対してセキュリティヘッダを付ける必要がないため。

```go
r := chi.NewRouter()
r.Use(handler.LoggingMiddleware)
r.Use(handler.BlockAIBotsMiddleware)  // 追加
r.Use(handler.SecurityHeadersMiddleware)
r.Use(handler.BodyLimitMiddleware(1 << 20))
```

### robots.txt の扱い

ミドルウェアを最外層に近い位置で適用するため、AI crawler からの `/robots.txt` リクエストも一律 403 になる。
これは仕様上問題なし（仕様準拠のbotは403でクロールを諦める）。検索エンジンクローラー（Googlebot 等）は
ブロックリストに含まれていないので robots.txt を正常取得できる。

## 確認

### ローカルビルド

```sh
make build  # => CGO_ENABLED=0 go build -o bin/server ./cmd/server (成功)
```

### ローカル起動して curl で動作確認（PORT=18080）

| User-Agent | 期待 | 実測 |
|---|---|---|
| `GPTBot/1.0` | 403 | 403 |
| `Mozilla/5.0 ClaudeBot/1.0` | 403 | 403 |
| `PerplexityBot/1.0` | 403 | 403 |
| `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36`（通常ブラウザ） | 200 | 200 |
| `Mozilla/5.0 (compatible; Googlebot/2.1; ...)` | 200 | 200 |
| Googlebot で `/robots.txt` | 200 | 200 |
| GPTBot で `/robots.txt` | 403 | 403 |
| 空 UA | 200 | 200 |

ロギングミドルウェアより後ろに配置したため、ブロックされたリクエストも標準出力ログに `403` で残ることを確認:

```
2026/05/03 18:24:45 HEAD / 403 8µs
2026/05/03 18:24:45 HEAD /robots.txt 403 9µs
```

## 関連情報

- パフォーマンス: 1リクエストあたり最大19回の `strings.Contains` だが、文字列比較は十分高速で誤差レベル（数µs）
- 偽陽性リスク: ブロックリスト文字列はすべてbot固有の名称で、通常ブラウザUAと衝突する可能性は極めて低い
- メンテナンス: 新しいAI crawlerが登場したら `blockedAIBots` に追加。
  Cloud Run のアクセスログから不審なUAを定期確認:
  ```sh
  gcloud logging read \
    'resource.type=cloud_run_revision AND httpRequest.userAgent=~".*[Bb]ot.*"' \
    --limit=50 --format="value(httpRequest.userAgent)" | sort | uniq -c | sort -rn
  ```
- 関連タスク: `02_robots_txt_ai_crawler_block.md`（robots.txt 側のブロック設定）
