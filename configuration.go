package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func loadConfiguration(dev, allowPaidMaps bool) error {
	if allowPaidMaps && !dev {
		return errors.New("-allow-paid-maps requires -dev")
	}
	if !dev {
		if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			// Dotenv parse errors can contain secret source lines.
			return errors.New("could not load .env; check its syntax and permissions (values hidden)")
		}
		return nil
	}

	values, err := godotenv.Read(".env")
	if err != nil {
		return errors.New("could not read .env; run make setup or check its syntax and permissions (values hidden)")
	}
	for _, setting := range [][2]string{
		{"DB_HOST", "127.0.0.1"},
		{"DB_USER", "geobattles_dev"},
		{"DB_NAME", "geobattles_dev"},
		{"LISTEN_ADDR", "127.0.0.1:8080"},
		{"LOG_LEVEL", "INFO"},
	} {
		if values[setting[0]] != setting[1] {
			return fmt.Errorf("development requires %s=%s", setting[0], setting[1])
		}
	}
	// The legacy database connector interpolates this into an unquoted DSN.
	password := values["DB_PASSWORD"]
	if len(password) != 64 || strings.Trim(password, "0123456789abcdef") != "" {
		return errors.New("development DB_PASSWORD must be exactly 64 lowercase hexadecimal characters")
	}
	if !allowPaidMaps {
		values["GMAPS_API"] = ""
	}
	// Only application settings may override the inherited environment.
	for _, key := range []string{
		"DB_HOST", "DB_USER", "DB_NAME", "DB_PASSWORD", "JWT_KEY",
		"LISTEN_ADDR", "LOG_LEVEL", "GMAPS_API",
	} {
		if err := os.Setenv(key, values[key]); err != nil {
			return fmt.Errorf("could not configure %s (value hidden)", key)
		}
	}
	// main validates JWT_KEY with auth.Configure before any database access.
	return nil
}
