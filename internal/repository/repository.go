package repository

import (
	"github.com/JustScorpio/urlshortener/internal/models"
)

// Интерфейс реализующий паттерн "репозиторий"
type IRepository[T models.Entity] interface {
	GetAll() ([]T, error)
	Get(id string) (*T, error)
	Create(entity *T) error
	Update(entity *T) error
	Delete(id string) error

	CloseConnection()
	PingDB() bool
}
