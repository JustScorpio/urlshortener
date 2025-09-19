// Пакет customcontext содержит
package customcontext

import "context"

// contextKey - алиас вокруг int. Нужен для оопределение кастомных типов ключей
type contextKey int

// Кастомные типы ключей
const (
	userIDKey contextKey = iota
)

// WithUserID - добавить в контекст информацию о пользователе
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithUserID - извлечь из контекста информацию о пользователе
func GetUserID(ctx context.Context) string {
	userID := ctx.Value(userIDKey)
	if userID == nil {
		userID = ""
	}

	return userID.(string)
}
