package state

import (
	_ "embed"
	"void/types"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Services types.Document
	Logger   *zap.SugaredLogger
)

// Services is set by main.go, we need to init everything else here
func init() {
	// lumberjack.Logger is already safe for concurrent use, so we don't need to
	// lock it.
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/var/log/void.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.DebugLevel,
	)

	Logger = zap.New(core).Sugar()
}
