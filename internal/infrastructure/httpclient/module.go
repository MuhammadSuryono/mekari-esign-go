package httpclient

import "go.uber.org/fx"

var Module = fx.Module("httpclient",
	fx.Provide(NewHTTPClient),
)
