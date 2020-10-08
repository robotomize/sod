package server

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"rango/internal/logging"
	"rango/internal/srvenv"
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

func (s *Server) ServeGRPC(ctx context.Context, srv *grpc.Server) error {
	logger := logging.FromContext(ctx)
	errCh := make(chan error, 1)
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("server: создание GRPC завершилось с ошибкой на %s: %w", s.addr, err)
	}
	logger.Debugf("server: сервер GRPC запущен на порту %s", s.addr)
	go func() {
		<-ctx.Done()
		logger.Debugf("server: завершение по контексту GRPC")
		srv.GracefulStop()
	}()

	if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("server: ошибка запуска GRPC: %w", err)
	}

	logger.Debugf("server: сервис GRPC остановлен")

	select {
	case err := <-errCh:
		return fmt.Errorf("server: ошибка graceful shutdown: %w", err)
	}
}
