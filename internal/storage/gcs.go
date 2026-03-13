package storage

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	mydb "github.com/motty93/pokemon-vega-wiki-crawler/internal/db"
)

// DownloadPokemonImages はDBに保存されたWiki画像URLをローカルにダウンロードし、URLを更新する
func DownloadPokemonImages(d *sql.DB, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	rows, err := d.Query("SELECT id, image_url FROM pokemon WHERE image_url IS NOT NULL AND image_url != '' AND image_url LIKE 'http%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var imageURL string
		if err := rows.Scan(&id, &imageURL); err != nil {
			continue
		}

		localPath, err := downloadImage(destDir, id, imageURL)
		if err != nil {
			log.Printf("WARNING: failed to download image for pokemon %d: %v", id, err)
			continue
		}

		// /static/images/pokemon/001.png 形式で保存
		relativePath := "/" + localPath
		if err := mydb.UpdateImageURL(d, id, relativePath); err != nil {
			log.Printf("WARNING: failed to update image URL for pokemon %d: %v", id, err)
		}
	}

	return nil
}

func downloadImage(destDir string, pokemonID int, srcURL string) (string, error) {
	resp, err := http.Get(srcURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, srcURL)
	}

	ext := filepath.Ext(srcURL)
	if ext == "" || len(ext) > 5 {
		ext = ".png"
	}
	filename := fmt.Sprintf("%03d%s", pokemonID, ext)
	filePath := filepath.Join(destDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	return filePath, nil
}
