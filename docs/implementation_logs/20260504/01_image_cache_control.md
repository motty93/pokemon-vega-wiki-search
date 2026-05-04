# 画像レスポンスへの長期immutableキャッシュ付与

## 背景

`http.FileServer` のデフォルトでは `Cache-Control` が付かないため、ブラウザは `Last-Modified` ベースの
ヒューリスティック判断になり、再訪時に毎回 304 確認リクエストが飛ぶ。
ポケモン画像はバージョニング不要・差し替えほぼなしなので、`public, max-age=31536000, immutable` を付与し:

- 同一ユーザー再訪時の通信量削減（ブラウザは1年間サーバーに問い合わせず disk cache を使う）
- 将来 CDN（Cloudflare 等）を前段に置いた際のエッジキャッシュヒット率最大化
- Cloud Run の課金最適化（リクエスト数・転送量削減）

タスク2〜4の bot 対策とは違い、通常ユーザーの体験向上が目的。

## 変更内容

### 新規ファイル: `internal/handler/cachecontrol.go`

タスク4と同じく `internal/handler/` パッケージ慣習に合わせて配置。
`ImageCacheControlMiddleware` を実装し、`r.URL.Path` の拡張子が画像系であれば
`Cache-Control: public, max-age=31536000, immutable` を Set。それ以外は何もしない。

対象拡張子: `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`, `.ico`, `.avif`

`.css` / `.js` / `.woff2` 等は意図的に除外。`immutable` を付ける前提として
ファイル名ハッシュなどのバージョニング運用が必要だが、現プロジェクトでは未導入のため。

### `cmd/server/main.go` の静的配信ハンドラを変更

変更前:
```go
staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Cache-Control", "public, max-age=2592000") // 30日
    staticHandler.ServeHTTP(w, req)
})
```

変更後:
```go
fileServer := http.FileServer(http.Dir("static"))
staticHandler := http.StripPrefix("/static/", handler.ImageCacheControlMiddleware(fileServer))
r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
    // デフォルト30日。画像系は ImageCacheControlMiddleware で 1年 immutable に上書きされる
    w.Header().Set("Cache-Control", "public, max-age=2592000")
    staticHandler.ServeHTTP(w, req)
})
```

### 実行順序の理由

```
r.Get クロージャ実行
  → w.Header().Set("Cache-Control", "30日")          (デフォルト)
  → staticHandler.ServeHTTP(w, req)
    → StripPrefix がパス書き換え
    → ImageCacheControlMiddleware
      → 画像なら w.Header().Set("Cache-Control", "1年immutable") で上書き
      → fileServer.ServeHTTP                          (実ファイル配信)
```

ヘッダーは `WriteHeader` 呼び出し前なら何度でも上書き可能なので、
「クロージャで30日デフォルト → ミドルウェアで画像のみ1年immutableに上書き」が成立する。

現状 `static/` 配下は `images/pokemon/*.png` のみで、CSS/JS は無い。
今後 CSS/JS を追加した際は自動で 30日キャッシュが効く構造。

## 確認

### ローカルビルド

```sh
make build  # => 成功
```

### ローカル起動して curl で動作確認（PORT=18080, GET）

| パス | 期待 | 実測 |
|---|---|---|
| `/static/images/pokemon/267.png` | `public, max-age=31536000, immutable` | ✓ |
| `/`（HTML） | Cache-Control なし | ✓ |
| `/search`（動的） | Cache-Control なし | ✓ |

`curl -I` (HEAD) は chi が GET ハンドラのみ登録のため 405 を返すが、これは無関係（GET で確認済み）。

### デプロイ後の確認コマンド

```sh
curl -i https://vega-pokedex-838747766225.asia-northeast1.run.app/static/images/pokemon/001.png \
  | grep -i cache-control
# => cache-control: public, max-age=31536000, immutable

curl -i https://vega-pokedex-838747766225.asia-northeast1.run.app/ | grep -i cache-control
# => （長期キャッシュヘッダは付かない）
```

ブラウザ DevTools で 2回目以降の画像読み込みが `(disk cache)` から返ることでも検証可能。

## 関連情報

### `immutable` の前提

`immutable` は「この URL のコンテンツは絶対に変わらない」とブラウザに伝える指示。
ユーザーがリロード（F5）してもサーバーに `If-Modified-Since` などの確認リクエストを送らない。
将来「同じURLで画像を差し替える」運用が発生する場合は `immutable` を外し、
`public, max-age=86400` 程度に短縮する必要がある。

スクレイパー (`cmd/scraper`) で再取得する際は同じURL（`/static/images/pokemon/{No}.png`）に
書き戻すので、原則差し替え発生し得るが、ベガROM の図鑑画像は確定データのため実質不変。
万一の差し替え時は手動でCloud Run リビジョン切替・キャッシュバスティング運用で対応する想定。

### 効果が出るタイミング

- 既訪問ユーザーの2回目以降のアクセス
- 同一セッション内で複数の画像を読み込む詳細ページ → 一覧 → 詳細の遷移
- CDN を前段に置いた時のエッジキャッシュ

初回訪問のリクエスト数自体は減らないので、bot 対策（タスク2〜4）の代替にはならない。
タスク1〜4の上に乗せる最適化レイヤー。
