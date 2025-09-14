package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/repository/inmemory"
	"github.com/JustScorpio/urlshortener/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShURLHandler_GetFullURL(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080")

	// Setup test data
	ctx := context.Background()
	newURL := dtos.NewShURL{
		LongURL:   "https://example.com",
		CreatedBy: "user1",
	}
	shURL, err := service.Create(ctx, newURL)
	require.NoError(t, err)

	t.Run("successful redirect", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+shURL.Token, nil)
		w := httptest.NewRecorder()

		handler.GetFullURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, shURL.LongURL, resp.Header.Get("Location"))
	})

	t.Run("non-existing token returns error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()

		handler.GetFullURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("empty token returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.GetFullURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("wrong method returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/"+shURL.Token, nil)
		w := httptest.NewRecorder()

		handler.GetFullURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestShURLHandler_ShortenURL(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080")

	t.Run("successful creation with text content type", func(t *testing.T) {
		body := strings.NewReader("https://example_text.com")
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "text/plain")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ShortenURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		bodyBytes, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(bodyBytes), "http://localhost:8080/")
	})

	t.Run("successful creation with JSON content type", func(t *testing.T) {
		jsonBody := `{"url": "https://example_json.com"}`
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

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var response struct {
			Result string `json:"result"`
		}
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response.Result, "http://localhost:8080/")
	})

	t.Run("duplicate URL returns conflict status", func(t *testing.T) {
		// Пытаемся создать дубликат
		body := strings.NewReader("https://example_text.com")
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "text/plain")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ShortenURL(w, req)
		resp2 := w.Result()
		defer resp2.Body.Close()

		// Должен вернуть статус Conflict
		assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	})

	t.Run("empty body returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ShortenURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("wrong method returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.ShortenURL(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestShURLHandler_ShortenURLsBatch(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080")

	t.Run("successful batch creation", func(t *testing.T) {
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

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var response []map[string]string
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "1", response[0]["correlation_id"])
		assert.Contains(t, response[0]["short_url"], "http://localhost:8080/")
	})

	t.Run("wrong content type returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/shorten/batch", strings.NewReader("test"))
		req.Header.Set("Content-Type", "text/plain")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ShortenURLsBatch(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty body returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/shorten/batch", nil)
		req.Header.Set("Content-Type", "application/json")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ShortenURLsBatch(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestShURLHandler_GetShURLsByUserID(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080")

	// Setup test data
	ctx := context.Background()
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
	}
	for _, url := range urls {
		_, err := service.Create(ctx, url)
		require.NoError(t, err)
	}

	t.Run("successful get user URLs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetShURLsByUserID(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var response []map[string]string
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("no user ID returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)
		w := httptest.NewRecorder()

		handler.GetShURLsByUserID(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("no URLs for user returns no content", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user/urls", nil)
		ctx := customcontext.WithUserID(req.Context(), "user2")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetShURLsByUserID(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestShURLHandler_DeleteMany(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	handler := handlers.NewShURLHandler(service, "localhost:8080")

	// Setup test data
	ctx := context.Background()
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
	}

	var tokens []string
	for _, url := range urls {
		shURL, err := service.Create(ctx, url)
		require.NoError(t, err)
		tokens = append(tokens, shURL.Token)
	}

	t.Run("successful delete URLs", func(t *testing.T) {
		jsonBody, _ := json.Marshal([]string{tokens[0]})
		req := httptest.NewRequest("DELETE", "/api/user/urls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteMany(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("no user ID returns unauthorized", func(t *testing.T) {
		jsonBody, _ := json.Marshal([]string{tokens[0]})
		req := httptest.NewRequest("DELETE", "/api/user/urls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.DeleteMany(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("empty body returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/user/urls", nil)
		req.Header.Set("Content-Type", "application/json")
		ctx := customcontext.WithUserID(req.Context(), "user1")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteMany(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("wrong method returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/user/urls", nil)
		w := httptest.NewRecorder()

		handler.DeleteMany(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}
