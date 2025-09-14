package inmemory

import (
	"context"
	"errors"
	"sync"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
)

var errNotFound = customerrors.NewNotFoundError(errors.New("not found"))
var errAlreadyExists = errors.New("already exists")
var errGone = customerrors.NewGoneError(errors.New("shurl has been deleted"))

type InMemoryRepository struct {
	mu            sync.RWMutex
	shURLs        map[string]entities.ShURL
	deletedShURLs map[string]entities.ShURL
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		shURLs:        make(map[string]entities.ShURL),
		deletedShURLs: make(map[string]entities.ShURL),
	}
}

func (m *InMemoryRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]entities.ShURL, 0, len(m.shURLs))
	for _, shURL := range m.shURLs {
		result = append(result, shURL)
	}

	return result, nil
}

func (m *InMemoryRepository) Get(ctx context.Context, token string) (*entities.ShURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if shURL, exists := m.shURLs[token]; exists {
		return &shURL, nil
	}

	if _, exists := m.deletedShURLs[token]; exists {
		return nil, errGone
	}

	return nil, errNotFound
}

func (m *InMemoryRepository) Create(ctx context.Context, shURL *entities.ShURL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.shURLs[shURL.Token]; exists {
		return errAlreadyExists
	}

	m.shURLs[shURL.Token] = *shURL
	return nil
}

func (m *InMemoryRepository) Update(ctx context.Context, shURL *entities.ShURL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.shURLs[shURL.Token]; exists {
		m.shURLs[shURL.Token] = *shURL
		return nil
	} else {
		return errNotFound
	}
}

func (m *InMemoryRepository) Delete(ctx context.Context, tokens []string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, token := range tokens {
		if shURL, exists := m.shURLs[token]; exists && shURL.CreatedBy == userID {
			m.deletedShURLs[token] = shURL
			delete(m.shURLs, token)
		}
	}
	return nil
}

func (m *InMemoryRepository) PingDB() bool {
	return m.shURLs != nil
}

func (m *InMemoryRepository) CloseConnection() {
}
