package inmemory

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
)

var errNotFound = customerrors.NewNotFoundError(errors.New("not found"))
var errAlreadyExists = errors.New("already exists")
var errGone = customerrors.NewGoneError(errors.New("shurl has been deleted"))

type InMemoryRepository struct {
	mu     sync.RWMutex
	shURLs map[string]ShURLEntry
}

type ShURLEntry struct {
	ShURL   entities.ShURL
	Deleted bool
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		shURLs: make(map[string]ShURLEntry),
	}
}

func (m *InMemoryRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shurls := make([]entities.ShURL, 0, len(m.shURLs))
	for _, entry := range m.shURLs {
		if !entry.Deleted {
			shurls = append(shurls, entry.ShURL)
		}
	}

	return shurls, nil
}

func (m *InMemoryRepository) GetByCondition(ctx context.Context, key string, value string) ([]entities.ShURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	//Через рефлексию проверяем удовлетворение условия (допустимо только потому что все поля ShURL строковые)
	shurls := make([]entities.ShURL, 0, len(m.shURLs))
	for _, entry := range m.shURLs {
		val := reflect.ValueOf(entry.ShURL)
		if !entry.Deleted && val.FieldByName(key).String() == value {
			shurls = append(shurls, entry.ShURL)
		}
	}

	return shurls, nil
}

func (m *InMemoryRepository) GetById(ctx context.Context, token string) (*entities.ShURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, exists := m.shURLs[token]; exists {
		if !entry.Deleted {
			return &entry.ShURL, nil
		} else {
			return nil, errGone
		}

	}
	return nil, errNotFound
}

func (m *InMemoryRepository) Create(ctx context.Context, shURL *entities.ShURL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if foundShURL, exists := m.shURLs[shURL.Token]; exists && !foundShURL.Deleted {
		return errAlreadyExists
	}

	m.shURLs[shURL.Token] = ShURLEntry{*shURL, false}
	return nil
}

func (m *InMemoryRepository) Update(ctx context.Context, shURL *entities.ShURL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if foundShURL, exists := m.shURLs[shURL.Token]; exists {
		m.shURLs[shURL.Token] = ShURLEntry{*shURL, foundShURL.Deleted}
		return nil
	} else {
		return errNotFound
	}
}

func (m *InMemoryRepository) Delete(ctx context.Context, tokens []string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, token := range tokens {
		if entry, exists := m.shURLs[token]; exists && entry.ShURL.CreatedBy == userID {
			m.shURLs[token] = ShURLEntry{entry.ShURL, true}
		}
	}
	return nil
}

func (m *InMemoryRepository) PingDB() bool {
	return m.shURLs != nil
}

func (m *InMemoryRepository) CloseConnection() {
}
