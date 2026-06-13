package main

import (
	"fmu-backend/internal/config"
	"fmu-backend/internal/database"
	"log"
	"net/http"

	"github.com/joho/godotenv"
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

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Println("server running on port", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
