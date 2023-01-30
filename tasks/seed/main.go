package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/blue-health/blue-health-go-srv/app/cake"
	"github.com/blue-health/blue-health-go-srv/app/cookie"
	"github.com/blue-health/blue-health-go-srv/app/storage"
	"github.com/getsentry/sentry-go"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type (
	config struct {
		Delay   time.Duration `split_words:"true" default:"1s"`
		Debug   bool          `default:"false"`
		Project struct {
			ID string `required:"true"`
		}
		Database struct {
			DSN       string `required:"true"`
			MaxConns  int    `split_words:"true" default:"10"`
			CloudName string `split_words:"true" required:"true"`
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

	seeds struct {
		Cakes   []cake.Cake     `yaml:"cakes,flow"`
		Cookies []cookie.Cookie `yaml:"cookies,flow"`
	}
)

const app = "blue_health_srv_go_seed"

//go:embed data.yaml
var seedFs embed.FS

func main() {
	ctx := context.Background()

	var cfg config
	if err := envconfig.Process(app, &cfg); err != nil {
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

	time.Sleep(cfg.Delay)

	err = sentry.Init(sentry.ClientOptions{
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
		log.Fatalf("failed to setup logging client: %v", err)
	}

	defer func() {
		sentry.Flush(cfg.Sentry.Timeout)

		if err := c.Close(); err != nil {
			log.Fatalf("failed to close logging client: %v", err)
		}
	}()

	l := c.Logger(app, logging.RedirectAsJSON(os.Stdout))

	if err := run(ctx, &cfg, &s, l); err != nil {
		sentry.CaptureException(err)
		l.StandardLogger(logging.Critical).Fatalf("failed to run: %v", err)
	}
}

func run(ctx context.Context, cfg *config, s *seeds, lg *logging.Logger) error {
	errLog := lg.StandardLogger(logging.Critical)

	var (
		err error
		db  *sqlx.DB
	)

	if cfg.App.Environment == "local" {
		db, err = storage.NewPostgres(ctx, cfg.Database.DSN, cfg.Database.MaxConns)
		if err != nil {
			return fmt.Errorf("failed to connect to the database: %w", err)
		}
	} else {
		var cleanup func() error

		db, cleanup, err = storage.NewCloudPostgres(ctx, cfg.Database.DSN, cfg.Database.CloudName, cfg.Database.MaxConns)
		if err != nil {
			return fmt.Errorf("failed to connect to the database: %w", err)
		}

		defer func() {
			_ = cleanup()
		}()
	}

	defer func() {
		if err = db.Close(); err != nil {
			sentry.CaptureException(err)
			errLog.Printf("failed to close the database: %v", err)
		}
	}()

	// Seed database with loaded data
	fmt.Println(s)
	fmt.Println(db)

	return nil
}
