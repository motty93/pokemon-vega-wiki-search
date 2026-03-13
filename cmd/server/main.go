package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
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

	h, err := handler.New(db)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	r := chi.NewRouter()
	r.Use(handler.LoggingMiddleware)

	r.Get("/", h.Index)
	r.Get("/pokemon/{id}", h.PokemonDetail)
	r.Get("/search", h.Search)
	r.Get("/sitemap.xml", h.Sitemap)
	r.Get("/robots.txt", h.RobotsTxt)

	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	r.Get("/static/*", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30日
		staticHandler.ServeHTTP(w, req)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
