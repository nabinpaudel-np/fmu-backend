package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	}
	dbURL = "pgx5" + strings.TrimPrefix(dbURL, "postgres")

	if err := db.RunMigrations(dbURL, "internal/db/migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
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

	authMW := auth.AuthMiddleware(cfg)
	adminMW := auth.RequireRole(auth.RoleAdmin)

	universityRepo := university.NewUniversityRepository(queries, pool)
	universitySvc := university.NewUniversityService(universityRepo)
	universityHandler := university.NewUniversityHandler(universitySvc)

	r := chi.NewRouter()
	auth.RegisterRoutes(r, authHandler)
	university.RegisterRoutes(r, universityHandler, authMW, adminMW)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	log.Println("server running on port", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
