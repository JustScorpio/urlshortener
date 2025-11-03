// Пакет whitelist содержит middleware и функции для проверки что отправитель запроса входит в доверенную подсеть
package whitelist

import (
	"net"
	"net/http"
)

// Middleware проверки что отправитель запроса входит в доверенную подсеть
type CIDRWhitelistMiddleware struct {
	allowedNetwork *net.IPNet
}

// NewCIDRWhitelistMiddleware - создать middleware для проверки что отправитель запроса входит в доверенную подсеть
func NewCIDRWhitelistMiddleware(allowedCIDR string) (*CIDRWhitelistMiddleware, error) {
	m := &CIDRWhitelistMiddleware{}

	// Парсим allowedCIDR
	_, network, _ := net.ParseCIDR(allowedCIDR)
	m.allowedNetwork = network

	return m, nil
}

// isIPAllowed - проверить что IP-адрес входит в доверенную подсеть
func (m *CIDRWhitelistMiddleware) isIPAllowed(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Проверяем вхождение IP в подсеть
	if m.allowedNetwork.Contains(ip) {
		return true
	}

	return false
}

// CIDRWhitelistMiddleware - middleware для проверки что отправитель запроса входит в доверенную подсеть
func (m *CIDRWhitelistMiddleware) CIDRWhitelistMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := r.Header.Get("X-Real-IP")

			if realIP == "" || !m.isIPAllowed(realIP) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Передаем запрос дальше
			next.ServeHTTP(w, r)
		})
	}
}
