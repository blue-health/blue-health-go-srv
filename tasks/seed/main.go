package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/blue-health/blue-health-go-srv/app/storage"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type (
	config struct {
		Delay    time.Duration `split_words:"true" default:"6s"`
		Database struct {
			DSN string `required:"true"`
		}
		Secrets struct {
			DatabaseKey string `split_words:"true" required:"true"`
		}
	}

	seeds struct{}
)

const app = "blue_health_go_srv_seed"

//go:embed data.yaml
var seedFs embed.FS

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	ctx := context.Background()

	var cfg config
	if err := envconfig.Process(app, &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	time.Sleep(cfg.Delay)

	if err := run(ctx, &cfg); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
}

func run(ctx context.Context, cfg *config) error {
	f, err := seedFs.ReadFile("data.yaml")
	if err != nil {
		log.Fatalf("failed to read data.yaml: %v", err)
	}

	var s seeds
	if err = yaml.Unmarshal(f, &s); err != nil {
		log.Fatalf("failed to unmarshal data.yaml: %v", err)
	}

	db, err := storage.NewPostgres(ctx, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			log.Fatalf("failed to close the database: %v", err)
		}
	}()

	var c int
	if err := db.GetContext(ctx, &c, "select count(id) from cards"); err != nil {
		return fmt.Errorf("failed to query card number: %w", err)
	}

	if c > 0 {
		return nil
	}

	var (
		txManager = storage.NewTransactionManager(db)
	)

	tx := txManager.MustBegin(ctx)

	// Insert models

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
