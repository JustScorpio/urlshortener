// Пакет handlers_test содержит тесты обработчиков входящих запросов и вспомогательные функции
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/JustScorpio/urlshortener/internal/http/handlers"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/repository/inmemory"
	"github.com/JustScorpio/urlshortener/internal/services"
)

// ExampleShURLHandler_GetFullURL - демонстрирует использование обработчика GetFullURL.
func ExampleShURLHandler_GetFullURL() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	// Создаем тестовые данные
	ctx := context.Background()
	newURL := dtos.NewShURL{
		LongURL:   "https://example.com",
		CreatedBy: "user1",
	}
	shURL, _ := service.Create(ctx, newURL)

	// Создаем HTTP запрос
	req := httptest.NewRequest("GET", "/"+shURL.Token, nil)
	w := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetFullURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}

// ExampleShURLHandler_ShortenURL - демонстрирует создание ShURL через текстовый запрос.
func ExampleShURLHandler_ShortenURL() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	// Текстовый запрос
	body := strings.NewReader("https://example.com")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "text/plain")
	ctx := customcontext.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ShortenURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}

// ExampleShURLHandler_ShortenURL_json - демонстрирует создание ShURL через JSON запрос.
func ExampleShURLHandler_ShortenURL_json() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	// JSON запрос
	jsonBody := `{"url": "https://example.com"}`
	body := strings.NewReader(jsonBody)
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	ctx := customcontext.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ShortenURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}

// ExampleShURLHandler_ShortenURLsBatch - демонстрирует пакетное создание ShURL.
func ExampleShURLHandler_ShortenURLsBatch() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	batch := []map[string]string{
		{"correlation_id": "1", "original_url": "https://example1.com"},
		{"correlation_id": "2", "original_url": "https://example2.com"},
	}
	jsonBody, _ := json.Marshal(batch)

	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := customcontext.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ShortenURLsBatch(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}

// ExampleShURLHandler_GetShURLsByUserID - демонстрирует получение всех ShURL пользователя.
func ExampleShURLHandler_GetShURLsByUserID() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	// Создаем тестовые данные
	ctx := context.Background()
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
	}
	for _, url := range urls {
		service.Create(ctx, url)
	}

	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	ctxReq := customcontext.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctxReq)
	w := httptest.NewRecorder()

	handler.GetShURLsByUserID(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}

// ExampleShURLHandler_DeleteMany демонстрирует удаление ShURL пользователя.
func ExampleShURLHandler_DeleteMany() {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080", false)

	// Создаем тестовые данные
	ctx := context.Background()
	url := dtos.NewShURL{
		LongURL:   "https://example.com",
		CreatedBy: "user1",
	}
	shURL, _ := service.Create(ctx, url)

	jsonBody, _ := json.Marshal([]string{shURL.Token})
	req := httptest.NewRequest("DELETE", "/api/user/urls", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctxReq := customcontext.WithUserID(req.Context(), "user1")
	req = req.WithContext(ctxReq)
	w := httptest.NewRecorder()

	handler.DeleteMany(w, req)

	resp := w.Result()
	defer resp.Body.Close()
}
