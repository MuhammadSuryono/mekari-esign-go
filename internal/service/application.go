package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"mekari-esign/internal/config"
	deliveryhttp "mekari-esign/internal/delivery/http"
	"mekari-esign/internal/infrastructure/database"
	"mekari-esign/internal/infrastructure/document"
	"mekari-esign/internal/infrastructure/httpclient"
	"mekari-esign/internal/infrastructure/logger"
	"mekari-esign/internal/infrastructure/oauth2"
	"mekari-esign/internal/infrastructure/redis"
	"mekari-esign/internal/infrastructure/repository"
	"mekari-esign/internal/server"
	"mekari-esign/internal/usecase"
)

// Application wraps the fx.App for service management
type Application struct {
	app      *fx.App
	ctx      context.Context
	cancel   context.CancelFunc
	doneChan chan struct{}
}

// NewApplication creates a new Application instance
func NewApplication() *Application {
	ctx, cancel := context.WithCancel(context.Background())
	return &Application{
		ctx:      ctx,
		cancel:   cancel,
		doneChan: make(chan struct{}),
	}
}

// Run starts the application
func (a *Application) Run() {
	a.app = fx.New(
		// Provide context
		fx.Provide(func() context.Context { return a.ctx }),

		// Configuration
		config.Module,

		// Infrastructure
		logger.Module,
		database.Module,
		redis.Module,
		oauth2.Module,
		document.Module,
		httpclient.Module,
		repository.Module,

		// Business Logic
		usecase.Module,

		// Delivery
		deliveryhttp.Module,

		// Server
		server.Module,
	)

	// Start the application
	if err := a.app.Start(a.ctx); err != nil {
		return
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		a.Shutdown()
	case <-a.ctx.Done():
		// Context was cancelled
	}

	close(a.doneChan)
}

// Shutdown gracefully shuts down the application
func (a *Application) Shutdown() {
	a.cancel()
	if a.app != nil {
		ctx, cancel := context.WithTimeout(context.Background(), fx.DefaultTimeout)
		defer cancel()
		a.app.Stop(ctx)
	}
}

// Wait blocks until the application exits
func (a *Application) Wait() {
	<-a.doneChan
}
