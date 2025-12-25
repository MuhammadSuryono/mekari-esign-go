package usecase

import "go.uber.org/fx"

var Module = fx.Module("usecase",
	fx.Provide(NewEsignUsecase),
	fx.Provide(NewOAuthUsecase),
	fx.Provide(NewWebhookUsecase),
)
