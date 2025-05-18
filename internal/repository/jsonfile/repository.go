package jsonfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JustScorpio/urlshortener/internal/models"
	_ "modernc.org/sqlite"
)

type JSONFileShURLRepository struct {
	filePath string
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
		emptyJSONCollection, _ := json.Marshal([]models.ShURL{})
		err = os.WriteFile(filePath, emptyJSONCollection, 0644)

		if err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return &JSONFileShURLRepository{filePath: filePath}, nil
}

func (r *JSONFileShURLRepository) GetAll() ([]models.ShURL, error) {

	var file, err = os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	var shurls []models.ShURL
	if err := json.Unmarshal(file, &shurls); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return shurls, nil
}

func (r *JSONFileShURLRepository) Get(id string) (*models.ShURL, error) {
	shurls, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, shurl := range shurls {
		if shurl.Token == id {
			return &shurl, nil
		}
	}

	return nil, errors.New("not Found")
}

func (r *JSONFileShURLRepository) Create(shurl *models.ShURL) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for _, existedShurl := range existedShurls {
		if existedShurl.Token == shurl.Token {
			return errors.New("already exists")
		}
	}

	existedShurls = append(existedShurls, *shurl)

	jsonShurls, err := json.MarshalIndent(existedShurls, "", "   ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, jsonShurls, 0644)
}

func (r *JSONFileShURLRepository) Update(shurl *models.ShURL) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for i, existedShurl := range existedShurls {
		if existedShurl.Token == shurl.Token {
			existedShurls[i] = *shurl

			jsonShurls, err := json.MarshalIndent(existedShurls, "", "   ")
			if err != nil {
				return err
			}

			return os.WriteFile(r.filePath, jsonShurls, 0644)
		}
	}

	return errors.New("not found")
}

func (r *JSONFileShURLRepository) Delete(id string) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for i, existedShurl := range existedShurls {
		if existedShurl.Token == id {
			existedShurls[i] = existedShurls[len(existedShurls)-1]

			//Возвращаем slice без последнего элемента, где удаляемый элемент заменён последним
			jsonShurls, err := json.MarshalIndent(existedShurls[:len(existedShurls)-1], "", "   ")
			if err != nil {
				return err
			}

			return os.WriteFile(r.filePath, jsonShurls, 0644)
		}
	}

	return errors.New("not found")
}

func (r *JSONFileShURLRepository) CloseConnection() {
	//Nothing
}
