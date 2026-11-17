package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := setup(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setup() error {
	// Lstat also preserves dangling symlinks without reading existing secrets.
	if _, err := os.Lstat(".env"); err == nil {
		fmt.Println("Kept existing .env (values hidden).")
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("could not inspect .env (values hidden)")
	}
	values, err := godotenv.Read("example.env")
	if err != nil {
		return errors.New("could not read example.env; check syntax and permissions (values hidden)")
	}
	for _, key := range []string{"DB_PASSWORD", "JWT_KEY"} {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return errors.New("could not generate local credentials")
		}
		values[key] = hex.EncodeToString(secret)
	}
	content, err := godotenv.Marshal(values)
	if err != nil {
		return errors.New("could not encode local configuration (values hidden)")
	}
	file, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		fmt.Println("Kept existing .env (values hidden).")
		return nil
	}
	if err != nil {
		return errors.New("could not create private .env (values hidden)")
	}
	_, writeErr := file.WriteString(content + "\n")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("could not finish writing .env; remove the incomplete file before retrying (values hidden)")
	}
	fmt.Println("Created private .env (values hidden).")
	return nil
}
