package main

import (
	"log"
	"os"

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

	// 画像保存: GCSが設定されていればGCSへ、なければローカルにダウンロード
	if os.Getenv("GCS_BUCKET_NAME") != "" {
		log.Println("Uploading images to GCS...")
		if err := storage.UploadPokemonImages(db); err != nil {
			log.Fatalf("Image upload failed: %v", err)
		}
		log.Println("Image upload completed")
	} else {
		log.Println("GCS_BUCKET_NAME not set, downloading images locally...")
		if err := storage.DownloadPokemonImages(db, "static/images/pokemon"); err != nil {
			log.Fatalf("Image download failed: %v", err)
		}
		log.Println("Image download completed")
	}
}
