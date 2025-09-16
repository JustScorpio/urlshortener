// Пакет services содержит структуры и методы, реализующие бизнес-логику приложения
package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/repository/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShURLService_Create - проверка создания ShURL
func TestShURLService_Create(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		newURL := dtos.NewShURL{
			LongURL:   "https://example.com",
			CreatedBy: "user1",
		}

		shURL, err := service.Create(ctx, newURL)
		require.NoError(t, err)
		assert.NotEmpty(t, shURL.Token)
		assert.Equal(t, newURL.LongURL, shURL.LongURL)
		assert.Equal(t, newURL.CreatedBy, shURL.CreatedBy)
	})

	t.Run("duplicate URL", func(t *testing.T) {
		newURL := dtos.NewShURL{
			LongURL:   "https://example.com",
			CreatedBy: "user1",
		}

		shURL, err := service.Create(ctx, newURL)
		assert.Error(t, err)
		assert.Equal(t, newURL.LongURL, shURL.LongURL)
		assert.Equal(t, newURL.CreatedBy, shURL.CreatedBy)

		// AlreadyExistsError
		var httpErr *customerrors.HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusConflict, httpErr.Code)
	})
}

// TestShURLService_Get - проверка получения ShURL
func TestShURLService_Get(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	// Setup test data
	newURL := dtos.NewShURL{
		LongURL:   "https://example.com",
		CreatedBy: "user1",
	}
	created, err := service.Create(ctx, newURL)
	require.NoError(t, err)

	t.Run("get existing URL", func(t *testing.T) {
		shURL, err := service.Get(ctx, created.Token)
		require.NoError(t, err)
		assert.Equal(t, created.Token, shURL.Token)
		assert.Equal(t, created.LongURL, shURL.LongURL)
	})

	t.Run("get non-existing URL", func(t *testing.T) {
		shURL, err := service.Get(ctx, "nonexistent")
		assert.Nil(t, shURL)

		// NotFound
		var httpErr *customerrors.HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
	})
}

// TestShURLService_GetAllShURLsByUserID - проверка получения ShURL конкретного пользователя
func TestShURLService_GetAllShURLsByUserID(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	// Setup test data
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
		{LongURL: "https://example3.com", CreatedBy: "user2"},
	}

	for _, url := range urls {
		_, err := service.Create(ctx, url)
		require.NoError(t, err)
	}

	t.Run("get URLs by user1", func(t *testing.T) {
		shURLs, err := service.GetAllShURLsByUserID(ctx, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 2)
		for _, shURL := range shURLs {
			assert.Equal(t, "user1", shURL.CreatedBy)
		}
	})

	t.Run("get URLs by user2", func(t *testing.T) {
		shURLs, err := service.GetAllShURLsByUserID(ctx, "user2")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
		assert.Equal(t, "user2", shURLs[0].CreatedBy)
	})

	t.Run("get URLs by non-existing user", func(t *testing.T) {
		shURLs, err := service.GetAllShURLsByUserID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})
}

// TestShURLService_Delete - проверка удаления ShURL
func TestShURLService_Delete(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	// Setup test data
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
		{LongURL: "https://example3.com", CreatedBy: "user2"},
	}

	var tokens []string
	for _, url := range urls {
		shURL, err := service.Create(ctx, url)
		require.NoError(t, err)
		tokens = append(tokens, shURL.Token)
	}

	t.Run("delete URLs by owner", func(t *testing.T) {
		err := service.Delete(ctx, []string{tokens[0]}, "user1")
		require.NoError(t, err)

		// Verify deletion
		shURLs, err := service.GetAllShURLsByUserID(ctx, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
	})

	t.Run("delete URLs by wrong user", func(t *testing.T) {
		err := service.Delete(ctx, []string{tokens[2]}, "user1")
		require.NoError(t, err)

		// URL should still exist since user1 doesn't own it
		shURL, err := service.Get(ctx, tokens[2])
		require.NoError(t, err)
		assert.NotNil(t, shURL)
	})
}
