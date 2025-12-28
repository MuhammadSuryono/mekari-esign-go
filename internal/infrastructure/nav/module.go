package nav

import "go.uber.org/fx"

var Module = fx.Provide(NewClient)
