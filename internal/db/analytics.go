package db

import (
	"database/sql"
	"log"
)

// LogSearch は検索クエリをログに記録する（db が nil のときは no-op）
func LogSearch(db *sql.DB, query string, resultCount int) {
	if db == nil {
		return
	}
	_, err := db.Exec("INSERT INTO search_log (query, result_count) VALUES (?, ?)", query, resultCount)
	if err != nil {
		log.Printf("failed to log search: %v", err)
	}
}

// LogPageView はポケモンページの閲覧を記録する（db が nil のときは no-op）
func LogPageView(db *sql.DB, pokemonID int) {
	if db == nil {
		return
	}
	_, err := db.Exec("INSERT INTO page_view (pokemon_id) VALUES (?)", pokemonID)
	if err != nil {
		log.Printf("failed to log page view: %v", err)
	}
}
