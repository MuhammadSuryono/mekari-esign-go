package repository

import "go.uber.org/fx"

var Module = fx.Module("repository",
	fx.Provide(NewEsignRepository),
	fx.Provide(NewOAuthRepository),
	fx.Provide(NewAPILogRepository),
)
