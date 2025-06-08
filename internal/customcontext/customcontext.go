package customcontext

import "context"

const (
	userIdKey = "user_id"
)

func WithUserId(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIdKey, userID)
}

func GetUserId(ctx context.Context) string {
	userID := ctx.Value(userIdKey)
	if userID == nil {
		userID = ""
	}

	return userID.(string)
}
