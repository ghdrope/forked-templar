package main

import (
	"os"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {

	// Setup a context that is automatically cancelled on SIGINT/SIGTERM.
	ctx := signals.SetupSignalHandler()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	zap.ReplaceGlobals(logger)

	if err := Execute(ctx); err != nil {
		zap.L().Error("fatal error during execution", zap.Error(err))
		os.Exit(1)
	}
}
