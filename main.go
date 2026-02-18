package main

import (
	"log"

	"ecotracker/cmd/server"
	"ecotracker/internal/config"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	if cfg.DBURL == "" {
		log.Fatal("DB_URL is required. Please set it in .env file")
	}

	// Connect to PostgreSQL (Supabase)
	db, err := config.NewDBPool(cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Build and start the server
	srv := server.New(cfg, db)

	log.Printf("🌱 EcoTracker API starting on port %s", cfg.Port)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
