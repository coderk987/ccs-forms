package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func postgresConnect() *pgxpool.Pool {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env")
	}

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Error Connecting to PostgreSQL")
		return nil
	}
	return pool
}
