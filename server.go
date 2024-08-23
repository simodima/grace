package grace

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type options struct {
	timeout time.Duration
	address string
	signals []os.Signal
}

type ServerOpt func(*options)

func WithShutdownTimeout(t time.Duration) ServerOpt {
	return func(o *options) {
		o.timeout = t
	}
}

func WithBindAddress(a string) ServerOpt {
	return func(o *options) {
		o.address = a
	}
}

func WithSignals(sigs ...os.Signal) ServerOpt {
	return func(o *options) {
		o.signals = append(o.signals, sigs...)
	}
}

func RunGracefully(router http.Handler, serverOptions ...ServerOpt) error {

	// ############## Options configutation
	opts := &options{
		timeout: 5 * time.Second,
		address: ":8080",
		signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}

	for _, o := range serverOptions {
		o(opts)
	}

	// ############## Server startup
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, opts.signals...)
	defer stop()

	ctx, cancel := context.WithCancelCause(ctx)

	srv := &http.Server{
		Addr:    opts.address,
		Handler: router,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cancel(err)
		}
	}()

	// Wait for signals or startup error
	<-ctx.Done()

	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return errors.Join(errors.New("failed to start servers"), err)
	}

	// ############## Shutdown the server
	slog.Info("Shutting down Server ...")

	ctx, cancelT := context.WithTimeout(context.Background(), opts.timeout)
	defer cancelT()

	var shutdownErr error

	//go func() {
	err := srv.Shutdown(ctx)
	if err != nil {
		shutdownErr = err
	}
	//  else {
	// 	cancelT()
	// }
	// }()

	if err := context.Cause(ctx); shutdownErr != nil || (err != nil && !errors.Is(err, context.Canceled)) {
		return errors.Join(errors.New("failed to shutdown gracefully"), shutdownErr, err)
	}

	slog.Info("Server exiting")
	return nil

	// select {
	// case err := <-shutdownErr:
	// 	slog.Error("Server Shutdown:", slog.Any("error", err))
	// case <-ctx.Done():
	// 	slog.Info("timeout of 5 seconds.")
	// }

}
