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

	token, err := auth.CreateToken(user.ID, user.UserName, user.DisplayName, user.IsGuest)

	JSON(w, http.StatusCreated, token)
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

	token, err := auth.CreateToken(guest.ID, "", guest.DisplayName, guest.IsGuest)

	JSON(w, http.StatusCreated, token)
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

	if auth.VerifyPassword(dbUser.Password, user.Password) != nil {
		ERROR(w, http.StatusUnauthorized, fmt.Errorf("wrong password"))
		return
	}

	token, err := auth.CreateToken(dbUser.ID, dbUser.UserName, dbUser.DisplayName, dbUser.IsGuest)

	JSON(w, http.StatusOK, token)
}
