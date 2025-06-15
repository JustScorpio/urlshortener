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

var errNotFound = errors.New("not found")
var errAlreadyExists = errors.New("already exists")
var errGone = customerrors.NewGoneError(errors.New("shurl has been deleted"))

type JSONFileShURLRepository struct {
	filePath string
}

type ShURLEntry struct {
	ShURL   entities.ShURL
	Deleted bool
}

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

func (r *JSONFileShURLRepository) Delete(ctx context.Context, ids []string) error {
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

			if entry.ShURL.Token == id {
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

func (r *JSONFileShURLRepository) CloseConnection() {
	//Nothing
}

func (r *JSONFileShURLRepository) PingDB() bool {
	_, err := os.Stat(r.filePath)
	return err == nil
}
