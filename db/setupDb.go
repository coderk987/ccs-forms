package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupDb(pool *pgxpool.Pool) error {
	dataSetup, err := os.ReadFile("./db/setup.sql")
	if err != nil {
		return err
	}

	_, err = pool.Exec(
		context.Background(),
		string(dataSetup),
	)

	if err != nil {
		return err
	}

	log.Println("Ran setup succesfully")
	dataSample, err := os.ReadFile("./db/sample_data.sql")
	if err != nil {
		return err
	}

	_, err = pool.Exec(
		context.Background(),
		string(dataSample),
	)

	if err != nil {
		return err
	}

	log.Println("Ran sample succesfully")
	return nil
}
