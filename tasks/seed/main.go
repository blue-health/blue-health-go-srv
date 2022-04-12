package main

import (
	"context"
	"embed"
	"os"
	"time"

	"github.com/blue-health/blue-health-go-srv/app/storage"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type (
	input struct {
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

	var i input
	if err := envconfig.Process(app, &i); err != nil {
		log.Fatalf("failed to load input: %v", err)
	}

	f, err := seedFs.ReadFile("data.yaml")
	if err != nil {
		log.Fatalf("failed to read data.yaml: %v", err)
	}

	var s seeds
	if err = yaml.Unmarshal(f, &s); err != nil {
		log.Fatalf("failed to unmarshal data.yaml: %v", err)
	}

	time.Sleep(i.Delay)

	db, err := storage.NewPostgres(ctx, i.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			log.Fatalf("failed to close the database: %v", err)
		}
	}()

	var c int
	if err := db.GetContext(ctx, &c, "select count(id) from cards"); err != nil {
		log.Errorf("failed to query card number: %v", err)
		return
	}

	if c > 0 {
		return
	}

	var (
		txManager = storage.NewTransactionManager(db)
	)

	tx := txManager.MustBegin(ctx)

	// Insert models

	if err := tx.Commit(); err != nil {
		log.Errorf("failed to commit transaction: %v", err)
	}
}
