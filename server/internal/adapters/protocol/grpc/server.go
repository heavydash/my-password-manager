// Package grpc реализует gRPC сервер для GophKeeper.
//
// Содержит:
//   - Server: основная структура сервера
//   - Запуск и graceful остановка gRPC сервера
//   - Регистрация сервисов (AuthService, SecretService)
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

// Server представляет gRPC сервер GophKeeper.
//
// Объединяет:
//   - AuthService: аутентификация и OAuth
//   - SecretService: управление секретами
//   - AuthInterceptor: JWT-валидация
type Server struct {
	authService   *domain.AuthUseCase
	secretService *domain.SecretUseCase
	logger        logger.Logger
	grpcServer    *grpc.Server
}

// NewServer создаёт новый gRPC сервер с зарегистрированными сервисами.
//
// Алгоритм:
//  1. Проверяет обязательные зависимости (authUC, secretUC, log)
//  2. Создаёт gRPC сервер с AuthInterceptor
//  3. Регистрирует AuthService и SecretService
//  4. Регистрирует reflection (для отладки)
//
// Параметры:
//   - authUC: бизнес-логика аутентификации
//   - secretUC: бизнес-логика управления секретами
//   - log: логгер для записи событий
//
// Возвращает готовый Server.
func NewServer(authUC *domain.AuthUseCase, secretUC *domain.SecretUseCase, log logger.Logger) *Server {
	// Проверка обязательных зависимостей
	if authUC == nil {
		panic("authUC is nil")
	}
	if secretUC == nil {
		panic("secretUC is nil")
	}
	if log == nil {
		panic("log is nil")
	}

	// Создание gRPC сервера с интерсептором аутентификации
	// AuthInterceptor использует authUC для валидации JWT (authUC имеет доступ к ValidateJWT)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(AuthInterceptor(authUC, log)),
	)

	// Регистрация сервисов
	pb.RegisterAuthServiceServer(srv, NewAuthService(authUC, log))
	pb.RegisterSecretServiceServer(srv, NewSecretService(secretUC, log))

	// Регистрация reflection
	reflection.Register(srv)

	log.Info("gRPC server initialized",
		zap.Bool("reflection_enabled", true),
	)

	return &Server{grpcServer: srv,
		logger: log}
}

// Start запускает gRPC сервер на указанном адресе.
//
// Алгоритм:
//  1. Создаёт TCP listener на addr
//  2. Запускает gRPC сервер (блокирующий вызов)
//
// Параметры:
//   - addr: адрес для прослушивания (например, ":9090")
//
// Возвращает ошибку, если не удалось запустить сервер.
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Error("gRPC server failed to listen", zap.String("address", addr), zap.Error(err))
		return err
	}

	s.logger.Info("gRPC server started", zap.String("address", addr))
	return s.grpcServer.Serve(lis)
}

// GracefulStop останавливает gRPC сервер с ожиданием завершения активных запросов.
//
// Алгоритм:
//  1. Проверяет, что сервер существует
//  2. Вызывает GracefulStop (ждёт до завершения всех RPC)
//  3. Логирует остановку
//
// Используется при graceful shutdown всего приложения.
func (s *Server) GracefulStop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC server stopped gracefully")
	}
}
