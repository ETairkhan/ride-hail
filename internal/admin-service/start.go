package adminservice

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"ride-hail/internal/admin-service/adapters/operator"

	"ride-hail/internal/config"
	"ride-hail/internal/logger"
)

func Execute(ctx context.Context, mylog logger.Logger, cfg *config.Config) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	server := operator.NewServer(newCtx, ctx, mylog, cfg)

	// Run server in goroutine
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- server.Run()
	}()

	// Wait for signal or server crash
	select {
	case <-newCtx.Done():
		mylog.Action("shutdown_signal_received").Info("Shutdown signal received")
		return server.Stop(context.Background())
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			mylog.Action("order_service_failed").Error("Server failed unexpectedly", err)
			return err
		}
		mylog.Action("server_stopped").Info("Server exited normally")
		return nil
	}
}
