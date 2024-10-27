package main

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/slinarji/go-geo-server/pkg/api"
	"github.com/slinarji/go-geo-server/pkg/db"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/middleware"
	"github.com/slinarji/go-geo-server/pkg/reverse"
)

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/register/user", middleware.Cors(api.RegisterUser)).Methods("POST", "OPTIONS")       // register user
	r.HandleFunc("/register/guest", middleware.Cors(api.RegisterGuest)).Methods("POST", "OPTIONS")     // register guest
	r.HandleFunc("/login", middleware.Cors(api.LoginUser)).Methods("POST", "OPTIONS")                  // login user
	r.HandleFunc("/countryList", middleware.Cors(api.ServeCountryList)).Methods("GET", "OPTIONS")      // send list of available countries
	r.HandleFunc("/lobby", middleware.Cors(api.ServeGetLobby)).Methods("GET", "OPTIONS")               // got list of all lobbies
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeCreateLobby)).Methods("POST", "OPTIONS") // create lobby
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeDeleteLobby)).Methods("DELETE")          // delete lobby
	r.HandleFunc("/lobbySocket", middleware.AuthMiddleware(api.ServeLobbySocket))
}

func init() {
	// try to read .env file
	// in docker we just use ENV variables and this WILL throw an error
	err := godotenv.Load()
	if err != nil {
		slog.Info("Error loading .env file")
	}

	db.ConnectDB()

	logic.InitCountryDB()
	err2 := reverse.InitReverse()
	if err2 != nil {
		slog.Error(err2.Error())
	}
}

func main() {
	router := mux.NewRouter()
	setupRoutes(router)

	slog.Info("Server is ready")
	err := http.ListenAndServe("0.0.0.0:8080", router)
	if err != nil {
		slog.Error(err.Error())
	}

}
