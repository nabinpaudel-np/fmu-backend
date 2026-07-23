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
	"fmu-backend/internal/claim"
	"fmu-backend/internal/cloudinary"
	"fmu-backend/internal/college"
	"fmu-backend/internal/config"
	"fmu-backend/internal/db"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/favorites"
	"fmu-backend/internal/oauth"
	"fmu-backend/internal/supabase"
	"fmu-backend/internal/token"
	"fmu-backend/internal/university"
	"fmu-backend/internal/uploads"
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
	authHandler := auth.NewAuthHandler(authSvc, cfg)

	authMW := auth.AuthMiddleware(cfg)
	optionalAuthMW := auth.OptionalAuthMiddleware(cfg)
	adminMW := auth.RequireRole(auth.RoleAdmin)
	adminOrRepMW := auth.RequireRole(auth.RoleAdmin, auth.RoleRepresentative)
	studentMW := auth.RequireRole(auth.RoleStudent)

	favoritesRepo := favorites.NewRepository(queries, pool)

	universityRepo := university.NewUniversityRepository(queries, pool)
	universitySvc := university.NewUniversityService(universityRepo)
	universityHandler := university.NewUniversityHandler(universitySvc, favoritesRepo)

	collegeRepo := college.NewCollegeRepository(queries, pool)
	collegeSvc := college.NewCollegeService(collegeRepo)
	collegeHandler := college.NewCollegeHandler(collegeSvc, favoritesRepo)

	favoritesSvc := favorites.NewService(favoritesRepo, universitySvc, collegeSvc)
	favoritesHandler := favorites.NewHandler(favoritesSvc)

	uniClaimRepo := claim.NewUniversityClaimRepository(queries)
	colClaimRepo := claim.NewCollegeClaimRepository(queries)
	claimSvc := claim.NewClaimService(
		uniClaimRepo,
		colClaimRepo,
		claim.NewUniversityExisterAdapter(universitySvc),
		claim.NewCollegeExisterAdapter(collegeSvc),
		userSvc,
	)
	claimHandler := claim.NewClaimHandler(claimSvc)

	cld, err := cloudinary.New(cloudinary.Config{
		CloudName:      cfg.Cloudinary.CloudName,
		APIKey:         cfg.Cloudinary.APIKey,
		APISecret:      cfg.Cloudinary.APISecret,
		Folder:         cfg.Cloudinary.Folder,
		AppEnv:         cfg.AppEnv,
		SecureDelivery: cfg.Cloudinary.SecureDelivery,
	})
	if err != nil {
		log.Fatalf("cloudinary init failed: %v", err)
	}

	supa, err := supabase.New(supabase.Config{
		URL:            cfg.Supabase.URL,
		ServiceRoleKey: cfg.Supabase.ServiceRoleKey,
	})
	if err != nil {
		log.Fatalf("supabase init failed: %v", err)
	}

	uploadsSvc := uploads.NewService(cld, supa, cfg.Supabase.DocsBucket)
	uploadsHandler := uploads.NewHandler(uploadsSvc)

	r := chi.NewRouter()

	origins := splitAndTrim(cfg.AllowedOrigins, ",")
	for _, o := range origins {
		if o == "*" {
			log.Fatal("CORS: '*' is not allowed in ALLOWED_ORIGINS when cookies are used (AllowCredentials: true)")
		}
	}
	if len(origins) == 0 {
		log.Println("CORS: ALLOWED_ORIGINS not set — cross-origin requests will be blocked")
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	auth.RegisterRoutes(r, authHandler, authMW)
	university.RegisterRoutes(r, universityHandler, authMW, adminMW, optionalAuthMW)
	uploads.RegisterRoutes(r, uploadsHandler, authMW, adminOrRepMW)
	college.RegisterRoutes(r, collegeHandler, authMW, adminOrRepMW, optionalAuthMW)
	favorites.RegisterRoutes(r, favoritesHandler, authMW, studentMW)
	claim.RegisterRoutes(r, claimHandler, authMW, adminMW, optionalAuthMW)

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
