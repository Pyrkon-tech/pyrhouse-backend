package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server           ServerConfig
	Database         DatabaseConfig
	JWT              JWTConfig
	CORS             CORSConfig
	Discord          DiscordConfig
	Google           GoogleConfig
	EquipmentRequest EquipmentRequestConfig
}

type EquipmentRequestConfig struct {
	SheetID        string
	SheetName      string
	SyncEnabled    bool
	SyncInterval   time.Duration
	FuzzyThreshold int
}

type ServerConfig struct {
	Port           string
	RequestTimeout time.Duration
	Version        string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	FrontendURL  string // URL frontendu do przekierowania po logowaniu
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	FrontendURL  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnv("APP_PORT", "8080"),
			RequestTimeout: getDurationEnv("REQUEST_TIMEOUT_SECONDS", 0),
			Version:        getEnv("APP_VERSION", "1.0.0"),
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("DATABASE_URL"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME_MINUTES", 5) * time.Minute,
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME_MINUTES", 1) * time.Minute,
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			Expiration: getDurationEnv("JWT_EXPIRATION_HOURS", 120) * time.Hour,
		},
		CORS: CORSConfig{
			AllowedOrigins:   getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5000"}),
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With", "Cache-Control", "Last-Event-ID"},
			ExposedHeaders:   []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           getDurationEnv("CORS_MAX_AGE_HOURS", 12) * time.Hour,
		},
		Discord: DiscordConfig{
			ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
			ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("DISCORD_REDIRECT_URI"),
			FrontendURL:  os.Getenv("FRONTEND_URL"),
		},
		Google: GoogleConfig{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),
			FrontendURL:  os.Getenv("FRONTEND_URL"),
		},
		EquipmentRequest: EquipmentRequestConfig{
			SheetID:        os.Getenv("EQUIPMENT_REQUEST_SHEET_ID"),
			SheetName:      getEnv("EQUIPMENT_REQUEST_SHEET_NAME", "Zamówienia"),
			SyncEnabled:    getEnv("EQUIPMENT_REQUEST_SYNC_ENABLED", "false") == "true",
			SyncInterval:   parseDurationEnv("EQUIPMENT_REQUEST_SYNC_INTERVAL", 15*time.Minute),
			FuzzyThreshold: getIntEnv("EQUIPMENT_REQUEST_FUZZY_THRESHOLD", 3),
		},
	}

	return cfg, nil
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(intValue)
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
