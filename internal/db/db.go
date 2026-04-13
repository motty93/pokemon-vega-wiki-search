package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// Open はデータベースに接続する
// DATABASE_URL が設定されていればローカルSQLite、なければTurso（TURSO_URL, TURSO_AUTH_TOKEN）を使用
func Open() (*sql.DB, error) {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return openSQLite(dbURL)
	}
	return openTurso()
}

func openSQLite(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// WALモードとFTS5有効化
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	return db, nil
}

func openTurso() (*sql.DB, error) {
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
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to glob migration files: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		statements := splitSQL(string(data))
		for _, stmt := range statements {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %s: %w", file, stmt[:min(len(stmt), 80)], err)
			}
		}
	}

	return nil
}

// splitSQL はSQLをステートメント単位に分割する（BEGIN...END内のセミコロンを考慮）
func splitSQL(sql string) []string {
	var statements []string
	var current strings.Builder
	inBlock := false

	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		upperLine := strings.ToUpper(trimmed)

		if strings.Contains(upperLine, "BEGIN") {
			inBlock = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		if inBlock && strings.HasSuffix(upperLine, "END;") {
			inBlock = false
			statements = append(statements, strings.TrimSpace(current.String()))
			current.Reset()
		} else if !inBlock && strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		remaining = strings.TrimSuffix(remaining, ";")
		if remaining = strings.TrimSpace(remaining); remaining != "" {
			statements = append(statements, remaining)
		}
	}

	return statements
}
