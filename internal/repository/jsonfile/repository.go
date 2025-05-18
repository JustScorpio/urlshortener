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

type JsonFileShURLRepository struct {
	filePath string
}

func NewJsonFileShURLRepository(filePath string) (*JsonFileShURLRepository, error) {
	// Создаем директорию для БД, если ее нет
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(filePath), 0755)

		if err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// открываем файл для чтения и записи в конец
	// file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	//     return nil, err
	// }

	return &JsonFileShURLRepository{filePath: filePath}, nil
}

func (r *JsonFileShURLRepository) GetAll() ([]models.ShURL, error) {

	var file, err = os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения файла: %w", err)
	}

	var shurls []models.ShURL
	if err := json.Unmarshal(file, &shurls); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return shurls, nil
}

func (r *JsonFileShURLRepository) Get(id string) (*models.ShURL, error) {
	shurls, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	for _, shurl := range shurls {
		if shurl.Token == id {
			return &shurl, nil
		}
	}

	return nil, errors.New("Not Found")
}

func (r *JsonFileShURLRepository) Create(shurl *models.ShURL) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for _, existedShurl := range existedShurls {
		if existedShurl.Token == shurl.Token {
			return errors.New("Already exists")
		}
	}

	existedShurls = append(existedShurls, *shurl)

	jsonShurls, err := json.MarshalIndent(existedShurls, "", ", ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, jsonShurls, 0644)
}

func (r *JsonFileShURLRepository) Update(shurl *models.ShURL) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for _, existedShurl := range existedShurls {
		if existedShurl.Token == shurl.Token {
			existedShurl = *shurl

			jsonShurls, err := json.MarshalIndent(existedShurls, "", ", ")
			if err != nil {
				return err
			}

			return os.WriteFile(r.filePath, jsonShurls, 0644)
		}
	}

	return errors.New("NotFound")
}

func (r *JsonFileShURLRepository) Delete(id string) error {
	existedShurls, err := r.GetAll()
	if err != nil {
		return err
	}

	for i, existedShurl := range existedShurls {
		if existedShurl.Token == id {
			existedShurls[i] = existedShurls[len(existedShurls)-1]

			//Возвращаем slice без последнего элемента, где удаляемый элемент заменён последним
			jsonShurls, err := json.MarshalIndent(existedShurls[:len(existedShurls)-1], "", ", ")
			if err != nil {
				return err
			}

			return os.WriteFile(r.filePath, jsonShurls, 0644)
		}
	}

	return errors.New("NotFound")
}
