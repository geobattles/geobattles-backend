package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// hash provided string using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// compare hashed password with unhashed string
func VerifyPassword(hashedPassword string, inputPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
}
