package grpc

import (
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	pb "gophkeeper/server/proto"
	"net"
)

type Server struct {
	authService   *domain.AuthUseCase
	secretService *domain.SecretUseCase
	logger        logger.Logger
	grpcServer    *grpc.Server
}

func NewServer(authUC *domain.AuthUseCase, secretUC *domain.SecretUseCase, log logger.Logger) *Server {

	if authUC == nil {
		panic("authUC is nil")
	}
	if secretUC == nil {
		panic("secretUC is nil")
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(AuthInterceptor(secretUC.GetTokenValidator(), log)),
	)

	pb.RegisterAuthServiceServer(srv, NewAuthService(authUC, log))
	pb.RegisterSecretServiceServer(srv, NewSecretService(secretUC, log))

	reflection.Register(srv)

	return &Server{grpcServer: srv,
		logger: log}
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.logger.Info("gRPC server started", zap.String("address", addr))
	return s.grpcServer.Serve(lis)
}

func (s *Server) GracefulStop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC server stopped gracefully")
	}
}
