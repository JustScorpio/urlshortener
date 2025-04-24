package repository

import (
	"github.com/JustScorpio/urlshortener/internal/models"
)

// Не видел нигде рекомендаций по неймингу интерфейсов, но считаю уместным отличать их от структур
type IRepository[T models.Entity] interface {
	GetAll() ([]T, error)
	Get(id string) (*T, error)
	Create(entity *T) error
	Update(entity *T) error
	Delete(id string) error
}
