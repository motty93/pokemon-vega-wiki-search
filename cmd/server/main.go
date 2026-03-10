package main

import (
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("GET /pokemon/{id}", h.PokemonDetail)
	mux.HandleFunc("GET /search", h.Search)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
