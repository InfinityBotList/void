package state

import (
	_ "embed"
	"void/types"

	"go.uber.org/zap"
	"github.com/infinitybotlist/eureka/snippets"
)

var (
	Services types.Document
	Logger   *zap.SugaredLogger
)

// Services is set by main.go, we need to init everything else here
func Init() {
	Logger = snippets.CreateZap() 
}
