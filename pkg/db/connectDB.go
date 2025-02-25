package db

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/slinarji/go-geo-server/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// initialize connection to the database and migrate models
func ConnectDB() {
	var err error

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "db"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=UTC client_encoding=UTF8",
		os.Getenv("DB_HOST"),
		user,
		os.Getenv("DB_PASSWORD"),
		dbName,
	)

	slog.Debug("Connecting to DB", "dsn", dsn)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	slog.Info("Connected to DB")

	DB.AutoMigrate(
		&models.User{},
		&models.Game{},
		&models.Round{},
		&models.Result{},
	)

	slog.Debug("Migrated DB schema")
}
