package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/config"
	"fmu-backend/internal/db"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/oauth"
	"fmu-backend/internal/token"
	"fmu-backend/internal/university"
	"fmu-backend/internal/user"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := db.NewPostgres(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("postgresql database connected successfully")
	defer pool.Close()

	queries := sqlc.New(pool)

	userRepo := user.NewUserRepository(queries)
	userSvc := user.NewUserService(userRepo)
	tokenRepo := token.NewTokenRepository(pool)
	tokenSvc := token.NewTokenService(tokenRepo, cfg)
	oauthSvc := oauth.NewOAuthService(cfg)
	authSvc := auth.NewAuthService(cfg, userSvc, tokenSvc, oauthSvc)
	authHandler := auth.NewAuthHandler(authSvc)

	universityRepo := university.NewUniversityRepository(pool)
	universitySvc := university.NewUniversityService(universityRepo)
	universityHandler := university.NewUniversityHandler(universitySvc)

	r := chi.NewRouter()
	auth.RegisterRoutes(r, authHandler)
	university.RegisterRoutes(r, universityHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	log.Println("server running on port", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
