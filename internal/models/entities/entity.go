// Пакет entities содержит структуры реализующие сущности доменной модели приложения
package entities

// IEntity - интерфейс сущностей доменной модели приложения
type IEntity interface {
	GetID() string
}
