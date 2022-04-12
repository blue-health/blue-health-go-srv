package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blue-health/blue-health-go-srv/app/secret"
	"github.com/blue-health/blue-health-go-srv/app/storage"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/hellofresh/health-go/v4"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type input struct {
	Debug   bool `default:"false"`
	Servers struct {
		Addresses struct {
			Public        string `default:":8080"`
			Private       string `default:":8081"`
			Observability string `default:":9090"`
		}
		ReadTimeout     time.Duration `split_words:"true" default:"5s"`
		WriteTimeout    time.Duration `split_words:"true" default:"10s"`
		IdleTimeout     time.Duration `split_words:"true" default:"15s"`
		ShutdownTimeout time.Duration `split_words:"true" default:"30s"`
		RequestTimeout  time.Duration `split_words:"true" default:"45s"`
	}
	Secrets struct {
		RedisKey          string `split_words:"true" required:"true"`
		RedisCertificate  string `split_words:"true"`
		DatabaseKey       string `split_words:"true" required:"true"`
		IDTokenKeyID      string `split_words:"true" required:"true"`
		IDTokenPublicKey  string `split_words:"true" required:"true"`
		IDTokenPrivateKey string `split_words:"true" required:"true"`
		OpaqueTokenKey    string `split_words:"true" required:"true"`
	}
	Project struct {
		ID string `required:"true"`
	}
	Database struct {
		DSN string `required:"true"`
	}
}

var (
	healthy              int32
	app                  = "blue_health_go_srv"
	errShutdown          = errors.New("shutdown in progress")
	errTooManyGoroutines = errors.New("too many goroutines")
)

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

	time.Sleep(time.Second)

	db, err := storage.NewPostgres(ctx, i.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			log.Fatalf("failed to close the database: %v", err)
		}
	}()

	var secretSource secret.Source

	if i.Debug {
		secretSource = secret.NewEnvSource()
	} else {
		gsm, gerr := secret.NewGoogleSecretManager(ctx, i.Project.ID)
		if gerr != nil {
			//nolint:gocritic //begone
			log.Fatalf("failed to connect to gsm: %v", gerr)
		}

		defer gsm.Close()

		secretSource = gsm
	}

	fmt.Println(secretSource)

	checks, err := newHealthChecks(db)
	if err != nil {
		log.Fatalf("failed to setup healthz: %v", err)
	}

	var (
		warm sync.WaitGroup
		done = make(chan struct{})
		quit = make(chan os.Signal, 1)
	)

	warm.Add(3)

	var (
		publicServer        = newServer(&i, i.Servers.Addresses.Public, func(m chi.Router) {})
		privateServer       = newServer(&i, i.Servers.Addresses.Private, func(m chi.Router) {})
		observabilityServer = newServer(&i, i.Servers.Addresses.Observability, func(m chi.Router) {
			m.Mount("/livez", checks[0])
			m.Mount("/readyz", checks[1])
		})
		runServer = func(server *http.Server) {
			warm.Done()
			log.Println("starting server at", server.Addr)

			if errs := server.ListenAndServe(); errs != nil && errs != http.ErrServerClosed {
				log.Fatalf("failed to start server at %s: %v", server.Addr, errs)
			}
		}
	)

	go runServer(publicServer)
	go runServer(privateServer)
	go runServer(observabilityServer)

	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Println("server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, i.Servers.ShutdownTimeout)
		defer cancel()

		publicServer.SetKeepAlivesEnabled(false)
		privateServer.SetKeepAlivesEnabled(false)
		observabilityServer.SetKeepAlivesEnabled(false)

		if err := publicServer.Shutdown(ctx); err != nil {
			log.Fatalf("failed to gracefully shutdown public server: %v", err)
		}

		if err := privateServer.Shutdown(ctx); err != nil {
			log.Fatalf("failed to gracefully shutdown private server: %v", err)
		}

		if err := observabilityServer.Shutdown(ctx); err != nil {
			log.Fatalf("failed to gracefully shutdown observability server: %v", err)
		}

		close(done)
	}()

	warm.Wait()

	atomic.StoreInt32(&healthy, 1)

	<-done

	log.Println("server stopped")
}

const maxGoroutines = 1000

func newHealthChecks(db *sqlx.DB) ([2]http.Handler, error) {
	l, err := health.New(health.WithChecks(
		health.Config{
			Name:    "goroutine",
			Timeout: time.Second * 5,
			Check: func(ctx context.Context) error {
				if runtime.NumGoroutine() > maxGoroutines {
					return errTooManyGoroutines
				}

				return nil
			},
		},
	))

	if err != nil {
		return [2]http.Handler{}, fmt.Errorf("failed to set up health checks: %w", err)
	}

	r, err := health.New(health.WithChecks(
		health.Config{
			Name:    "database",
			Timeout: time.Second * 5,
			Check: func(ctx context.Context) error {
				if errp := db.PingContext(ctx); errp != nil {
					return err
				}

				if _, erre := db.ExecContext(ctx, `select version()`); err != nil {
					return erre
				}

				return nil
			}},
		health.Config{
			Name:    "shutdown",
			Timeout: time.Second,
			Check: func(ctx context.Context) error {
				if atomic.LoadInt32(&healthy) == 0 {
					return errShutdown
				}

				return nil
			},
		},
	))

	if err != nil {
		return [2]http.Handler{}, fmt.Errorf("failed to set up health checks: %w", err)
	}

	return [2]http.Handler{l.Handler(), r.Handler()}, nil
}

func newServer(cfg *input, address string, f func(chi.Router)) *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Servers.RequestTimeout))

	f(r)

	return &http.Server{
		Addr:         address,
		ReadTimeout:  cfg.Servers.ReadTimeout,
		WriteTimeout: cfg.Servers.WriteTimeout,
		IdleTimeout:  cfg.Servers.IdleTimeout,
		Handler:      r,
	}
}
