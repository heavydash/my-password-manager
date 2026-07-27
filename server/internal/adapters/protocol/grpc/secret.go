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

type secretService struct {
	pb.UnimplementedSecretServiceServer
	secretUC *domain.SecretUseCase
	logger   logger.Logger
}

func NewSecretService(secretUC *domain.SecretUseCase, log logger.Logger) pb.SecretServiceServer {
	return &secretService{
		secretUC: secretUC,
		logger:   log,
	}
}

func (s *secretService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	s.logger.Info("gRPC CreateSecret", zap.String("user_id", userID), zap.String("title", req.GetTitle()))

	secret, err := s.secretUC.CreateSecret(
		ctx,
		userID,
		domain.SecretType(req.GetType()),
		req.GetTitle(),
		req.GetData(),
	)
	if err != nil {
		s.logger.Error("CreateSecret failed", zap.Error(err))
		return nil, err
	}

	resp := &pb.CreateSecretResponse{}
	resp.SetSecretId(secret.ID)
	resp.SetMessage("secret created successfully")

	return resp, nil
}
func (s *secretService) GetSecrets(ctx context.Context, req *pb.GetSecretsRequest) (*pb.GetSecretsResponse, error) {

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	secrets, err := s.secretUC.GetSecrets(ctx, userID)
	if err != nil {
		return nil, err
	}

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

	return resp, nil
}

func (s *secretService) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {

	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	secret, err := s.secretUC.GetSecret(ctx, req.GetSecretId(), userID)
	if err != nil {
		return nil, err
	}

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

	return resp, nil
}

func (s *secretService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in context")
	}

	err := s.secretUC.DeleteSecret(ctx, userID, req.GetSecretId())
	if err != nil {
		return nil, err
	}

	resp := &pb.DeleteSecretResponse{}
	resp.SetMessage("secret deleted successfully")

	return resp, nil
}
