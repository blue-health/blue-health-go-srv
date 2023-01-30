package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"cloud.google.com/go/logging"
	"github.com/blue-health/blue-go-toolbox/authn"
	"github.com/blue-health/blue-go-toolbox/crypto"
	"github.com/blue-health/blue-go-toolbox/logger"
	"github.com/blue-health/blue-go-toolbox/secret"
	"github.com/blue-health/blue-health-go-srv/app/cake"
	"github.com/blue-health/blue-health-go-srv/app/cookie"
	"github.com/blue-health/blue-health-go-srv/app/storage"
	"github.com/blue-health/blue-health-go-srv/app/web/private"
	"github.com/blue-health/blue-health-go-srv/app/web/public"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hellofresh/health-go/v4"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type config struct {
	Debug  bool
	Delay  time.Duration `default:"1s"`
	Server struct {
		Address struct {
			Public        string `default:":8080"`
			Private       string `default:":8081"`
			Observability string `default:":9090"`
		}
		ReadTimeout       time.Duration `split_words:"true" default:"15s"`
		WriteTimeout      time.Duration `split_words:"true" default:"30s"`
		IdleTimeout       time.Duration `split_words:"true" default:"120s"`
		ReadyTimeout      time.Duration `split_words:"true" default:"6s"`
		ShutdownTimeout   time.Duration `split_words:"true" default:"10s"`
		RequestTimeout    time.Duration `split_words:"true" default:"50s"`
		ReadHeaderTimeout time.Duration `split_words:"true" default:"5s"`
	}
	Project struct {
		ID string `required:"true"`
	}
	App struct {
		Environment string `default:"develop"`
		Version     string `default:"unknown"`
	}
	Database struct {
		DSN       string `required:"true"`
		MaxConns  int    `split_words:"true" default:"10"`
		CloudName string `split_words:"true" required:"true"`
	}
	Sentry struct {
		DSN     string        `required:"true"`
		Timeout time.Duration `default:"30s"`
	}
	Secrets struct {
		RedisKey string `split_words:"true" required:"true"`
	}
	JWKService struct {
		BaseURL string `split_words:"true" required:"true"`
	} `split_words:"true"`
}

var (
	ready int32
	app   = "blue_health_go_srv"
	// Errors
	errShutdown           = errors.New("shutdown in progress")
	errTooManyGoroutines  = errors.New("too many goroutines")
	errRedisMisconfigured = errors.New("redis is misconfigured")
	errCertificateInvalid = errors.New("failed to decode PEM certificate")
)

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
		log.Fatalf("failed to setup logging client: %v", err)
	}

	defer func() {
		sentry.Flush(cfg.Sentry.Timeout)

		if err := c.Close(); err != nil {
			log.Fatalf("failed to close logging client: %v", err)
		}
	}()

	l := c.Logger(app, logging.RedirectAsJSON(os.Stdout))

	if err := run(ctx, &cfg, l); err != nil {
		sentry.CaptureException(err)
		l.StandardLogger(logging.Critical).Printf("failed to run app: %v", err)
	}
}

