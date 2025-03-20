package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/geobattles/geobattles-backend/pkg/auth"
	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/models"
)

type registerResponse struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

type response struct {
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

type contextKey string

const (
	UidKey         contextKey = "uid"
	DisplayNameKey contextKey = "displayname"
)

// writes response with given status code and payload
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		fmt.Fprintf(w, "%s", err.Error())
		slog.Error(err.Error())
	}
	slog.Debug("Sent response", "data", data)
}

// send response with given error code and message
func ERROR(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		JSON(w, statusCode, struct {
			Error string `json:"error"`
		}{
			Error: err.Error(),
		})
		return
	}
	JSON(w, http.StatusBadRequest, nil)
}

// creates user and returns id
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	user := models.User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	result := db.DB.Create(&user)

	if result.Error != nil {
		slog.Error("Error creating user", "error", result.Error.Error())
		ERROR(w, http.StatusConflict, result.Error)
		return
	}

	tokenPair, err := auth.CreateTokenPair(user.ID, user.UserName, user.DisplayName, user.IsGuest)
	if err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusCreated, tokenPair)
}

// creates guest and returns id
func RegisterGuest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	guest := models.Guest{}
	err = json.Unmarshal(body, &guest)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	result := db.DB.Create(&guest)

	if result.Error != nil {
		slog.Error("Error creating guest", "error", result.Error.Error())
		ERROR(w, http.StatusConflict, result.Error)
		return
	}

	tokenPair, err := auth.CreateTokenPair(guest.ID, "", guest.DisplayName, guest.IsGuest)
	if err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusCreated, tokenPair)
}

// validates user credentials and returns jwt auth_token
func LoginUser(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	user := models.User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	var dbUser models.User
	result := db.DB.First(&dbUser, "is_guest = False AND user_name = ?", strings.ToLower(user.UserName))
	if result.Error != nil {
		ERROR(w, http.StatusUnauthorized, result.Error)
		return
	}

	if logic.VerifyPassword(dbUser.Password, user.Password) != nil {
		ERROR(w, http.StatusUnauthorized, fmt.Errorf("wrong password"))
		return
	}

	tokenPair, err := auth.CreateTokenPair(dbUser.ID, dbUser.UserName, dbUser.DisplayName, dbUser.IsGuest)
	if err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusOK, tokenPair)
}

func RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ERROR(w, http.StatusBadRequest, err)
		return
	}

	claims, err := auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		ERROR(w, http.StatusUnauthorized, err)
		return
	}

	tokenID := claims.ID
	userID := claims.Subject

	// Revoke the old token
	if err := auth.InvalidateRefreshToken(tokenID); err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	// Get user details to generate new token pair
	var user models.User
	db.DB.First(&user, "id = ?", userID)

	tokenPair, err := auth.CreateTokenPair(userID, user.UserName, user.DisplayName, user.IsGuest)
	if err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusOK, tokenPair)
}

// updates user password and / or displayname
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	var updateRequest models.User
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		ERROR(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	ctx := r.Context()
	uid := ctx.Value(UidKey).(string)

	// check what to update
	fieldsToUpdate := make([]string, 0, 2)
	if updateRequest.DisplayName != "" {
		fieldsToUpdate = append(fieldsToUpdate, "DisplayName")
	}
	if updateRequest.Password != "" {
		fieldsToUpdate = append(fieldsToUpdate, "Password")
	}

	if len(fieldsToUpdate) == 0 {
		ERROR(w, http.StatusBadRequest, fmt.Errorf("nothing to update"))
		return
	}

	// update user in db
	updateRequest.ID = uid
	result := db.DB.Model(&updateRequest).Select(fieldsToUpdate).Updates(updateRequest)

	if result.Error != nil {
		ERROR(w, http.StatusConflict, result.Error)
		return
	}

	// get the complete updated user info
	var updatedUser models.User
	if err := db.DB.First(&updatedUser, "id = ?", uid).Error; err != nil {
		ERROR(w, http.StatusInternalServerError, fmt.Errorf("error retrieving updated user: %w", err))
		return
	}

	// invalidate all existing refresh tokens
	if err := auth.InvalidateAllUserRefreshTokens(uid); err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	// Generate and return new token pair
	tokenPair, err := auth.CreateTokenPair(
		updatedUser.ID,
		updatedUser.UserName,
		updatedUser.DisplayName,
		updatedUser.IsGuest,
	)
	if err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusOK, tokenPair)
}

// revokes all refresh tokens for user
func LogoutUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := ctx.Value(UidKey).(string)

	// invalidate all existing refresh tokens
	if err := auth.InvalidateAllUserRefreshTokens(uid); err != nil {
		ERROR(w, http.StatusInternalServerError, err)
		return
	}

	JSON(w, http.StatusOK, nil)
}
