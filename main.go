package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/geobattles/geobattles-backend/pkg/api"
	"github.com/geobattles/geobattles-backend/pkg/auth"
	"github.com/geobattles/geobattles-backend/pkg/db"
	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/middleware"
	"github.com/geobattles/geobattles-backend/pkg/reverse"
)

// version is public build identity, set by -ldflags "-X main.version=...".
var version = "dev"

func setupRoutes(r *mux.Router, diagnostics *api.Diagnostics) {
	// Diagnostics own method handling so failures and preflights also use no-store.
	r.HandleFunc("/healthz", diagnostics.ServeHealth)
	r.HandleFunc("/readyz", diagnostics.ServeReady)
	r.HandleFunc("/version", diagnostics.ServeVersion)
	r.HandleFunc("/auth/register/user", middleware.Cors(api.RegisterUser)).Methods("POST", "OPTIONS")   // register user
	r.HandleFunc("/auth/register/guest", middleware.Cors(api.RegisterGuest)).Methods("POST", "OPTIONS") // register guest
	r.HandleFunc("/auth/login", middleware.Cors(api.LoginUser)).Methods("POST", "OPTIONS")              // login user
	r.HandleFunc("/auth/logout", middleware.AuthMiddleware(api.LogoutUser)).Methods("GET", "OPTIONS")   // logout user
	r.HandleFunc("/auth/refresh", middleware.Cors(api.RefreshToken)).Methods("POST", "OPTIONS")         // refresh access token
	r.HandleFunc("/updateUser", middleware.AuthMiddleware(api.UpdateUser)).Methods("POST", "OPTIONS")   // update user password / displayname
	r.HandleFunc("/countryList", middleware.Cors(api.ServeCountryList)).Methods("GET", "OPTIONS")       // send list of available countries
	r.HandleFunc("/countryCatalog", middleware.Cors(api.ServeCountryCatalog)).Methods("GET", "OPTIONS")
	r.HandleFunc("/lobby", middleware.Cors(api.ServeGetLobby)).Methods("GET", "OPTIONS")               // got list of all lobbies
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeCreateLobby)).Methods("POST", "OPTIONS") // create lobby
	r.HandleFunc("/lobby", middleware.AuthMiddleware(api.ServeDeleteLobby)).Methods("DELETE")          // delete lobby
	r.HandleFunc("/lobby/admission", middleware.AuthMiddleware(api.ServeLobbyAdmission)).Methods("GET", "OPTIONS")
	r.HandleFunc("/lobby/directory", middleware.OptionalAuth(api.ServeLobbyDirectory)).Methods("GET", "OPTIONS")
	r.HandleFunc("/lobby/directory/{code}", middleware.Cors(api.ServeFindLobby)).Methods("GET", "OPTIONS")
	r.HandleFunc("/lobbySocket", middleware.SocketAuthMiddleware(api.ServeLobbySocket))
}

func configureLogging() {
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
}

func main() {
	dev := flag.Bool("dev", false, "Require isolated local development configuration from .env")
	allowPaidMaps := flag.Bool("allow-paid-maps", false, "Forward the development Maps key; requires -dev and prior spending approval")
	flag.Parse()
	if err := loadConfiguration(*dev, *allowPaidMaps); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	configureLogging()
	if err := auth.Configure(os.Getenv("JWT_KEY")); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := db.ConnectDB(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := logic.InitCountryDB(); err != nil {
		slog.Error("Country assets failed validation", "error", err.Error())
		os.Exit(1)
	}
	err2 := reverse.InitReverse()
	if err2 != nil {
		slog.Error(err2.Error())
	}
	database, databaseErr := db.DB.DB()
	assetsInitialized := err2 == nil && len(logic.CountryList) > 0 && len(logic.PolyDB) > 0
	diagnostics := api.NewDiagnostics(database, databaseErr == nil && assetsInitialized, version)
	router := mux.NewRouter()
	setupRoutes(router, diagnostics)

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0:8080"
	}

	slog.Info("Starting HTTP server", "address", listenAddr)
	err := http.ListenAndServe(listenAddr, router)
	if err != nil {
		slog.Error(err.Error())
	}

}
