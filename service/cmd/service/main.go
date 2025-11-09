package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/berjistech/berjis-ecosystem/games/conquer/service/internal/config"
	"github.com/berjistech/berjis-ecosystem/games/conquer/service/internal/db"
	"github.com/berjistech/berjis-ecosystem/games/conquer/service/internal/migrate"
	"github.com/berjistech/berjis-ecosystem/games/conquer/service/internal/server"
	"github.com/berjistech/berjis-ecosystem/games/conquer/service/internal/worker"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Printf("warn: failed to connect to conquer DB: %v", err)
	} else {
		runner := migrate.Runner{Dir: "./migrations"}
		if err := runner.Up(conn); err != nil {
			log.Printf("warn: migrations failed: %v", err)
		}
	}

	app := server.New(server.Options{
		AllowedOrigins:      cfg.AllowedOrigins,
		DB:                  conn,
		CoreAPIBase:         cfg.CoreAPIBase,
		AdminUsers:          cfg.AdminUsers,
		DesertCooldownHours: cfg.DesertCooldownHours,
		CraftMinutes:        cfg.CraftMinutes,
		AdminGetSeed:        cfg.AdminGetSeed,
	})
	// Start background workers (craft completion)
	stop := worker.StartCraftWorker(conn, time.Duration(cfg.WorkerTickSeconds)*time.Second, log.Default())
	addr := ":" + cfg.Port
	log.Printf("starting %s on %s (env=%s)", cfg.AppName, addr, cfg.Env)
	if err := app.Listen(addr); err != nil {
		log.Println("shutdown:", err)
		close(stop)
		os.Exit(1)
	}
}
