// Пакет logger содержит middleware а также вспомогательные функции для логгирования
package logger

import (
	"context"
	"time"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCLoggingMiddleware - middleware-логер для входящих gRPC-запросов
func GRPCLoggingMiddleware(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Получаем метаданные для логирования
		var userAgent, clientIP string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ua := md.Get("user-agent"); len(ua) > 0 {
				userAgent = ua[0]
			}
			if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
				clientIP = xff[0]
			} else if peer := md.Get("x-real-ip"); len(peer) > 0 {
				clientIP = peer[0]
			}
		}

		userID := customcontext.GetUserID(ctx)

		// Логируем начало запроса
		logger.Debug("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("user-agent", userAgent),
			zap.String("ip", clientIP),
			zap.String("auth-token", userID),
		)

		// Пропускаем запрос дальше
		resp, err := handler(ctx, req)

		// Логируем после обработки
		duration := time.Since(start)

		// Получаем статус код
		grpcStatus := status.Convert(err)
		statusCode := grpcStatus.Code()

		// Логируем в зависимости от успешности
		if err != nil {
			logger.Warn("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("ip", clientIP),
				zap.String("user-agent", userAgent),
				zap.String("status", statusCode.String()),
				zap.Int32("status_code", int32(statusCode)),
				zap.String("error", grpcStatus.Message()),
				zap.String("auth-token", userID),
				zap.Error(err),
			)
		} else {
			logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("ip", clientIP),
				zap.String("user-agent", userAgent),
				zap.String("status", statusCode.String()),
				zap.Int32("status_code", int32(statusCode)),
				zap.String("auth-token", userID),
			)
		}

		return resp, err
	}
}
