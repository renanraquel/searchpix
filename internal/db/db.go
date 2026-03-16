package db

import (
	"database/sql"
	"fmt"
	"log"

	"searchpix/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open abre conexão e executa migrations
func Open(cfg *config.DBConfig) (*sql.DB, error) {
	driver := cfg.Driver
	openDriver := driver
	switch driver {
	case "postgres":
		openDriver = "pgx" // pgx stdlib registra como "pgx"
	case "sqlite3":
		openDriver = "sqlite" // modernc.org/sqlite registra como "sqlite"
	}
	db, err := sql.Open(openDriver, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("abrir banco: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping banco: %w", err)
	}

	if err := Migrate(db, cfg.Driver); err != nil { // Migrate usa driver original (postgres/sqlite3)
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Println("Banco de dados conectado e migrations aplicadas")
	return db, nil
}
