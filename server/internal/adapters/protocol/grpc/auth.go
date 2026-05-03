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

func (s *authService) GetOAuthURL(ctx context.Context, req *pb.GetOAuthURLRequest) (*pb.GetOAuthURLResponse, error) {
	s.logger.Info("gRPC GetOAuthURL", zap.String("provider", req.GetProvider()))

	url, _, err := s.authUC.GetOAuthURL(req.GetProvider())
	if err != nil {
		return nil, err
	}

	resp := &pb.GetOAuthURLResponse{}
	resp.SetAuthUrl(url)
	return resp, nil
}

func (s *authService) LoginOAuth(ctx context.Context, req *pb.LoginOAuthRequest) (*pb.LoginResponse, error) {
	s.logger.Info("gRPC LoginOAuth", zap.String("provider", req.GetProvider()))

	token, err := s.authUC.LoginOAuth(ctx, req.GetCode()) // or oneTimeCode
	if err != nil {
		return nil, err
	}

	resp := &pb.LoginResponse{}
	resp.SetToken(token)
	resp.SetMessage("login successfully")
	return resp, nil
}
