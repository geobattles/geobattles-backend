package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var signingKey string

// token object containing jwt token and its expiry
type Token struct {
	Auth_token string
	Expiry     int64
}

type TokenClaims struct {
	UID         interface{}
	DisplayName interface{}
}

// initialize token signing key
func init() {
	signingKey = os.Getenv("JWT_KEY")
}

// create jwt with user id and expiry
func CreateToken(uID string, userName string, displayName string, isGuest bool) (Token, error) {
	claims := jwt.MapClaims{}
	claims["uid"] = uID
	claims["user_name"] = userName
	claims["display_name"] = displayName
	claims["guest"] = isGuest

	// set token expiry 365d from creation TODO: reduce and implement token refresh
	expiry := time.Now().UTC().Add(time.Hour * 24 * 365).Unix()
	claims["exp"] = expiry

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// sign token using symetric cypher
	signed, err := t.SignedString([]byte(signingKey))
	if err != nil {
		return Token{}, err
	}

	return Token{
		Auth_token: signed,
		Expiry:     expiry,
	}, nil
}

// validate jwt and return users uid, throws error if token has expired
func ValidateToken(token string) (*TokenClaims, error) {
	// parse token and validate signing method
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(signingKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	// check if token has expired
	if int64(claims["exp"].(float64)) < time.Now().UTC().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return &TokenClaims{UID: claims["uid"], DisplayName: claims["display_name"]}, nil
}
