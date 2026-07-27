// Package grpc реализует gRPC сервер для GophKeeper.
//
// Содержит интерсепторы:
//   - AuthInterceptor: проверяет JWT-токен для защищённых методов
//   - ValidateAndExtractUserID: извлекает и валидирует токен из metadata
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

// contextKey — тип для ключей контекста, чтобы избежать конфликтов.
type contextKey string

// userIDKey — ключ для хранения user_id в контексте запроса.
const userIDKey contextKey = "user_id"

// Константы для metadata
const (
	metadataAuthKey = "authorization"
	bearerPrefix    = "Bearer "
)

// authMethods — список методов, не требующих аутентификации.
var authMethods = map[string]bool{
	"/gophkeeper.AuthService/Register":      true,
	"/gophkeeper.AuthService/LoginPassword": true,
	"/gophkeeper.AuthService/LoginOAuth":    true,
	"/gophkeeper.AuthService/GetOAuthURL":   true,
}

// AuthInterceptor создаёт gRPC интерсептор для аутентификации.
//
// Алгоритм:
//  1. Пропускает методы из authMethods (регистрация, логин)
//  2. Отклоняет запросы к reflection (безопасность)
//  3. Для остальных методов извлекает и валидирует JWT
//  4. Добавляет user_id в контекст
//
// Параметры:
//   - authUC: бизнес-логика аутентификации (имеет метод ValidateJWT)
//   - log: логгер для записи событий
//
// Возвращает grpc.UnaryServerInterceptor.
func AuthInterceptor(authUC *domain.AuthUseCase, log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Пропускаем методы аутентификации (не требуют токена)
		if authMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Отключаем gRPC reflection для безопасности
		if strings.HasPrefix(info.FullMethod, "/grpc.reflection") {
			log.Warn("grpc reflection blocked", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.PermissionDenied, "reflection is disabled")
		}

		// Валидация токена
		userID, err := ValidateAndExtractUserID(ctx, authUC)
		if err != nil {
			log.Warn("gRPC auth failed", zap.String("method", info.FullMethod), zap.Error(err))
			return nil, err
		}

		// Добавляем user_id в контекст для использования в хендлерах
		newCtx := context.WithValue(ctx, userIDKey, userID)

		return handler(newCtx, req)
	}
}

// ValidateAndExtractUserID извлекает JWT-токен из metadata и проверяет его.
//
// Алгоритм:
//  1. Извлекает metadata из контекста
//  2. Читает заголовок "authorization"
//  3. Удаляет префикс "Bearer " если есть
//  4. Валидирует токен через authUC.ValidateJWT
//
// Параметры:
//   - ctx: контекст с gRPC metadata
//   - authUC: бизнес-логика аутентификации (имеет метод ValidateJWT)
//
// Возвращает:
//   - userID: идентификатор пользователя из токена
//   - error: gRPC статус-ошибка (Unauthenticated)
func ValidateAndExtractUserID(ctx context.Context, authUC *domain.AuthUseCase) (string, error) {
	// Извлечение metadata из контекста
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata provided")
	}

	// Поиск заголовка authorization
	values := md.Get(metadataAuthKey)
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization header is missing")
	}

	// Извлечение токена
	token := values[0]
	if strings.HasPrefix(token, bearerPrefix) {
		token = token[len(bearerPrefix):]
	}

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	// Валидация токена
	userID, err := authUC.ValidateJWT(token)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return userID, nil
}

// GetUserIDFromContext извлекает user_id из контекста (вспомогательная функция).
//
// Используется в хендлерах после прохождения AuthInterceptor.
//
// Параметры:
//   - ctx: контекст с user_id (добавлен интерсептором)
//
// Возвращает userID или пустую строку, если ключа нет.
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}
