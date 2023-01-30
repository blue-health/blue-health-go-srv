package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/getsentry/sentry-go"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type config struct {
	Delay     time.Duration `default:"2s"`
	TouchFile string        `split_words:"true"`
	Debug     bool          `default:"false"`
	Project   struct {
		ID string `required:"true"`
	}
	Database struct {
		DSN string `required:"true"`
	}
	App struct {
		Environment string `default:"develop"`
		Version     string `default:"unknown"`
	}
	Sentry struct {
		DSN     string        `required:"true"`
		Timeout time.Duration `default:"30s"`
	}
}

const app = "billing_srv_migrate"

//go:embed sql/*.sql
var migrationsFs embed.FS

func main() {
	ctx := context.Background()

	var cfg config
	if err := envconfig.Process(app, &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	time.Sleep(cfg.Delay)

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.Sentry.DSN,
		Debug:       cfg.Debug,
		Environment: cfg.App.Environment,
		Release:     cfg.App.Version,
	})
	if err != nil {
		log.Fatalf("failed to initialize sentry: %v", err)
	}

	var opts []option.ClientOption
	if cfg.Debug {
		opts = append(opts,
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
	}

	c, err := logging.NewClient(ctx, cfg.Project.ID, opts...)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("failed to setup logging client: %v", err)
	}

	defer func() {
		sentry.Flush(cfg.Sentry.Timeout)

		if err := c.Close(); err != nil {
			sentry.CaptureException(err)
			log.Fatalf("failed to close logging client: %v", err)
		}
	}()

	l := c.Logger(app, logging.RedirectAsJSON(os.Stdout))

	if err := run(&cfg, l); err != nil {
		sentry.CaptureException(err)
		l.StandardLogger(logging.Critical).Fatalf("failed to migrate: %v", err)
	}
}

func run(cfg *config, lg *logging.Logger) error {
	if cfg.TouchFile != "" {
		defer func() {
			if err := touch(cfg.TouchFile); err != nil {
				sentry.CaptureException(err)
				lg.StandardLogger(logging.Critical).Printf("failed to touch file: %v", err)
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
