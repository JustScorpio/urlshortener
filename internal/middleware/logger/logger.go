// Пакет logger содержит middleware а также вспомогательные функции для логгирования
package logger

import (
	"net/http"
	"time"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"go.uber.org/zap"
)

// NewLogger - Инициализация синглтон логера с необходимым уровнем логирования
func NewLogger(level string, isProd bool) (*zap.Logger, error) {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	// создаём новую конфигурацию логера
	cfg := zap.NewDevelopmentConfig()
	if isProd {
		cfg = zap.NewProductionConfig()
	}

	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// LoggingMiddleware - middleware-логер для входящих HTTP-запросов
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем обертку для ResponseWriter, чтобы получить статус
			rw := &responseWriter{w, http.StatusOK, 0, ""}

			// Пропускаем запрос дальше
			next.ServeHTTP(rw, r)

			// Логируем после обработки
			duration := time.Since(start)

			userID := customcontext.GetUserID(r.Context())

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", duration),
				zap.String("ip", r.RemoteAddr),
				zap.String("user-agent", r.UserAgent()),
				zap.Int("status", rw.status),
				zap.Int("size", rw.size),
				zap.String("body", rw.body),
				zap.String("auth-token", userID),
			)
		})
	}
}

// responseWriter - обертка (встраивание) для ResponseWriter
type responseWriter struct {
	http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
	status              int
	size                int
	body                string
}

// Write - реализация интерфейса io.Writer. Осуществляет перенаправление данных записи через gzip-компрессор вместо непосредственной записи в HTTP-ответ
func (r *responseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.size += size // захватываем размер
	r.body = string(b)
	return size, err
}

// WriteHeader отправляет HTTP-заголовок с указанным статус-кодом и захватывает код для последующего логирования.
func (r *responseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.status = statusCode // захватываем код статуса
}
