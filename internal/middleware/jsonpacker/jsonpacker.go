package jsonpacker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// middleware для преобразования тела запроса из- и в- формат json.
func JSONPackingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var buf bytes.Buffer
			// читаем тело запроса
			_, err := buf.ReadFrom(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var reqData struct {
				URL string `json:"url"`
			}

			if err = json.Unmarshal(buf.Bytes(), &reqData); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Конвертируем в строку
			url := reqData.URL

			// Подменяем тело запроса
			r.Body = io.NopCloser(strings.NewReader(url))
			r.ContentLength = int64(len(url))
			r.Header.Set("Content-Type", "text/plain")

			// Создаем обертку для ответа
			wrappedWriter := &responseWriter{
				ResponseWriter: w,
			}

			// Пропускаем запрос дальше
			next.ServeHTTP(wrappedWriter, r)
		})
	}
}

// Обертка для ResponseWriter
type responseWriter struct {
	http.ResponseWriter
}

func (w *responseWriter) WriteHeader(code int) {
	// Устанавливаем Content-Type перед фиксацией заголовков
	w.Header().Set("Content-Type", "application/json")

	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(data []byte) (int, error) {

	// Конвертируем plain text в JSON
	var respData struct {
		Result string `json:"result"`
	}

	respData.Result = string(data)
	jsonData, err := json.Marshal(respData)
	if err != nil {
		return 0, err
	}
	return w.ResponseWriter.Write(jsonData)
}
