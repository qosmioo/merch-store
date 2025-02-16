package logger

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

func InitLogger() *zap.Logger {
	var err error
	Logger, err = zap.NewProduction()
	if err != nil {
		return zap.NewNop()
	}
	return Logger
}
