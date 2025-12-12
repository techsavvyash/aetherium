package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aetherium/aetherium/pkg/config"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
)

func main() {
	configPath := flag.String("config", "config/example.yaml", "Path to config file")
	migrationsPath := flag.String("migrations", "migrations", "Path to migrations directory")
	action := flag.String("action", "up", "Migration action: up or down")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create store
	store, err := postgres.NewStore(postgres.Config{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		Database:     cfg.Database.Database,
		SSLMode:      cfg.Database.SSLMode,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Get absolute path to migrations
	absPath, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	fullMigrationsPath := fmt.Sprintf("%s/%s", absPath, *migrationsPath)

	// Run migrations
	switch *action {
	case "up":
		if err := store.RunMigrations(fullMigrationsPath); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("✓ Migrations completed successfully")
	default:
		log.Fatalf("Unknown action: %s (use 'up' or 'down')", *action)
	}
}
