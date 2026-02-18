package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	DBURL                 string
	SupabaseURL           string
	SupabaseServiceRoleKey string
	JWTSecret             string
	StorageBucket         string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		Port:                  getEnv("PORT", "8080"),
		DBURL:                 getEnv("DB_URL", ""),
		SupabaseURL:           getEnv("SUPABASE_URL", ""),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		JWTSecret:             getEnv("JWT_SECRET", "change-me-in-production"),
		StorageBucket:         getEnv("STORAGE_BUCKET", "pickups"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
