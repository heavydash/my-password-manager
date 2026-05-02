package grpc

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	pb "gophkeeper/server/proto"
)

type authService struct {
	pb.UnimplementedAuthServiceServer
	authUC *domain.AuthUseCase
	logger logger.Logger
}

func NewAuthService(authUC *domain.AuthUseCase, log logger.Logger) pb.AuthServiceServer {
	return &authService{
		authUC: authUC,
		logger: log,
	}
}

func (s *authService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("gRPC Register", zap.String("email", req.GetEmail()))

	userID, err := s.authUC.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	resp := &pb.RegisterResponse{}
	resp.SetUserId(userID)
	resp.SetMessage("user registered successfully")

	return resp, nil
}

func (s *authService) LoginPassword(ctx context.Context, req *pb.LoginPasswordRequest) (*pb.LoginResponse, error) {
	s.logger.Info("gRPC LoginPassword", zap.String("email", req.GetEmail()))

	token, userID, err := s.authUC.LoginPassword(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	resp := &pb.LoginResponse{}
	resp.SetToken(token)
	resp.SetUserId(userID)
	resp.SetMessage("login successfully")

	return resp, nil
}

func (s *authService) LoginOAuth(ctx context.Context, req *pb.LoginOAuthRequest) (*pb.LoginResponse, error) {
	s.logger.Info("gRPC LoginOAuth", zap.String("provider", req.GetProvider()))

	resp := &pb.LoginResponse{}
	resp.SetMessage("oauth login not implemented yet")

	return resp, nil
}
