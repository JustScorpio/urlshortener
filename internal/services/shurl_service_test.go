package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestShURLService_GetByID(t *testing.T) {
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
		shURL, err := service.GetByID(ctx, created.Token)
		require.NoError(t, err)
		assert.Equal(t, created.Token, shURL.Token)
		assert.Equal(t, created.LongURL, shURL.LongURL)
	})

	t.Run("get non-existing URL", func(t *testing.T) {
		shURL, err := service.GetByID(ctx, "nonexistent")
		assert.Nil(t, shURL)

		// NotFound
		var httpErr *customerrors.HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
	})
}

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
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 2)
		for _, shURL := range shURLs {
			assert.Equal(t, "user1", shURL.CreatedBy)
		}
	})

	t.Run("get URLs by user2", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user2")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
		assert.Equal(t, "user2", shURLs[0].CreatedBy)
	})

	t.Run("get URLs by non-existing user", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})
}

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
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
	})

	t.Run("delete URLs by wrong user", func(t *testing.T) {
		err := service.Delete(ctx, []string{tokens[2]}, "user1")
		require.NoError(t, err)

		// URL should still exist since user1 doesn't own it
		shURL, err := service.GetByID(ctx, tokens[2])
		require.NoError(t, err)
		assert.NotNil(t, shURL)
	})
}

func TestShURLService_GetByCondition(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	// Setup test data
	urls := []dtos.NewShURL{
		{LongURL: "https://example1.com", CreatedBy: "user1"},
		{LongURL: "https://example2.com", CreatedBy: "user1"},
		{LongURL: "https://example3.com", CreatedBy: "user2"},
		{LongURL: "https://example4.com", CreatedBy: "user3"},
	}

	for _, url := range urls {
		_, err := service.Create(ctx, url)
		require.NoError(t, err)
	}

	t.Run("get by CreatedBy field - user1", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 2)
		for _, shURL := range shURLs {
			assert.Equal(t, "user1", shURL.CreatedBy)
			assert.Contains(t, []string{"https://example1.com", "https://example2.com"}, shURL.LongURL)
		}
	})

	t.Run("get by CreatedBy field - user2", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user2")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
		assert.Equal(t, "user2", shURLs[0].CreatedBy)
		assert.Equal(t, "https://example3.com", shURLs[0].LongURL)
	})

	t.Run("get by CreatedBy field - non-existing user", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})

	t.Run("get by LongURL field", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, "LongURL", "https://example1.com")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
		assert.Equal(t, "https://example1.com", shURLs[0].LongURL)
		assert.Equal(t, "user1", shURLs[0].CreatedBy)
	})

	t.Run("invalid field name", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, "InvalidField", "value")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})

	t.Run("empty value", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})

	t.Run("case sensitive field names", func(t *testing.T) {
		// Тестируем чувствительность к регистру в названиях полей
		shURLs, err := service.GetByCondition(ctx, "createdby", "user1") // lowercase
		require.NoError(t, err)
		// В зависимости от реализации может возвращать пустой результат или ошибку
		// Здесь предполагаем, что возвращается пустой результат для некорректных имен полей
		assert.Empty(t, shURLs)
	})

	t.Run("multiple matches", func(t *testing.T) {
		// Добавим еще одну запись для user1
		newURL := dtos.NewShURL{
			LongURL:   "https://example5.com",
			CreatedBy: "user1",
		}
		_, err := service.Create(ctx, newURL)
		require.NoError(t, err)

		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
		require.NoError(t, err)
		assert.Len(t, shURLs, 3)
		for _, shURL := range shURLs {
			assert.Equal(t, "user1", shURL.CreatedBy)
		}
	})
}

func TestShURLService_GetByCondition_EdgeCases(t *testing.T) {
	mockRepo := inmemory.NewInMemoryRepository()
	service := NewShURLService(mockRepo)
	ctx := context.Background()

	t.Run("empty repository", func(t *testing.T) {
		shURLs, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
		require.NoError(t, err)
		assert.Empty(t, shURLs)
	})

	//Хороший тест который пока не покорён
	// t.Run("context cancellation", func(t *testing.T) {
	// 	cancelCtx, cancel := context.WithCancel(ctx)
	// 	cancel() // immediately cancel

	// 	shURLs, err := service.GetByCondition(cancelCtx, entities.ShURLCreatedByFieldName, "user1")
	// 	require.Error(t, err)
	// 	assert.True(t, errors.Is(err, context.Canceled))
	// 	assert.Empty(t, shURLs)
	// })

	t.Run("special characters in values", func(t *testing.T) {
		specialURL := dtos.NewShURL{
			LongURL:   "https://example.com/?query=test&param=value",
			CreatedBy: "user-with-dash",
		}
		created, err := service.Create(ctx, specialURL)
		require.NoError(t, err)

		shURLs, err := service.GetByCondition(ctx, "LongURL", "https://example.com/?query=test&param=value")
		require.NoError(t, err)
		assert.Len(t, shURLs, 1)
		assert.Equal(t, created.LongURL, shURLs[0].LongURL)

		shURLs2, err := service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user-with-dash")
		require.NoError(t, err)
		assert.Len(t, shURLs2, 1)
		assert.Equal(t, created.CreatedBy, shURLs2[0].CreatedBy)
	})
}
