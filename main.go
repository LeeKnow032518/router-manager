package main

import (
	"log"
	"os"
	"router-manager/internal/app"

	_ "router-manager/internal/metrics"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	logFile, err := os.OpenFile("/var/log/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	runMigration()
	myApp := app.NewApplication()

	myApp.Run()
}

func runMigration() {
	dsn := os.Getenv("POSTGRES_GO_URL")
	if dsn == "" {
		log.Fatal("Dsn id empty")
	}

	migration, err := migrate.New("file://db/migration", dsn)
	if err != nil {
		log.Fatalf("Failed to initialize migration: %v", err)
	}
	defer migration.Close()

	migration.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run execute migration: %v", err)
	}
	if err == migrate.ErrNoChange {
		log.Printf("No changes in table scheme")
	} else {
		log.Printf("Migration completed")
	}
}
