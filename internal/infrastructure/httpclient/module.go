package httpclient

import (
	"go.uber.org/fx"

	"mekari-esign/internal/infrastructure/nav"
)

// provideNAVAPILogSender wraps nav.Client as NAVAPILogSender interface
func provideNAVAPILogSender(client *nav.Client) NAVAPILogSender {
	return client
}

var Module = fx.Module("httpclient",
	fx.Provide(NewHTTPClient),
	fx.Provide(provideNAVAPILogSender),
)
