package repository

import (
	"go.uber.org/fx"

	"mekari-esign/internal/infrastructure/httpclient"
	"mekari-esign/internal/infrastructure/nav"
)

var Module = fx.Module("repository",
	fx.Provide(NewEsignRepository),
	fx.Provide(NewOAuthRepository),
	fx.Provide(NewAPILogRepository),
	fx.Provide(
		fx.Annotate(
			func(repo APILogRepository) httpclient.APILogSaver { return repo },
			fx.From(new(APILogRepository)),
		),
	),
	fx.Provide(
		fx.Annotate(
			func(repo APILogRepository) nav.APILogSaver { return repo },
			fx.From(new(APILogRepository)),
		),
	),
)
