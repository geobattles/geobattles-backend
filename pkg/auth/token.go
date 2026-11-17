package auth

import (
	"errors"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var signingKey string

const (
	accessTokenValidity  = time.Minute * 15   // 15 minutes
	refreshTokenValidity = time.Hour * 24 * 7 // 7 days
	accessPurpose        = "access"
	refreshPurpose       = "refresh"
)

type TokenPair struct {
	AccessToken   string
	RefreshToken  string
	AccessExpiry  int64
	RefreshExpiry int64
}

type AccessClaims struct {
	TokenUse    string `json:"token_use"`
	UserName    string `json:"user_name"`
	DisplayName string `json:"display_name"`
	IsGuest     bool   `json:"guest"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	TokenUse string `json:"token_use"`
	jwt.RegisteredClaims
}

// Configure must be called once during startup, before serving requests.
func Configure(key string) error {
	if utf8.RuneCountInString(key) < 32 {
		signingKey = ""
		return errors.New("JWT_KEY must be at least 32 characters")
	}
	signingKey = key
	return nil
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
	if signingKey == "" {
		return "", errors.New("auth is not configured")
	}
	claims := AccessClaims{
		TokenUse:    accessPurpose,
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
	if signingKey == "" {
		return "", errors.New("auth is not configured")
	}
	claims := RefreshClaims{
		TokenUse: refreshPurpose,
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
	if signingKey == "" {
		return nil, errors.New("auth is not configured")
	}
	parsedToken, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	}, jwt.WithLeeway(time.Second*10), jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("expired")
		}
		return nil, errors.New("invalid access token")
	}
	if parsedToken == nil || !parsedToken.Valid {
		return nil, errors.New("invalid access token")
	}
	claims, ok := parsedToken.Claims.(*AccessClaims)
	if !ok || claims == nil || claims.Subject == "" || claims.TokenUse != accessPurpose || claims.ID != "" {
		return nil, errors.New("invalid access token claims")
	}
	return claims, nil
}

// validate jwt, check with db if token is revoken and return claims in valid
func ValidateRefreshToken(token string) (*RefreshClaims, error) {
	if signingKey == "" {
		return nil, errors.New("auth is not configured")
	}
	parsedToken, err := jwt.ParseWithClaims(token, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	}, jwt.WithLeeway(time.Second*10), jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("refresh token expired")
		}
		return nil, errors.New("invalid refresh token")
	}
	if parsedToken == nil || !parsedToken.Valid {
		return nil, errors.New("invalid refresh token")
	}
	claims, ok := parsedToken.Claims.(*RefreshClaims)
	if !ok || claims == nil || claims.Subject == "" || claims.ID == "" || claims.TokenUse != refreshPurpose {
		return nil, errors.New("invalid token claims")
	}
	if db.DB == nil {
		return nil, errors.New("refresh validation unavailable")
	}

	var storedToken models.RefreshToken
	if result := db.DB.First(&storedToken, "id = ? AND user_id = ? AND revoked = false AND expires_at > ?", claims.ID, claims.Subject, time.Now().UTC().Unix()); result.Error != nil {
		return nil, errors.New("invalid or revoked token")
	}

	return claims, nil
}

// Store refresh token in database
func storeRefreshToken(tokenID string, uID string, expiry time.Time) error {
	if db.DB == nil {
		return errors.New("token storage unavailable")
	}
	token := models.RefreshToken{
		ID:        tokenID,
		UserID:    uID,
		Revoked:   false,
		ExpiresAt: expiry.Unix(),
		CreatedAt: time.Now().UTC().Unix(),
	}

	if err := db.DB.Create(&token).Error; err != nil {
		slog.Error("Error storing refresh token")
		return errors.New("error storing refresh token")
	}

	return nil
}

// Invalidate a specific refresh token by its ID
func InvalidateRefreshToken(tokenID string) error {
	if db.DB == nil {
		return errors.New("token storage unavailable")
	}
	result := db.DB.Model(&models.RefreshToken{}).
		Where("id = ? AND revoked = false", tokenID).
		Update("revoked", true)

	if result.Error != nil {
		slog.Error("Error invalidating refresh token")
		return errors.New("failed to invalidate token")
	}
	if result.RowsAffected != 1 {
		return errors.New("invalid or revoked token")
	}

	return nil
}

// Invalidate all refresh tokens for a specific user
func InvalidateAllUserRefreshTokens(userID string) error {
	if db.DB == nil {
		return errors.New("token storage unavailable")
	}
	result := db.DB.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Where("revoked = false").
		Update("revoked", true)

	if result.Error != nil {
		slog.Error("Error invalidating user refresh tokens")
		return errors.New("failed to invalidate user tokens")
	}

	slog.Debug("Invalidated all refresh tokens for user", "userID", userID)

	return nil
}
