package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/geobattles/geobattles-backend/pkg/api"
	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/middleware"
	"github.com/geobattles/geobattles-backend/pkg/reverse"
)

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/register/user", middleware.Cors(api.RegisterUser)).Methods("POST", "OPTIONS")   // register user
	r.HandleFunc("/auth/register/guest", middleware.Cors(api.RegisterGuest)).Methods("POST", "OPTIONS") // register guest
	r.HandleFunc("/auth/login", middleware.Cors(api.LoginUser)).Methods("POST", "OPTIONS")              // login user
	r.HandleFunc("/auth/logout", middleware.AuthMiddleware(api.LogoutUser)).Methods("GET", "OPTIONS")   // logout user
	r.HandleFunc("/auth/refresh", middleware.Cors(api.RefreshToken)).Methods("POST", "OPTIONS")         // refresh access token
	r.HandleFunc("/updateUser", middleware.AuthMiddleware(api.UpdateUser)).Methods("POST", "OPTIONS")   // update user password / displayname
	r.HandleFunc("/countryList", middleware.Cors(api.ServeCountryList)).Methods("GET", "OPTIONS")       // send list of available countries
	r.HandleFunc("/lobby", middleware.Cors(api.ServeGetLobby)).Methods("GET", "OPTIONS")                // got list of all lobbies
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeCreateLobby)).Methods("POST", "OPTIONS")  // create lobby
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeDeleteLobby)).Methods("DELETE")           // delete lobby
	r.HandleFunc("/lobbySocket", middleware.SocketAuthMiddleware(api.ServeLobbySocket))
}

func init() {
	// try to read .env file
	// in docker we just use ENV variables and this WILL throw an error
	err := godotenv.Load()
	if err != nil {
		slog.Warn("Error loading .env file")
	}

	logLevel := os.Getenv("LOG_LEVEL")

	var lvl slog.Level
	switch logLevel {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})))

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
