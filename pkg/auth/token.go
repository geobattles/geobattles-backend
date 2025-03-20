package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var signingKey string

const (
	accessTokenValidity  = time.Minute * 15   // 15 minutes
	refreshTokenValidity = time.Hour * 24 * 7 // 7 days
)

type TokenPair struct {
	AccessToken   string
	RefreshToken  string
	AccessExpiry  int64
	RefreshExpiry int64
}

type AccessClaims struct {
	UserName    string `json:"user_name"`
	DisplayName string `json:"display_name"`
	IsGuest     bool   `json:"guest"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	jwt.RegisteredClaims
}

// initialize token signing key
func init() {
	signingKey = os.Getenv("JWT_KEY")
}

// create jwt with user id and expiry
func CreateTokenPair(uID string, userName string, displayName string, isGuest bool) (TokenPair, error) {
	accessExpiry := time.Now().UTC().Add(accessTokenValidity)
	accessToken, err := createAccessToken(uID, userName, displayName, isGuest, accessExpiry)
	if err != nil {
		return TokenPair{}, err
	}

	refreshExpiry := time.Now().UTC().Add(refreshTokenValidity)
	tokenID := uuid.NewString() // Generate unique ID for this token
	refreshToken, err := createRefreshToken(uID, tokenID, refreshExpiry)
	if err != nil {
		return TokenPair{}, err
	}

	// store refresh token in database
	if err := storeRefreshToken(tokenID, uID, refreshExpiry); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpiry:  accessExpiry.Unix(),
		RefreshExpiry: refreshExpiry.Unix(),
	}, nil
}

// Create signed short lived access token with user claims
func createAccessToken(uID string, userName string, displayName string, isGuest bool, expiry time.Time) (string, error) {
	claims := AccessClaims{
		UserName:    userName,
		DisplayName: displayName,
		IsGuest:     isGuest,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uID,
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// sign token using symetric cypher
	signed, err := t.SignedString([]byte(signingKey))
	if err != nil {
		return "", err
	}

	return signed, nil
}

// Create signed long lived refresh token with limited claims
func createRefreshToken(uID string, tokenID string, expiry time.Time) (string, error) {
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uID,
			ExpiresAt: jwt.NewNumericDate(expiry),
			ID:        tokenID,
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// sign token using symetric cypher
	signed, err := t.SignedString([]byte(signingKey))
	if err != nil {
		return "", err
	}

	return signed, nil
}

// validate jwt and return users uid, throws error if token has expired
func ValidateAccessToken(token string) (*AccessClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	}, jwt.WithLeeway(time.Second*10), jwt.WithValidMethods([]string{"HS256"}))

	switch {
	case parsedToken.Valid:
		if claims, ok := parsedToken.Claims.(*AccessClaims); ok {
			return claims, nil
		} else {
			return nil, errors.New("error parsing claims")
		}
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, fmt.Errorf("expired")
	default:
		return nil, fmt.Errorf("error parsing token")
	}
}

// validate jwt and return users uid, throws error if token has expired
func ValidateRefreshToken(token string) (*RefreshClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	}, jwt.WithLeeway(time.Second*10), jwt.WithValidMethods([]string{"HS256"}))

	switch {
	case parsedToken.Valid:
		if claims, ok := parsedToken.Claims.(*RefreshClaims); ok {
			return claims, nil
		} else {
			return nil, errors.New("error parsing claims")
		}
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, fmt.Errorf("expired")
	default:
		return nil, fmt.Errorf("error parsing token")
	}
}

// Store refresh token in database
func storeRefreshToken(tokenID string, uID string, expiry time.Time) error {
	token := models.RefreshToken{
		ID:        tokenID,
		UserID:    uID,
		Revoked:   false,
		ExpiresAt: expiry.Unix(),
		CreatedAt: time.Now().UTC().Unix(),
	}

	if err := db.DB.Create(&token).Error; err != nil {
		slog.Error("Error storing refresh token", "error", err)
		return errors.New("error storing refresh token")
	}

	return nil
}
