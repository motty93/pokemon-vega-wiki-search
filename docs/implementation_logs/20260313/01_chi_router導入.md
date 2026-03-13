# chi router導入

## 背景

標準の `net/http.ServeMux` からchiルーターに移行し、ルーティングの記述を簡潔にする。

## 変更内容

### cmd/server/main.go
- `http.NewServeMux()` → `chi.NewRouter()` に置き換え
- `mux.HandleFunc("GET /path", ...)` → `r.Get("/path", ...)` に変更
- `handler.LoggingMiddleware(mux)` → `r.Use(handler.LoggingMiddleware)` でミドルウェア登録
- 静的ファイル配信に `Cache-Control: public, max-age=2592000`（30日）ヘッダーを追加

### internal/handler/handler.go
- `r.PathValue("id")` → `chi.URLParam(r, "id")` に変更
- `github.com/go-chi/chi/v5` をimportに追加

### internal/handler/middleware.go
- 変更なし（`func(http.Handler) http.Handler` シグネチャがchi互換のためそのまま使用）

## 関連情報

- `github.com/go-chi/chi/v5 v5.2.5` を `go.mod` に追加
