package db

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/geobattles/geobattles-backend/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB publishes the database connection only after migrating its models.
func ConnectDB() error {
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

	// GORM's default error logger can print SQL parameter values (including
	// password hashes/token rows) or connection details. Log safe context here.
	candidate, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		// Open can return a handle with an allocated pool after a failed ping.
		if candidate != nil {
			if sqlDB, poolErr := candidate.DB(); poolErr == nil {
				_ = sqlDB.Close()
			}
		}
		return errors.New("Failed to connect to the database; check local database configuration and availability")
	}

	if err := candidate.AutoMigrate(
		&models.User{},
		&models.Game{},
		&models.Round{},
		&models.Result{},
		&models.RefreshToken{},
	); err != nil {
		if sqlDB, poolErr := candidate.DB(); poolErr == nil {
			_ = sqlDB.Close()
		}
		return errors.New("Failed to migrate the database schema; check local database permissions and schema compatibility")
	}

	DB = candidate
	slog.Info("Connected to DB")
	slog.Debug("Migrated DB schema")
	return nil
}
