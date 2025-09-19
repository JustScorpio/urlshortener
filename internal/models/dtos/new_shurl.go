// Пакет dtos содержит структуры используемые для переноса данных между разными частями приложения
package dtos

// NewShURL - dto для новых создаваемых shURL
type NewShURL struct {
	LongURL   string
	CreatedBy string
}
