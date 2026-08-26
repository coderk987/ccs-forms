package db

import (
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
		Pool = nil
		log.Fatal("Error loading .env")
	}

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		Pool = nil
		log.Fatal("Error Connecting to PostgreSQL")
	}

	Pool = pool
}
