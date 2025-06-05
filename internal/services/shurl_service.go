package services

import (
	"context"
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

func (s *ShURLService) GetAll(ctx context.Context) ([]models.ShURL, error) {
	return s.repo.GetAll(ctx)
}

func (s *ShURLService) Get(ctx context.Context, token string) (*models.ShURL, error) {
	return s.repo.Get(ctx, token)
}

func (s *ShURLService) Create(ctx context.Context, longURL string) (*models.ShURL, error) {

	// Проверка наличие урла в БД
	existedURLs, err := s.GetAll(ctx)
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

	err = s.repo.Create(ctx, &shurl)
	if err != nil {
		return nil, err
	}

	return &shurl, nil
}

func (s *ShURLService) Update(ctx context.Context, shurl *models.ShURL) error {
	return s.repo.Update(ctx, shurl)
}

func (s *ShURLService) Delete(ctx context.Context, token string) error {
	return s.repo.Delete(ctx, token)
}
