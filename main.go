package main

import (
	"log"
	"os"
	"time"

	"ecotracker/cmd/server"
	"ecotracker/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	// ✅ Set timezone to Asia/Jakarta
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: Could not load Asia/Jakarta timezone: %v. Using UTC.", err)
	} else {
		time.Local = loc
		log.Println("✓ Timezone set to Asia/Jakarta (WIB)")
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize config
	cfg := &config.Config{
		Port:                   getEnv("PORT", "8080"),
		DBURL:                  os.Getenv("DB_URL"),  // ✅ Fixed: DBUrl → DBURL
		JWTSecret:              os.Getenv("JWT_SECRET"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		StorageBucket:          os.Getenv("STORAGE_BUCKET"),
	}

	// Initialize database
	db, err := config.NewDBPool(cfg.DBURL)  // ✅ Fixed: DBUrl → DBURL
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Start server
	srv := server.New(cfg, db)
	log.Printf("🌱 EcoTracker API starting on port %s", cfg.Port)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}