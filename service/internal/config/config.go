package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppName             string
	Env                 string
	Port                string
	DatabaseURL         string
	CoreAPIBase         string
	AllowedOrigins      string
	AdminUsers          string
	DesertCooldownHours int
	WorkerTickSeconds   int
	CraftMinutes        int
	AdminGetSeed        bool
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		AppName:             getenv("APP_NAME", "berjis-conquer"),
		Env:                 getenv("APP_ENV", "development"),
		Port:                getenv("PORT", "8087"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5440/berjis_conquer?sslmode=disable"),
		CoreAPIBase:         getenv("CORE_API_BASE", "http://localhost:8080"),
		AllowedOrigins:      getenv("ALLOWED_ORIGINS", "*"),
		AdminUsers:          getenv("ADMIN_USER_IDS", ""),
		DesertCooldownHours: getenvInt("DESERT_COOLDOWN_HOURS", 24),
		WorkerTickSeconds:   getenvInt("WORKER_TICK_SECONDS", 10),
		CraftMinutes:        getenvInt("CRAFT_MINUTES", 10),
		AdminGetSeed:        getenvBool("ADMIN_GET_SEED", false),
	}
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if v == "1" || v == "true" || v == "TRUE" || v == "yes" {
			return true
		}
		if v == "0" || v == "false" || v == "FALSE" || v == "no" {
			return false
		}
	}
	return def
}
