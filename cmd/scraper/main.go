package main

import (
	"log"

	mydb "github.com/motty93/pokemon-vega-wiki-crawler/internal/db"
	"github.com/motty93/pokemon-vega-wiki-crawler/internal/scraper"
	"github.com/motty93/pokemon-vega-wiki-crawler/internal/storage"
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

	log.Println("Starting scraper...")
	if err := scraper.Run(db); err != nil {
		log.Fatalf("Scraper failed: %v", err)
	}
	log.Println("Scraping completed successfully")

	log.Println("Downloading images locally...")
	if err := storage.DownloadPokemonImages(db, "static/images/pokemon"); err != nil {
		log.Fatalf("Image download failed: %v", err)
	}
	log.Println("Image download completed")
}
