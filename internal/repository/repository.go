// Пакет repository содержит интерфейс для реализации паттерна "Репозиторий"
package repository

import (
	"context"

	"github.com/JustScorpio/urlshortener/internal/models/entities"
)

// Интерфейс реализующий паттерн "репозиторий"
type IRepository[T entities.IEntity] interface {
	// GetAll - получить все сущности
	GetAll(ctx context.Context) ([]T, error)
	// Get - получить сущность по ID
	Get(ctx context.Context, id string) (*T, error)
	// Create - создать сущность
	Create(ctx context.Context, IEntity *T) error
	// Update - обновить сущность
	Update(ctx context.Context, IEntity *T) error
	// Delete - удалить сущность
	Delete(ctx context.Context, id []string, userID string) error

	// CloseConnection - закрыть соединение с базой данных
	CloseConnection()
	// PingDB - проверить подключение к базе данных
	PingDB() bool
}
