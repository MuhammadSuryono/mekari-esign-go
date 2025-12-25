package server

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"mekari-esign/internal/config"
	"mekari-esign/internal/delivery/http/router"
)

func NewServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	r *router.Router,
	logger *zap.Logger,
) error {
	app := r.Setup()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%d", cfg.App.Port)
			logger.Info("Starting HTTP server",
				zap.String("address", addr),
				zap.String("env", cfg.App.Env),
			)

			go func() {
				if err := app.Listen(addr); err != nil {
					logger.Error("Failed to start server", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down HTTP server")
			return app.Shutdown()
		},
	})

	return nil
}
