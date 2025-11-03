// Пакет dtos содержит структуры используемые для переноса данных между разными частями приложения
package dtos

// Stats - dto для обмена статистикой сервиса
type Stats struct {
	URLsNum  int `json:"urls"`
	UsersNum int `json:"users"`
}
