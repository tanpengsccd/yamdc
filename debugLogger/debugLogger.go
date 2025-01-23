package debugLogger

import (
	"sync"
	"yamdc/config"

	"github.com/xxxsen/common/logger"
	"go.uber.org/zap"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

func Shared() *zap.Logger {
	if globalLogger == nil {

		once.Do(func() {
			c := config.Shared()
			globalLogger = logger.Init(c.LogConfig.File, c.LogConfig.Level, int(c.LogConfig.FileCount), int(c.LogConfig.FileSize), int(c.LogConfig.KeepDays), c.LogConfig.Console)

		})
	}
	return globalLogger
}
