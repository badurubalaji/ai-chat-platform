package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	cmd := flag.Arg(0)
	if cmd == "" && len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://backend/db/migrations",
		"postgres", driver)
	if err != nil {
		// Try relative path if running from backend dir
		m, err = migrate.NewWithDatabaseInstance(
			"file://db/migrations",
			"postgres", driver)
		if err != nil {
			log.Fatal(err)
		}
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil {
			log.Fatal(err)
		}
		log.Println("Migrations rolled back successfully")
	default:
		log.Println("Usage: go run cmd/migrate/main.go [up|down]")
	}
}
