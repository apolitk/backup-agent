package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port         string
	Token        string
	TLSCert      string
	TLSKey       string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Karboii integration
	KarboiiEndpoint  string
	KarboiiToken     string
	ProjectID        string
	PollingInterval  time.Duration

	// S3
	S3Endpoint  string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3Bucket    string
	S3Timeout   time.Duration

	// Storage
	TempDir string

	// Worker pool
	MaxWorkers int

	// Logging
	LogLevel string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("AGENT_PORT", "8080"),
		Token:        getEnv("AGENT_TOKEN", ""),
		TLSCert:      getEnv("AGENT_TLS_CERT", ""),
		TLSKey:       getEnv("AGENT_TLS_KEY", ""),
		ReadTimeout:  getDuration("AGENT_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getDuration("AGENT_WRITE_TIMEOUT", 30*time.Second),

		KarboiiEndpoint: getEnv("KARBOII_ENDPOINT", ""),
		KarboiiToken:    getEnv("KARBOII_TOKEN", ""),
		ProjectID:       getEnv("PROJECT_ID", ""),
		PollingInterval: getDuration("POLLING_INTERVAL", 30*time.Second),

		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3Region:    getEnv("S3_REGION", ""),
		S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		S3UseSSL:    getBool("S3_USE_SSL", true),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		S3Timeout:   getDuration("S3_TIMEOUT", 5*time.Minute),

		TempDir:    getEnv("TEMP_DIR", "/tmp"),
		MaxWorkers: getInt("MAX_WORKERS", 4),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, _ := strconv.ParseBool(val)
	return b
}

func getInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, _ := strconv.Atoi(val)
	return i
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, _ := time.ParseDuration(val)
	return d
}