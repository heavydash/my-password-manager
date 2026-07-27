// Package grpc реализует gRPC сервер для GophKeeper.
//
// Содержит сервисы:
//   - AuthService: регистрация, логин, OAuth
//   - SecretService: управление секретами (CRUD)
//
// Все методы логируют запросы и возвращают ошибки из domain слоя.
package grpc

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	pb "gophkeeper/server/proto"
	"time"
)

// secretService реализует gRPC сервис для управления секретами.
//
// Реализует интерфейс pb.SecretServiceServer.
// Делегирует вызовы в SecretUseCase.
type secretService struct {
	pb.UnimplementedSecretServiceServer
	secretUC *domain.SecretUseCase
	logger   logger.Logger
}

// NewSecretService создаёт новый gRPC сервис для управления секретами.
//
// Параметры:
//   - secretUC: бизнес-логика управления секретами
//   - log: логгер для записи событий
//
// Возвращает реализацию pb.SecretServiceServer.
func NewSecretService(secretUC *domain.SecretUseCase, log logger.Logger) pb.SecretServiceServer {
	return &secretService{
		secretUC: secretUC,
		logger:   log,
	}
}

// CreateSecret обрабатывает gRPC запрос на создание секрета.
//
// Алгоритм:
//  1. Извлекает user_id из контекста
//  2. Валидирует обязательные поля
//  3. Вызывает SecretUseCase.CreateSecret
//  4. Возвращает secret_id или ошибку
func (s *secretService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {

	// Извлечение user_id из контекста
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		s.logger.Warn("grpc create secret: user_id not found in context")
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	// Валидация входных данных
	if req.GetTitle() == "" {
		s.logger.Warn("grpc create secret: empty title", zap.String("user_id", userID))
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.GetData() == "" || len(req.GetData()) == 0 {
		s.logger.Warn("grpc create secret: empty data", zap.String("user_id", userID))
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	s.logger.Info("grpc create secret request",
		zap.String("user_id", userID),
		zap.String("title", req.GetTitle()),
	)

	secret, err := s.secretUC.CreateSecret(
		ctx,
		userID,
		domain.SecretType(req.GetType()),
		req.GetTitle(),
		req.GetData(),
	)
	if err != nil {
		s.logger.Error("grpc create secret failed",
			zap.String("user_id", userID),
			zap.String("title", req.GetTitle()),
			zap.Error(err),
		)
		return nil, err
	}

	resp := &pb.CreateSecretResponse{}
	resp.SetSecretId(secret.ID)
	resp.SetMessage("secret created successfully")

	s.logger.Info("grpc create secret success",
		zap.String("user_id", userID),
		zap.String("secret_id", secret.ID),
	)
	return resp, nil
}

// GetSecrets обрабатывает gRPC запрос на получение всех секретов пользователя.
//
// Алгоритм:
//  1. Извлекает user_id из контекста
//  2. Вызывает SecretUseCase.GetSecrets
//  3. Возвращает список секретов или ошибку
func (s *secretService) GetSecrets(ctx context.Context, req *pb.GetSecretsRequest) (*pb.GetSecretsResponse, error) {

	// Извлечение user_id из контекста
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		s.logger.Warn("grpc get secrets: user_id not found in context")
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	s.logger.Info("grpc get secrets request", zap.String("user_id", userID))

	secrets, err := s.secretUC.GetSecrets(ctx, userID)
	if err != nil {
		s.logger.Error("grpc get secrets failed", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}

	// Конвертация domain.Secret в pb.SecretData
	var pbSecrets []*pb.SecretData
	for _, sec := range secrets {
		data := &pb.SecretData{}
		data.SetId(sec.ID)
		data.SetUserId(sec.UserID)
		data.SetType(sec.Type)
		data.SetTitle(sec.Title)
		data.SetData(sec.Data)
		data.SetMetadata(sec.Metadata)
		data.SetCreatedAt(sec.CreatedAt.Format(time.RFC3339))
		data.SetUpdatedAt(sec.UpdatedAt.Format(time.RFC3339))

		pbSecrets = append(pbSecrets, data)
	}

	resp := &pb.GetSecretsResponse{}
	resp.SetSecrets(pbSecrets)

	s.logger.Info("grpc get secrets success",
		zap.String("user_id", userID),
		zap.Int("count", len(pbSecrets)),
	)
	return resp, nil
}

// GetSecret обрабатывает gRPC запрос на получение одного секрета по ID.
//
// Алгоритм:
//  1. Извлекает user_id из контекста
//  2. Валидирует secret_id
//  3. Вызывает SecretUseCase.GetSecret
//  4. Возвращает секрет или ошибку
func (s *secretService) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {

	// Извлечение user_id из контекста
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		s.logger.Warn("grpc get secret: user_id not found in context")
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	// Валидация входных данных
	if req.GetSecretId() == "" {
		s.logger.Warn("grpc get secret: empty secret_id", zap.String("user_id", userID))
		return nil, status.Error(codes.InvalidArgument, "secret_id is required")
	}

	s.logger.Info("grpc get secret request",
		zap.String("user_id", userID),
		zap.String("secret_id", req.GetSecretId()),
	)

	secret, err := s.secretUC.GetSecret(ctx, req.GetSecretId(), userID)
	if err != nil {
		s.logger.Error("grpc get secret failed",
			zap.String("user_id", userID),
			zap.String("secret_id", req.GetSecretId()),
			zap.Error(err),
		)
		return nil, err
	}

	// Конвертация domain.Secret в pb.SecretData
	data := &pb.SecretData{}
	data.SetId(secret.ID)
	data.SetUserId(secret.UserID)
	data.SetType(secret.Type)
	data.SetTitle(secret.Title)
	data.SetData(secret.Data)
	data.SetMetadata(secret.Metadata)
	data.SetCreatedAt(secret.CreatedAt.Format(time.RFC3339))
	data.SetUpdatedAt(secret.UpdatedAt.Format(time.RFC3339))

	resp := &pb.GetSecretResponse{}
	resp.SetSecret(data)

	s.logger.Info("grpc get secret success",
		zap.String("user_id", userID),
		zap.String("secret_id", secret.ID),
	)
	return resp, nil
}

// DeleteSecret обрабатывает gRPC запрос на удаление секрета.
//
// Алгоритм:
//  1. Извлекает user_id из контекста
//  2. Валидирует secret_id
//  3. Вызывает SecretUseCase.DeleteSecret
//  4. Возвращает подтверждение или ошибку
func (s *secretService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	// Извлечение user_id из контекста
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		s.logger.Warn("grpc delete secret: user_id not found in context")
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	// Валидация входных данных
	if req.GetSecretId() == "" {
		s.logger.Warn("grpc delete secret: empty secret_id", zap.String("user_id", userID))
		return nil, status.Error(codes.InvalidArgument, "secret_id is required")
	}

	s.logger.Info("grpc delete secret request",
		zap.String("user_id", userID),
		zap.String("secret_id", req.GetSecretId()),
	)

	err := s.secretUC.DeleteSecret(ctx, userID, req.GetSecretId())
	if err != nil {
		s.logger.Error("grpc delete secret failed",
			zap.String("user_id", userID),
			zap.String("secret_id", req.GetSecretId()),
			zap.Error(err),
		)
		return nil, err
	}

	resp := &pb.DeleteSecretResponse{}
	resp.SetMessage("secret deleted successfully")

	s.logger.Info("grpc delete secret success",
		zap.String("user_id", userID),
		zap.String("secret_id", req.GetSecretId()),
	)
	return resp, nil
}
