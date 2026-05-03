package grpc

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"strings"
)

func AuthInterceptor(validator domain.TokenValidator, log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/gophkeeper.AuthService/Register" ||
			info.FullMethod == "/gophkeeper.AuthService/LoginPassword" ||
			info.FullMethod == "/gophkeeper.AuthService/LoginOAuth" ||
			info.FullMethod == "/gophkeeper.AuthService/GetOAuthURL" {
			return handler(ctx, req)
		}

		if strings.HasPrefix(info.FullMethod, "/grpc.reflection") {
			return nil, status.Error(codes.Unimplemented, "reflection is disabled")
		}

		userID, err := ValidateAndExtractUserID(ctx, validator)
		if err != nil {
			log.Warn("gRPC auth failed", zap.String("method", info.FullMethod), zap.Error(err))
			return nil, err
		}

		// Кладём userID в контекст
		newCtx := context.WithValue(ctx, "user_id", userID)

		return handler(newCtx, req)
	}
}

func ValidateAndExtractUserID(ctx context.Context, validator domain.TokenValidator) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization header is missing")
	}

	token := values[0]
	if strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	userID, err := validator.ValidateToken(token)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return userID, nil
}
