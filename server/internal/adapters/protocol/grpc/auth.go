// Package grpc реализует gRPC сервер для GophKeeper.
//
// Содержит сервисы:
//   - AuthService: регистрация, логин, OAuth
//   - SecretService: управление секретами (в отдельном файле)
//
// Все методы логируют запросы и возвращают ошибки из domain слоя.
package grpc

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	pb "gophkeeper/server/proto"
)

// authService реализует gRPC сервис аутентификации.
//
// Реализует интерфейс pb.AuthServiceServer.
// Делегирует вызовы в AuthUseCase.
type authService struct {
	pb.UnimplementedAuthServiceServer
	authUC *domain.AuthUseCase
	logger logger.Logger
}

// NewAuthService создаёт новый gRPC сервис аутентификации.
//
// Параметры:
//   - authUC: бизнес-логика аутентификации
//   - log: логгер для записи событий
//
// Возвращает реализацию pb.AuthServiceServer.
func NewAuthService(authUC *domain.AuthUseCase, log logger.Logger) pb.AuthServiceServer {
	return &authService{
		authUC: authUC,
		logger: log,
	}
}

// Register обрабатывает gRPC запрос на регистрацию пользователя.
//
// Алгоритм:
//  1. Проверяет обязательность полей email и password
//  2. Вызывает AuthUseCase.Register
//  3. Возвращает user_id или ошибку
func (s *authService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Валидация входных данных
	if req.GetEmail() == "" {
		s.logger.Warn("grpc register: empty email")
		return nil, domain.ErrEmailRequired
	}
	if req.GetPassword() == "" {
		s.logger.Warn("grpc register: empty password")
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("grpc Register", zap.String("email", req.GetEmail()))

	userID, err := s.authUC.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		s.logger.Error("grpc register failed", zap.String("email", req.GetEmail()), zap.Error(err))
		return nil, err
	}

	resp := &pb.RegisterResponse{}
	resp.SetUserId(userID)
	resp.SetMessage("user registered successfully")

	s.logger.Info("grpc register success", zap.String("user_id", userID))
	return resp, nil
}

// LoginPassword обрабатывает gRPC запрос на вход по email и паролю.
//
// Алгоритм:
//  1. Проверяет обязательность полей email и password
//  2. Вызывает AuthUseCase.LoginPassword
//  3. Возвращает JWT-токен и user_id или ошибку
func (s *authService) LoginPassword(ctx context.Context, req *pb.LoginPasswordRequest) (*pb.LoginResponse, error) {
	// Валидация входных данных
	if req.GetEmail() == "" {
		s.logger.Warn("grpc login: empty email")
		return nil, domain.ErrEmailRequired
	}
	if req.GetPassword() == "" {
		s.logger.Warn("grpc login: empty password")
		return nil, domain.ErrInvalidInput
	}

	s.logger.Info("grpc LoginPassword", zap.String("email", req.GetEmail()))

	token, userID, err := s.authUC.LoginPassword(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		s.logger.Error("grpc login failed", zap.String("email", req.GetEmail()), zap.Error(err))
		return nil, err
	}

	resp := &pb.LoginResponse{}
	resp.SetToken(token)
	resp.SetUserId(userID)
	resp.SetMessage("login successfully")

	s.logger.Info("grpc login success", zap.String("user_id", userID))
	return resp, nil
}

// GetOAuthURL возвращает URL для редиректа на OAuth-провайдера.
//
// Алгоритм:
//  1. Проверяет, что провайдер указан
//  2. Вызывает AuthUseCase.GetOAuthURL
//  3. Возвращает URL (state не нужен для gRPC, т.к. редирект делает клиент)
func (s *authService) GetOAuthURL(ctx context.Context, req *pb.GetOAuthURLRequest) (*pb.GetOAuthURLResponse, error) {
	// Валидация входных данных
	if req.GetProvider() == "" {
		s.logger.Warn("grpc get oauth url: empty provider")
		return nil, domain.ErrUnsupportedOAuthProvider
	}

	s.logger.Info("grpc GetOAuthURL", zap.String("provider", req.GetProvider()))

	// state игнорируется, так как в gRPC нет HTTP-редиректа,
	// клиент сам сохранит state при необходимости
	url, _, err := s.authUC.GetOAuthURL(req.GetProvider())
	if err != nil {
		s.logger.Error("grpc get oauth url failed", zap.String("provider", req.GetProvider()), zap.Error(err))
		return nil, err
	}

	resp := &pb.GetOAuthURLResponse{}
	resp.SetAuthUrl(url)

	s.logger.Info("grpc get oauth url success", zap.String("provider", req.GetProvider()))
	return resp, nil
}

// LoginOAuth обрабатывает gRPC запрос на вход через OAuth.
//
// Алгоритм:
//  1. Проверяет обязательность provider и code (authorization code)
//  2. Вызывает AuthUseCase.LoginOAuth
//  3. Возвращает JWT-токен или ошибку
func (s *authService) LoginOAuth(ctx context.Context, req *pb.LoginOAuthRequest) (*pb.LoginResponse, error) {
	// Валидация входных данных
	if req.GetProvider() == "" {
		s.logger.Warn("grpc login oauth: empty provider")
		return nil, domain.ErrUnsupportedOAuthProvider
	}
	if req.GetCode() == "" {
		s.logger.Warn("grpc login oauth: empty code")
		return nil, domain.ErrOAuthCodeRequired
	}

	s.logger.Info("grpc LoginOAuth", zap.String("provider", req.GetProvider()))

	token, err := s.authUC.LoginOAuth(ctx, req.GetCode())
	if err != nil {
		s.logger.Error("grpc login oauth failed", zap.String("provider", req.GetProvider()), zap.Error(err))
		return nil, err
	}

	resp := &pb.LoginResponse{}
	resp.SetToken(token)
	resp.SetMessage("login successfully")

	s.logger.Info("grpc login oauth success", zap.String("provider", req.GetProvider()))
	return resp, nil
}