func run(ctx context.Context, cfg *config, lg *logging.Logger) error {
	var (
		lgr     = logger.New(lg)
		infoLog = lg.StandardLogger(logging.Info)
		errLog  = lg.StandardLogger(logging.Critical)
	)

	var (
		db  *sqlx.DB
		err error
	)

	// Create database instance
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

	// Create Secret Manager
	var secretSource secret.Source
	if cfg.Debug {
		secretSource = secret.NewEnvSource()
	} else {
		secretSource, err = secret.NewGoogleSecretManager(ctx, cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to connect to gsm: %w", err)
		}
	}

	defer secretSource.Close()

	// Load example secret
	redisKey, err := secretSource.Get(ctx, cfg.Secrets.RedisKey)
	if err != nil {
		return fmt.Errorf("failed to fetch redis key: %w", err)
	}

	// Create AES cryptographic primitive
	redisAES, err := crypto.NewAES(redisKey)
	if err != nil {
		return fmt.Errorf("failed to create redis aes: %w", err)
	}

	// Create an AuthenticatioN policy for user requests
	authnPolicy, err := authn.NewPolicy(ctx, cfg.JWKService.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to create authn: %w", err)
	}

	// Just to show how to create Crypto primitives
	fmt.Println(redisAES)

	// Create health checks
	checks, err := newHealthChecks(db)
	if err != nil {
		return fmt.Errorf("failed to setup healthz: %w", err)
	}

	// Create Sentry handler (catches panics & 500s)
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
		Timeout: cfg.Sentry.Timeout,
	})

	var (
		warm sync.WaitGroup
		done = make(chan struct{})
		quit = make(chan os.Signal, 1)

		// Create domain services
		cakeService   = cake.NewService()
		cookieService = cookie.NewService()
	)

	warm.Add(3)

	var (
		// Create public server (for frontends / mobile apps)
		publicServer = newServer(ctx, cfg, cfg.Server.Address.Public, func(m chi.Router) {
			m.Mount("/", public.NewCakeService(authnPolicy, cakeService, lgr).Router())
		}, sentryHandler.Handle)
		// Create private server (for other in-cluster apps)
		privateServer = newServer(ctx, cfg, cfg.Server.Address.Private, func(m chi.Router) {
			m.Mount("/", private.NewCookieService(cookieService, lgr).Router())
		}, sentryHandler.Handle)
		// Create observability server (for Kubernetes)
		observabilityServer = newServer(ctx, cfg, cfg.Server.Address.Observability, func(m chi.Router) {
			m.Mount("/livez", checks[0])
			m.Mount("/readyz", checks[1])
		}, sentryHandler.Handle)

		runServer = func(server *http.Server) {
			warm.Done()
			infoLog.Println("starting server at", server.Addr)

			if errs := server.ListenAndServe(); errs != nil && errs != http.ErrServerClosed {
				sentry.CaptureException(err)
				errLog.Printf("failed to start server at %s: %v", server.Addr, errs)
			}
		}
	)

	go runServer(publicServer)
	go runServer(privateServer)
	go runServer(observabilityServer)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit

		infoLog.Println("server is shutting down...")
		atomic.StoreInt32(&ready, 0)

		publicServer.SetKeepAlivesEnabled(false)
		privateServer.SetKeepAlivesEnabled(false)
		observabilityServer.SetKeepAlivesEnabled(false)

		time.Sleep(cfg.Server.ReadyTimeout)

		c, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := publicServer.Shutdown(c); err != nil {
			sentry.CaptureException(err)
			errLog.Printf("failed to gracefully shutdown public server: %v", err)
		}

		if err := privateServer.Shutdown(c); err != nil {
			sentry.CaptureException(err)
			errLog.Printf("failed to gracefully shutdown private server: %v", err)
		}

		if err := observabilityServer.Shutdown(c); err != nil {
			sentry.CaptureException(err)
			errLog.Printf("failed to gracefully shutdown observability server: %v", err)
		}

		close(done)
	}()

	warm.Wait()

	atomic.StoreInt32(&ready, 1)

	infoLog.Println("app started")

	<-done

	infoLog.Println("app stopped")

	return nil
}

const maxGoroutines = 1000

func newHealthChecks(db *sqlx.DB) ([2]http.Handler, error) {
	l, err := health.New(health.WithChecks(
		health.Config{
			Name:    "goroutine",
			Timeout: time.Second * 5,
			Check: func(_ context.Context) error {
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
			Name:    "shutdown",
			Timeout: time.Second,
			Check: func(_ context.Context) error {
				if atomic.LoadInt32(&ready) == 0 {
					return errShutdown
				}

				return nil
			},
		},
		health.Config{
			Name:    "database",
			Timeout: time.Second * 5,
			Check: func(ctx context.Context) error {
				if errp := db.PingContext(ctx); errp != nil {
					return errp
				}

				if _, erre := db.ExecContext(ctx, `select version()`); erre != nil {
					return erre
				}

				return nil
			}},
	))

	if err != nil {
		return [2]http.Handler{}, fmt.Errorf("failed to set up health checks: %w", err)
	}

	return [2]http.Handler{l.Handler(), r.Handler()}, nil
}

func newServer(ctx context.Context, cfg *config, addr string, f func(chi.Router), m ...func(http.Handler) http.Handler) *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.RequestTimeout))

	for _, h := range m {
		r.Use(h)
	}

	f(r)

	return &http.Server{
		Addr:              addr,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		Handler:           r,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
}
