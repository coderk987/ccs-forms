package config

import (
	"os"
	"strings"
)

var Env string

func init() {
	Env = strings.ToLower(os.Getenv("APP_ENV"))
	if Env == "" {
		Env = "dev"
	}
}

func IsDev() bool {
	return Env == "dev" || Env == "development"
}

func IsProd() bool {
	return Env == "prod" || Env == "production"
}
