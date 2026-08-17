package config

import (
	"os"
	"strings"
)

type Config struct {
	DataProvider  string // "seed" or "fmp"
	FMPApiKey     string
	FinnhubApiKey string
	LogLevel      string
	Port          string
}

func Load() *Config {
	c := &Config{
		DataProvider:  strings.ToLower(strings.TrimSpace(os.Getenv("DATA_PROVIDER"))),
		FMPApiKey:     strings.TrimSpace(os.Getenv("FMP_API_KEY")),
		FinnhubApiKey: strings.TrimSpace(os.Getenv("FINNHUB_API_KEY")),
		LogLevel:      strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))),
		Port:          strings.TrimSpace(os.Getenv("PORT")),
	}
	if c.DataProvider == "" {
		c.DataProvider = "seed"
	}
	if c.DataProvider != "seed" && c.DataProvider != "fmp" {
		c.DataProvider = "seed"
	}
	if c.DataProvider == "fmp" && c.FMPApiKey == "" {
		c.DataProvider = "seed"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return c
}
