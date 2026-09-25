package database

import (
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// MustConnect establishes a connection to the PostgreSQL database and pings it.
// It panics if the connection cannot be established.
func MustConnect(url string) *sqlx.DB {
	db, err := sqlx.Connect("pgx", url)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		panic(err)
	}
	
	// Ensure connection is valid
	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
		panic(err)
	}
	
	slog.Info("Successfully connected to database")
	return db
}

// MustMigrate runs the SQL migrations found in the given path.
// This is a naive implementation that just executes all SQL files in alphabetical order.
// In a real production system, you would use a tool like golang-migrate or goose.
func MustMigrate(db *sqlx.DB, migrationsPath string) {
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.sql"))
	if err != nil {
		slog.Error("Failed to find migrations", "error", err)
		panic(err)
	}
	
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read migration file", "file", file, "error", err)
			panic(err)
		}
		
		_, err = db.Exec(string(content))
		if err != nil {
			slog.Error("Failed to execute migration", "file", file, "error", err)
			// Ignore errors like "relation already exists" for this naive approach
			// Just log them and continue. A real tool tracks applied migrations.
		} else {
			slog.Info("Applied migration", "file", file)
		}
	}
}
