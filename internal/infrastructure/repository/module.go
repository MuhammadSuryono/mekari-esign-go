package repository

import (
	"go.uber.org/fx"

	"mekari-esign/internal/infrastructure/httpclient"
)

var Module = fx.Module("repository",
	fx.Provide(NewEsignRepository),
	fx.Provide(NewOAuthRepository),
	fx.Provide(
		fx.Annotate(
			NewAPILogRepository,
			fx.As(new(httpclient.APILogSaver)),
		),
	),
)
