// Пакет entities содержит структуры реализующие сущности доменной модели приложения
package entities

// ShURL - укороченная ссылка
type ShURL struct {
	Token     string
	LongURL   string
	CreatedBy string
}

// GetID - реализация интерфейса IEntity
func (su ShURL) GetID() string {
	return su.Token
}
