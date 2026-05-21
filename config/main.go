package config

import (
	"os"

	"github.com/bank-data-db/server/db"
	"github.com/joho/godotenv"
)

var (
	JWT_SECRET []byte
)

func loadDB(require bool) {
	uri := os.Getenv("POSTGRES_URI")
	if uri == "" {
		if require {
			panic("Postgres URI (POSTGRES_URI) not defined, but required")
		}

		return
	}

	db.LoadPool(uri)
}

// Returns cleanup
func LoadBasics() func() error {
	godotenv.Load(".env")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET is absolutely required!")
	}

	JWT_SECRET = []byte(jwtSecret)

	loadDB(true)

	return func() error {
		db.Close()

		return nil
	}
}

func LoadForCLI(withDB bool) func() {
	godotenv.Load(".env")

	loadDB(withDB)

	return func() {
		db.Close()
	}
}

func LoadForTests() {
	loadDB(false)
}
