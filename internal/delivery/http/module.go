package http

import (
	"go.uber.org/fx"

	"mekari-esign/internal/delivery/http/handler"
	"mekari-esign/internal/delivery/http/router"
)

var Module = fx.Module("http",
	fx.Provide(
		handler.NewEsignHandler,
		handler.NewHealthHandler,
		handler.NewOAuthHandler,
		handler.NewWebhookHandler,
		handler.NewLogHandler,
		router.NewRouter,
	),
)
