// Пакет auth содержит middleware а также вспомогательные функции для аутентификации и авторизации пользователей
package auth

import (
	"context"
	"strings"
	"time"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// Имя заголовка для JWT-токена (вместо куки)
	jwtHeaderName = "authorization"
	// Префикс для Bearer токена
	bearerPrefix = "Bearer "
	// Время жизни токена
	tokenLifeTime = time.Hour * 3
	// Ключ для генерации и расшифровки токена
	secretKey = "supersecretkey"
)

// Claims — структура утверждений для JWT
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// newJWTString - создаёт токен и возвращает его в виде строки.
func newJWTString(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLifeTime)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GRPCAuthMiddleware - middleware для аутентификации в gRPC
func GRPCAuthMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		var userID string

		// Получаем метаданные из контекста
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// Если нет метаданных, создаем нового пользователя
			return createNewUserAndHandle(ctx, req, handler)
		}

		// Извлекаем токен из заголовка Authorization
		authHeaders := md.Get(jwtHeaderName)
		if len(authHeaders) == 0 {
			// Если нет заголовка Authorization, создаем нового пользователя
			return createNewUserAndHandle(ctx, req, handler)
		}

		tokenString := authHeaders[0]

		// Убираем префикс "Bearer " если есть
		if after, ok := strings.CutPrefix(tokenString, bearerPrefix); ok {
			tokenString = after
		}

		// Валидируем токен
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			// Если токен невалиден, создаем нового пользователя
			return createNewUserAndHandle(ctx, req, handler)
		}

		userID = claims.UserID

		// Добавляем userID в контекст
		ctx = customcontext.WithUserID(ctx, userID)

		return handler(ctx, req)
	}
}

// createNewUserAndHandle создает нового пользователя и обрабатывает запрос
func createNewUserAndHandle(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (interface{}, error) {
	userID := uuid.NewString()
	newToken, err := newJWTString(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create authentication token")
	}

	// Добавляем userID в контекст
	ctx = customcontext.WithUserID(ctx, userID)

	// Вызываем обработчик
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, err
	}

	grpc.SetHeader(ctx, metadata.Pairs(jwtHeaderName, bearerPrefix+newToken))

	return resp, nil
}
