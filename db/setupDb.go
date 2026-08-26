package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupDb(pool *pgxpool.Pool) error {
	dataSetup, err := os.ReadFile("sql/setup.sql")
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

	dataSample, err := os.ReadFile("sql/sample_data.sql")
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

	return nil
}
