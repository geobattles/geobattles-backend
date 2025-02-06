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
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=db port=5432 sslmode=disable TimeZone=UTC client_encoding=UTF8",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
	)

	slog.Debug("Connecting to DB", "dsn", dsn)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	slog.Info("Connected to DB")

	DB.AutoMigrate(&models.User{})

	slog.Debug("Migrated DB schema")
}
