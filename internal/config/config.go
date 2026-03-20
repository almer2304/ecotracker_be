package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Supabase SupabaseConfig
	CORS     CORSConfig
	Worker   WorkerConfig
	Bcrypt   BcryptConfig
}

type AppConfig struct {
	Env         string
	Port        string
	Name        string
	AdminSecret string
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type SupabaseConfig struct {
	URL           string
	Key           string
	BucketPickups string
	BucketReports string
	BucketAvatars string
}

type CORSConfig struct {
	AllowedOrigins string
}

type WorkerConfig struct {
	TimeoutCheckInterval time.Duration
	AssignmentTimeout    time.Duration
}

type BcryptConfig struct {
	Cost int
}

// Load reads config from environment variables (after loading .env)
func Load() (*Config, error) {
	// Load .env file (ignore error in production where env vars are set directly)
	_ = godotenv.Load()

	cfg := &Config{}

	// App
	cfg.App = AppConfig{
		Env:  getEnv("APP_ENV", "development"),
		Port: getEnv("APP_PORT", "8080"),
		Name: getEnv("APP_NAME", "EcoTracker"),
	}

	// Database
	maxOpen, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdle, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	connLifetime, err := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	if err != nil {
		connLifetime = 5 * time.Minute
	}

	cfg.DB = DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", ""),
		Name:            getEnv("DB_NAME", "ecotracker"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: connLifetime,
	}

	// Redis
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	cfg.Redis = RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       redisDB,
	}

	// JWT
	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		refreshExpiry = 7 * 24 * time.Hour
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET wajib diisi")
	}

	cfg.JWT = JWTConfig{
		Secret:        jwtSecret,
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}

	// Supabase
	cfg.Supabase = SupabaseConfig{
		URL:           getEnv("SUPABASE_URL", ""),
		Key:           getEnv("SUPABASE_KEY", ""),
		BucketPickups: getEnv("SUPABASE_BUCKET_PICKUPS", "pickup-photos"),
		BucketReports: getEnv("SUPABASE_BUCKET_REPORTS", "report-photos"),
		BucketAvatars: getEnv("SUPABASE_BUCKET_AVATARS", "avatars"),
	}

	// CORS
	cfg.CORS = CORSConfig{
		AllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
	}

	// Worker
	timeoutCheck, err := time.ParseDuration(getEnv("TIMEOUT_CHECK_INTERVAL", "60s"))
	if err != nil {
		timeoutCheck = 60 * time.Second
	}
	assignmentTimeout, err := time.ParseDuration(getEnv("ASSIGNMENT_TIMEOUT", "15m"))
	if err != nil {
		assignmentTimeout = 15 * time.Minute
	}

	cfg.Worker = WorkerConfig{
		TimeoutCheckInterval: timeoutCheck,
		AssignmentTimeout:    assignmentTimeout,
	}

	// Bcrypt
	bcryptCost, _ := strconv.Atoi(getEnv("BCRYPT_COST", "12"))
	cfg.Bcrypt = BcryptConfig{Cost: bcryptCost}

	// Admin Secret
	cfg.App.AdminSecret = getEnv("ADMIN_SECRET", "ecotracker-admin-secret-2026")

	return cfg, nil
}

// DSN returns PostgreSQL connection string
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}