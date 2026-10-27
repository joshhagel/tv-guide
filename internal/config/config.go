package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHDHRBaseURL       = "http://10.0.30.99"
	defaultPort              = "8080"
	defaultRefreshMinHrs     = 20
	defaultRefreshMaxHrs     = 28
	defaultCachePath         = "cache/xmltv.xml"
	defaultHTTPClientTimeout = 2 * time.Minute
)

type Config struct {
	HDHRBaseURL       string
	Port              string
	RefreshMin        time.Duration
	RefreshMax        time.Duration
	FavoritesOnly     bool
	CachePath         string
	HTTPClientTimeout time.Duration
}

func Load() Config {
	cfg := Config{
		HDHRBaseURL:       envOr("HDHR_BASE_URL", defaultHDHRBaseURL),
		Port:              envOr("PORT", defaultPort),
		RefreshMin:        hoursEnv("REFRESH_MIN_HOURS", defaultRefreshMinHrs),
		RefreshMax:        hoursEnv("REFRESH_MAX_HOURS", defaultRefreshMaxHrs),
		CachePath:         envOr("CACHE_PATH", defaultCachePath),
		HTTPClientTimeout: durationEnv("HTTP_CLIENT_TIMEOUT", defaultHTTPClientTimeout),
	}
	if cfg.RefreshMax < cfg.RefreshMin {
		cfg.RefreshMax = cfg.RefreshMin
	}
	cfg.FavoritesOnly = envBool("FAVORITES_ONLY")
	return cfg
}

func (c Config) ListenAddr() string {
	return fmt.Sprintf(":%s", c.Port)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hoursEnv(key string, fallback int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(fallback) * time.Hour
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return time.Duration(fallback) * time.Hour
	}
	return time.Duration(n) * time.Hour
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func envBool(key string) bool {
	v := os.Getenv(key)
	switch v {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	default:
		return false
	}
}
