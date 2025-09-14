package repository

import (
	"context"

	"github.com/JustScorpio/urlshortener/internal/models/entities"
)

// Интерфейс реализующий паттерн "репозиторий"
type IRepository[T entities.Entity] interface {
	GetAll(ctx context.Context) ([]T, error)
	Get(ctx context.Context, id string) (*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id []string, userID string) error

	CloseConnection()
	PingDB() bool
}
