package whitelist

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

// GRPCCIDRWhitelistMiddleware - middleware с выборочной проверкой методов
func (m *CIDRWhitelistMiddleware) CIDRWhitelistMiddleware(protectedMethods ...string) grpc.UnaryServerInterceptor {
	protectedSet := make(map[string]bool)
	for _, method := range protectedMethods {
		protectedSet[method] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Проверяем только защищенные методы
		if protectedSet[info.FullMethod] {
			realIP := m.getRealIPFromContext(ctx)
			if !m.isIPAllowed(realIP) {
				return nil, status.Error(codes.PermissionDenied, "access denied: IP not in trusted subnet")
			}
		}

		return handler(ctx, req)
	}
}

func (m *CIDRWhitelistMiddleware) getRealIPFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	if values := md.Get("X-Real-IP"); len(values) > 0 {
		return values[0]
	}

	return ""
}
