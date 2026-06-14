package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/config"
	"fmu-backend/internal/database"
	"fmu-backend/internal/user"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("postgresql database connected successfully")
	defer db.Close()

	userRepo := user.NewUserRepository(db)
	userSvc := user.NewUserService(userRepo)
	authSvc := auth.NewAuthService(userSvc)
	authHandler := auth.NewAuthHandler(authSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Println("server running on port", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
