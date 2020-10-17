package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sod/internal/logging"
	"sod/internal/srvenv"
	"time"
)

type Server struct {
	addr     string
	listener net.Listener
	env      *srvenv.SrvEnv
}

func New(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	return &Server{
		addr:     addr,
		listener: listener,
	}, nil
}

func (s *Server) ServeHTTP(ctx context.Context, srv *http.Server) error {
	logger := logging.FromContext(ctx)
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		logger.Debugf("server.Serve: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()

		logger.Debugf("server.Serve: shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	if err := srv.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	logger.Debugf("server.Serve: serving stopped")

	select {
	case err := <-errCh:
		return fmt.Errorf("failed to shutdown: %w", err)
	default:
		return nil
	}
}

func (s *Server) ServeHTTPHandler(ctx context.Context, handler http.Handler) error {
	return s.ServeHTTP(ctx, &http.Server{
		Handler: handler,
	})
}
