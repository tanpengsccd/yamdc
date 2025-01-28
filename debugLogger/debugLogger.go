package debugLogger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	globalLogger *zap.SugaredLogger
	once         sync.Once
)

func Shared() *zap.SugaredLogger {
	if globalLogger == nil {

		once.Do(func() {
			globalLogger = zap.NewNop().Sugar()
		})
	}
	return globalLogger
}
