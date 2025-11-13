package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
}

// InitDB initializes the database connection and creates tables
func InitDB() (*DB, error) {
	db, err := initSQLite()
	if err != nil {
		return nil, err
	}

	wrapper := &DB{db}

	// Create tables
	if err := wrapper.createTables(); err != nil {
		return nil, err
	}

	return wrapper, nil
}

func initSQLite() (*sql.DB, error) {
	dbPath := getEnv("DB_PATH", "./url_shortener.db")
	return sql.Open("sqlite", dbPath)
}

func (db *DB) createTables() error {
	urlsTable := `
	CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_code VARCHAR(20) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		click_count INTEGER DEFAULT 0,
		user_ip VARCHAR(45),
		is_custom BOOLEAN DEFAULT FALSE
	);`

	clicksTable := `
	CREATE TABLE IF NOT EXISTS clicks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url_short_code VARCHAR(20) NOT NULL,
		ip_address VARCHAR(45),
		user_agent TEXT,
		referer TEXT,
		country VARCHAR(100),
		city VARCHAR(100),
		clicked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (url_short_code) REFERENCES urls(short_code)
	);`

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);",
		"CREATE INDEX IF NOT EXISTS idx_clicks_short_code ON clicks(url_short_code);",
		"CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);",
	}

	// Execute table creation
	if _, err := db.Exec(urlsTable); err != nil {
		return fmt.Errorf("failed to create urls table: %v", err)
	}

	if _, err := db.Exec(clicksTable); err != nil {
		return fmt.Errorf("failed to create clicks table: %v", err)
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
