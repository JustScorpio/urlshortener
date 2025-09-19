// Пакет jsonfile содержит репозиторий, который хранит данные в виде json-файла
package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	_ "modernc.org/sqlite"
)

// Кастомные типы ошибок, возвращаемых некоторыми из функций пакета
var (
	errNotFound      = errors.New("not found")
	errAlreadyExists = errors.New("already exists")
	errGone          = customerrors.NewGoneError(errors.New("shurl has been deleted"))
)

// JSONFileShURLRepository - репозиторий
type JSONFileShURLRepository struct {
	filePath string
}

// ShURLEntry - расширение ShURL с информацией о том удалена ли сущность
type ShURLEntry struct {
	ShURL   entities.ShURL
	Deleted bool
}

// NewJSONFileShURLRepository - инициализация репозитория
func NewJSONFileShURLRepository(filePath string) (*JSONFileShURLRepository, error) {
	// Создаем директорию, если ее нет
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Создаем пустой файл БД, если ее нет
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		emptyJSONCollection, _ := json.Marshal([]entities.ShURL{})
		err = os.WriteFile(filePath, emptyJSONCollection, 0644)

		if err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return &JSONFileShURLRepository{filePath: filePath}, nil
}

// GetAllEntries - получить все сущности из json-файла
// В отличие от GetAll возвращает []ShURLEntry которые содержат метку удаления deleted
func (r *JSONFileShURLRepository) GetAllEntries(ctx context.Context) ([]ShURLEntry, error) {

	var file, err = os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Проверяем, не отменен ли контекст пока читали файл
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var entries []ShURLEntry
	if err := json.Unmarshal(file, &entries); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return entries, nil
}

// GetAll - получить все ShURL
// Возвращает ShURL'ы, у которых deleted = false
func (r *JSONFileShURLRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	var entries, err = r.GetAllEntries(ctx)
	if err != nil {
		return nil, err
	}

	var shurls []entities.ShURL
	for _, entry := range entries {
		if !entry.Deleted {
			shurls = append(shurls, entry.ShURL)
		}
	}

	return shurls, nil
}

// Get - получить ShURL по ID (токену)
func (r *JSONFileShURLRepository) Get(ctx context.Context, id string) (*entities.ShURL, error) {
	entries, err := r.GetAllEntries(ctx)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Проверяем, не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if entry.ShURL.Token == id {
			if entry.Deleted {
				return nil, errGone
			}

			return &entry.ShURL, nil
		}
	}

	return nil, errNotFound
}

// Create - создать ShURL
func (r *JSONFileShURLRepository) Create(ctx context.Context, shurl *entities.ShURL) error {
	//При работе с json-файлом перезаписывается всё содержимое, поэтому работаем с ShURLEntry чтобы не потерять удалённые записи
	entries, err := r.GetAllEntries(ctx)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Проверяем, не отменен ли контекст перед началом работы
		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.ShURL.Token == shurl.Token && !entry.Deleted {
			return errAlreadyExists
		}
	}

	entries = append(entries, ShURLEntry{*shurl, false})

	jsonShurls, err := json.MarshalIndent(entries, "", "   ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, jsonShurls, 0644)
}

// Update - обновить ShURL
func (r *JSONFileShURLRepository) Update(ctx context.Context, shurl *entities.ShURL) error {
	//При работе с json-файлом перезаписывается всё содержимое, поэтому работаем с ShURLEntry чтобы не потерять удалённые записи
	entries, err := r.GetAllEntries(ctx)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		// Проверяем, не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.ShURL.Token == shurl.Token && !entry.Deleted {
			entries[i].ShURL = *shurl

			jsonShurls, err := json.MarshalIndent(entries, "", "   ")
			if err != nil {
				return err
			}

			return os.WriteFile(r.filePath, jsonShurls, 0644)
		}
	}

	return errNotFound
}

// Delete - удалить ShURL
func (r *JSONFileShURLRepository) Delete(ctx context.Context, ids []string, userID string) error {
	//При работе с json-файлом перезаписывается всё содержимое, поэтому работаем с ShURLEntry чтобы не потерять удалённые записи
	entries, err := r.GetAllEntries(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		for i, entry := range entries {
			// Проверяем, не отменен ли контекст перед началом работы
			if err := ctx.Err(); err != nil {
				return err
			}

			if entry.ShURL.Token == id && entry.ShURL.CreatedBy == userID {
				entries[i].Deleted = true
				break
			}
		}
	}

	jsonShurls, err := json.MarshalIndent(entries, "", "   ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, jsonShurls, 0644)
}

// CloseConnection - закрыть соединение с базой данных
func (r *JSONFileShURLRepository) CloseConnection() {
	//Nothing
}

// PingDB - проверить подключение к базе данных
func (r *JSONFileShURLRepository) PingDB() bool {
	_, err := os.Stat(r.filePath)
	return err == nil
}
