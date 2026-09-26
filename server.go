// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServerConfig struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
	CertFile          string
	KeyFile           string
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// StreamingServerConfig disables WriteTimeout so SSE and WebSocket handlers
// can hold the connection open. Prefer this (or ServerWithConfig) for realtime routes.
func StreamingServerConfig() ServerConfig {
	cfg := DefaultServerConfig()
	cfg.WriteTimeout = 0
	return cfg
}

func (a *App) Server(host string, port int) {
	_ = a.ListenAndServe(fmt.Sprintf("%s:%d", host, port), DefaultServerConfig())
}

func (a *App) ServerWithConfig(host string, port int, cfg ServerConfig) {
	_ = a.ListenAndServe(fmt.Sprintf("%s:%d", host, port), cfg)
}

func (a *App) ListenAndServe(addr string, cfg ServerConfig) error {
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           a.Handler(),
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
	a.httpServer = server

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("listening", "addr", addr, "tls", cfg.CertFile != "" && cfg.KeyFile != "")
		var err error
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			err = server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		a.logger.Info("shutdown signal", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := a.Shutdown(ctx); err != nil {
			return err
		}
		return <-errCh
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.httpServer == nil {
		return nil
	}
	a.logger.Info("shutting down")
	return a.httpServer.Shutdown(ctx)
}

func (a *App) HTTPServer() *http.Server {
	return a.httpServer
}
