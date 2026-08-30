package taskservice

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ShutdownTimeout:   3 * time.Second,
	}
}

func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	return ServeWithConfig(ctx, listener, handler, DefaultServerConfig())
}

func ServeWithConfig(ctx context.Context, listener net.Listener, handler http.Handler, config ServerConfig) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}

	shutdownSignal, stopShutdown := context.WithCancel(ctx)
	defer stopShutdown()

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-shutdownSignal.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), config.ShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := server.Serve(listener)
	stopShutdown()
	<-stopped
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return err
}
