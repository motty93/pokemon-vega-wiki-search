package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	mydb "github.com/motty93/pokemon-vega-wiki-crawler/internal/db"
	"github.com/motty93/pokemon-vega-wiki-crawler/internal/handler"
)

func main() {
	db, err := mydb.Open()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := mydb.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// マイグレーション後にSQLiteを読み取り専用化（書き込みを物理的に禁止）
	if os.Getenv("DATABASE_URL") != "" {
		if err := mydb.EnableReadOnly(db); err != nil {
			log.Fatalf("Failed to enable read-only mode: %v", err)
		}
	}

	// アナリティクス用DB（Turso）。TURSO_URL 未設定なら nil となり、ログ記録はスキップされる
	analyticsDB, err := mydb.OpenAnalytics()
	if err != nil {
		log.Fatalf("Failed to open analytics database: %v", err)
	}
	if analyticsDB != nil {
		defer analyticsDB.Close()
		if err := mydb.MigrateAnalytics(analyticsDB); err != nil {
			log.Fatalf("Failed to migrate analytics database: %v", err)
		}
	} else {
		log.Printf("Analytics DB not configured (TURSO_URL unset), analytics disabled")
	}

	h, err := handler.New(db, analyticsDB)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	r := chi.NewRouter()
	r.Use(handler.LoggingMiddleware)
	r.Use(handler.BlockAIBotsMiddleware)
	r.Use(handler.SecurityHeadersMiddleware)
	r.Use(handler.BodyLimitMiddleware(1 << 20)) // 1MB

	// Cloud RunではX-Forwarded-Forに実クライアントIPが入るのでLimitByRealIPを使う
	// 全体: 毎分300リクエスト
	r.Use(httprate.LimitByRealIP(300, time.Minute))

	r.Get("/", h.Index)
	r.Get("/pokemon/{id}", h.PokemonDetail)
	r.Get("/sitemap.xml", h.Sitemap)
	r.Get("/robots.txt", h.RobotsTxt)

	// /search はDBを叩くのでより厳しめに（毎分60リクエスト）
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByRealIP(60, time.Minute))
		r.Get("/search", h.Search)
	})

	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30日
		staticHandler.ServeHTTP(w, req)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	log.Printf("Server starting on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
