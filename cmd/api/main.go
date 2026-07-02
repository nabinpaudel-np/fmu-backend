package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
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

	origins := splitAndTrim(cfg.AllowedOrigins, ",")
	if len(origins) == 0 {
		log.Println("CORS: ALLOWED_ORIGINS not set — cross-origin requests will be blocked")
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

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

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := parts[:0]
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
