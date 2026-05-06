package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-playground/apps/gateway-api/internal/config"
	"order-playground/apps/gateway-api/internal/grpcclients"
	"order-playground/apps/gateway-api/internal/httpapi"
	"order-playground/apps/gateway-api/internal/httpapi/handlers"
	"order-playground/apps/gateway-api/internal/httpapi/middleware"
	"order-playground/apps/gateway-api/internal/mongorepo"
	"order-playground/apps/gateway-api/internal/mysqlrepo"
	"order-playground/apps/gateway-api/internal/service"
	"order-playground/apps/gateway-api/internal/temporalapp"
	"order-playground/pkg/transport/grpcjson"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config) *App {
	return &App{
		cfg:    cfg,
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (a *App) Run(ctx context.Context) error {
	grpcjson.Register()

	db, err := sql.Open("mysql", a.cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(a.cfg.MySQLMaxOpenConns)
	db.SetMaxIdleConns(a.cfg.MySQLMaxIdleConns)
	db.SetConnMaxLifetime(a.cfg.MySQLConnLifetime)
	db.SetConnMaxIdleTime(a.cfg.MySQLConnIdleTime)

	mongoClient, err := mongo.Connect(options.Client().
		ApplyURI(a.cfg.MongoURI).
		SetMaxPoolSize(a.cfg.MongoMaxPoolSize).
		SetMinPoolSize(a.cfg.MongoMinPoolSize).
		SetMaxConnIdleTime(a.cfg.MongoMaxIdleTime))
	if err != nil {
		return err
	}
	defer mongoClient.Disconnect(ctx)
	mongoDB := mongoClient.Database(a.cfg.MongoDatabase)

	tokenVerifier, err := middleware.NewTokenVerifier(ctx, a.cfg.KeycloakIssuerURL, a.cfg.KeycloakJWKSBase, a.cfg.KeycloakClientID)
	if err != nil {
		return err
	}

	ordersRepo := mysqlrepo.New(db)
	auditRepo := mongorepo.New(mongoDB)

	inventoryClient, inventoryConn, err := grpcclients.NewInventoryClient(a.cfg.InventoryGRPCAddr)
	if err != nil {
		return err
	}
	defer inventoryConn.Close()

	notifierClient, notifierConn, err := grpcclients.NewNotifierClient(a.cfg.NotifierGRPCAddr)
	if err != nil {
		return err
	}
	defer notifierConn.Close()

	temporalClient, err := a.dialTemporalWithRetry(ctx)
	if err != nil {
		return err
	}
	defer temporalClient.Close()

	activities := &temporalapp.Activities{
		Orders:    ordersRepo,
		Audit:     auditRepo,
		Inventory: inventoryClient,
		Notifier:  notifierClient,
		Logger:    a.logger,
	}
	worker := temporalapp.NewWorker(temporalClient, a.cfg.TemporalTaskQueue, activities)
	if err := a.startWorkerWithRetry(ctx, worker); err != nil {
		return err
	}
	defer worker.Stop()

	orderService := service.NewOrderService(
		ordersRepo,
		auditRepo,
		temporalapp.NewStarter(temporalClient, a.cfg.TemporalTaskQueue),
		time.Now,
		func() string { return uuid.NewString() },
	)

	handler := handlers.New(orderService, "v1")
	router := httpapi.NewRouter(
		middleware.Logger(a.logger, auditRepo),
		tokenVerifier,
		handler,
		func() error {
			if err := ordersRepo.Ping(ctx); err != nil {
				return fmt.Errorf("mysql not ready: %w", err)
			}
			if err := auditRepo.Ping(ctx); err != nil {
				return fmt.Errorf("mongo not ready: %w", err)
			}
			return nil
		},
		a.cfg.RequestTimeout,
	)

	server := &http.Server{
		Addr:              ":" + a.cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting gateway api", "port", a.cfg.HTTPPort)
		errCh <- server.ListenAndServe()
	}()

	stopCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-stopCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func (a *App) dialTemporalWithRetry(ctx context.Context) (client.Client, error) {
	var lastErr error
	for attempt := range 12 {
		temporalClient, err := client.Dial(client.Options{
			HostPort: a.cfg.TemporalAddress,
		})
		if err == nil {
			return temporalClient, nil
		}
		lastErr = err
		a.logger.Warn("temporal not ready yet", "attempt", attempt+1, "error", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return nil, fmt.Errorf("connect temporal after retries: %w", lastErr)
}

func (a *App) startWorkerWithRetry(ctx context.Context, worker worker.Worker) error {
	var lastErr error
	for attempt := range 12 {
		err := worker.Start()
		if err == nil {
			return nil
		}
		lastErr = err
		a.logger.Warn("temporal worker not ready yet", "attempt", attempt+1, "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return fmt.Errorf("start temporal worker after retries: %w", lastErr)
}
