package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// Open はTursoデータベースに接続する
// 環境変数 TURSO_URL, TURSO_AUTH_TOKEN が必須
func Open() (*sql.DB, error) {
	tursoURL := os.Getenv("TURSO_URL")
	if tursoURL == "" {
		return nil, fmt.Errorf("TURSO_URL is required")
	}
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")
	if tursoToken == "" {
		return nil, fmt.Errorf("TURSO_AUTH_TOKEN is required")
	}

	dsn := tursoURL + "?authToken=" + tursoToken

	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Migrate はマイグレーションを実行する
func Migrate(db *sql.DB) error {
	data, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	statements := strings.Split(string(data), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute migration: %s: %w", stmt[:min(len(stmt), 80)], err)
		}
	}

	return nil
}
