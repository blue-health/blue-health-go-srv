package main

import (
	"embed"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type input struct {
	Delay     time.Duration `split_words:"true" default:"3s"`
	TouchFile string        `split_words:"true" default:""`
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
	var i input
	if err := envconfig.Process(app, &i); err != nil {
		log.Fatalf("failed to load input: %v", err)
	}

	d, err := iofs.New(migrationsFs, "sql")
	if err != nil {
		log.Fatalf("failed to create iofs: %v", err)
	}

	time.Sleep(i.Delay)

	m, err := migrate.NewWithSourceInstance("iofs", d, i.Database.DSN)
	if err != nil {
		log.Fatalf("failed to create migration source: %v", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to migrate: %v", err)
	}

	if i.TouchFile != "" {
		if err := touch(i.TouchFile); err != nil {
			log.Fatalf("failed to touch file: %v", err)
		}
	}
}

func touch(name string) error {
	f, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}

	return f.Close()
}
