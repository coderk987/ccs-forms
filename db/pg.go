package db

import (
	"ccs-forms/config"
	"context"
	"fmt"
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
			fmt.Errorf("Error loading .env: %v", err)
		}
		log.Println("warning: no .env file found, using system environement variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		Pool = nil
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}

	log.Println("Connected to Postgres.")
	Pool = pool
}
