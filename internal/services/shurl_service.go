package services

import (
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/JustScorpio/urlshortener/internal/repository"
	"github.com/jaevor/go-nanoid"
)

type ShURLService struct {
	//ВАЖНО: В Go интерфейсы УЖЕ ЯВЛЯЮТСЯ ССЫЛОЧНЫМ ТИПОМ (под капотом — указатель на структуру)
	repo repository.IRepository[models.ShURL]
}

func NewShURLService(repo repository.IRepository[models.ShURL]) *ShURLService {
	return &ShURLService{repo: repo}
}

func (s *ShURLService) GetAll() ([]models.ShURL, error) {
	return s.repo.GetAll()
}

func (s *ShURLService) Get(token string) (*models.ShURL, error) {
	return s.repo.Get(token)
}

func (s *ShURLService) Create(longURL string) (*models.ShURL, error) {

	// Проверка наличие урла в БД
	existedURLs, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	for _, existedURL := range existedURLs {
		if existedURL.LongURL == string(longURL) {
			return &existedURL, customerrors.NewAlreadyExistsError(fmt.Errorf("shurl for %v already exists", longURL))
		}
	}

	//Добавление shurl в БД
	generate, _ := nanoid.CustomASCII("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	token := generate() // Пример: "EwHXdJfB"
	shurl := models.ShURL{
		Token:   token,
		LongURL: longURL,
	}

	err = s.repo.Create(&shurl)
	if err != nil {
		return nil, err
	}

	return &shurl, nil
}

func (s *ShURLService) Update(shurl *models.ShURL) error {
	return s.repo.Update(shurl)
}

func (s *ShURLService) Delete(token string) error {
	return s.repo.Delete(token)
}
