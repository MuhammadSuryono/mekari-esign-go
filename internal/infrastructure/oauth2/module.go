package oauth2

import "go.uber.org/fx"

var Module = fx.Module("oauth2",
	fx.Provide(NewTokenService),
)
