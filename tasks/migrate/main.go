package main

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type config struct {
	Delay     time.Duration `default:"2s"`
	TouchFile string        `split_words:"true"`
	Database  struct {
		DSN string `required:"true"`
	}
}

const app = "blue_health_go_srv_migrate"

//go:embed sql/*.sql
var migrationsFs embed.FS

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	var cfg config
	if err := envconfig.Process(app, &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	time.Sleep(cfg.Delay)

	if err := run(&cfg); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
}

func run(cfg *config) error {
	if cfg.TouchFile != "" {
		defer func() {
			if err := touch(cfg.TouchFile); err != nil {
				log.Errorf("failed to touch file: %v", err)
			}
		}()
	}

	d, err := iofs.New(migrationsFs, "sql")
	if err != nil {
		return fmt.Errorf("failed to create iofs: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	return nil
}

func touch(name string) error {
	f, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}

	return f.Close()
}
