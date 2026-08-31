package db

import (
	"ccs-forms/config"
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var Pool *pgxpool.Pool

func PostgresConnect() {
	err := godotenv.Load()

	if err != nil {
		if config.IsDev() {
			log.Printf("warning: could not load .env: %v", err)
		} else {
			log.Println("warning: no .env file found, using system environment variables")
		}
	}

	dbURL := os.Getenv("DATABASE_URL")

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Error parsing PostgreSQL config: %v", err)
	}

	poolConfig.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		poolConfig,
	)
	if err != nil {
		Pool = nil
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}

	log.Println("Connected to Postgres.")
	Pool = pool
}
