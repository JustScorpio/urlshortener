package services

import (
	"context"
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository"
	"github.com/jaevor/go-nanoid"
)

type ShURLService struct {
	//ВАЖНО: В Go интерфейсы УЖЕ ЯВЛЯЮТСЯ ССЫЛОЧНЫМ ТИПОМ (под капотом — указатель на структуру)
	repo repository.IRepository[entities.ShURL]
}

func NewShURLService(repo repository.IRepository[entities.ShURL]) *ShURLService {
	return &ShURLService{repo: repo}
}

func (s *ShURLService) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	return s.repo.GetAll(ctx)
}

func (s *ShURLService) Get(ctx context.Context, token string) (*entities.ShURL, error) {
	return s.repo.Get(ctx, token)
}

func (s *ShURLService) Create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {

	// Проверка наличие урла в БД
	existedURLs, err := s.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	longURL := newURL.LongURL

	for _, existedURL := range existedURLs {
		// Проверяем не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		//TODO: если разные пользователи укоротили один урл, дубль должен писаться? По идее да
		if existedURL.LongURL == longURL {
			return &existedURL, customerrors.NewAlreadyExistsError(fmt.Errorf("shurl for %v already exists", longURL))
		}
	}

	//Добавление shurl в БД
	generate, _ := nanoid.CustomASCII("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	token := generate() // Пример: "EwHXdJfB"
	shurl := entities.ShURL{
		Token:     token,
		LongURL:   longURL,
		CreatedBy: newURL.CreatedBy,
	}

	err = s.repo.Create(ctx, &shurl)
	if err != nil {
		return nil, err
	}

	return &shurl, nil
}

func (s *ShURLService) Update(ctx context.Context, shurl *entities.ShURL) error {
	return s.repo.Update(ctx, shurl)
}

func (s *ShURLService) Delete(ctx context.Context, token string, userID string) error {
	return s.repo.Delete(ctx, token)

	// Недочитал условия задачи в 15 инкременте :(
	// shURLToDelete, err := s.repo.Get(ctx, token)
	// if err != nil {
	// 	return err
	// }

	// if shURLToDelete.CreatedBy == userID {
	// 	return s.repo.Delete(ctx, token)
	// }

	// return customerrors.NewNotAllowedError(fmt.Errorf("shurl can be deleted only by its creator"))
}

func (s *ShURLService) DeleteAllShURLsByUserID(ctx context.Context, userID string) ([]entities.ShURL, error) {
	shURLsToDelete, err := s.GetAllShURLsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	//TODO: в идеале объединить в одну транзакцию
	for _, shURLToDelete := range shURLsToDelete {
		s.repo.Delete(ctx, shURLToDelete.Token)
	}

	return shURLsToDelete, nil
}

func (s *ShURLService) GetAllShURLsByUserID(ctx context.Context, userID string) ([]entities.ShURL, error) {
	allShURLs, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []entities.ShURL
	for _, shURL := range allShURLs {
		// Проверяем не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if shURL.CreatedBy == userID {
			result = append(result, shURL)
		}
	}

	return result, nil
}
