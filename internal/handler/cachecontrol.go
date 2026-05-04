package handler

import (
	"net/http"
	"path/filepath"
	"strings"
)

// ImageCacheControlMiddleware は画像系の拡張子に対して長期 immutable キャッシュを設定する。
// 画像以外（CSS/JS等）には何も設定しないので、上位ハンドラが設定したヘッダがそのまま残る。
func ImageCacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".avif":
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		next.ServeHTTP(w, r)
	})
}
