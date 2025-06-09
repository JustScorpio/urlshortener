package customcontext

import "context"

// Определяем собственный тип для ключа
type contextKey int

const (
	userIDKey contextKey = iota
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) string {
	userID := ctx.Value(userIDKey)
	if userID == nil {
		userID = ""
	}

	return userID.(string)
}
