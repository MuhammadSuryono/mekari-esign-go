package document

import "go.uber.org/fx"

var Module = fx.Module("document",
	fx.Provide(NewDocumentService),
)
