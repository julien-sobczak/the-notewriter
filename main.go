package main

import (
	"database/sql"
	"embed"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/julien-sobczak/the-notetaker/cmd"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

func init() {
	// Ensure database file exists
	db, _ := sql.Open("sqlite3", "./database.db")
	defer db.Close()

	instance, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Run migrations
	d, err := iofs.New(migrationsFS, "sql")
	if err != nil {
		log.Fatalf("Error while reading migrations: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", d, "sqlite3", instance)
	if err != nil {
		log.Fatalf("Error while initializing migrations: %v", err)
	}

	err = m.Up() // Create/Update table schema_migrations
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error while running migrations: %v", err)
	}
}

func main() {
	cmd.Execute()
}
