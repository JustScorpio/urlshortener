package services

import (
	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/JustScorpio/urlshortener/internal/repository"
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

func (s *ShURLService) Create(shurl *models.ShURL) error {
	return s.repo.Create(shurl)
}

func (s *ShURLService) Update(shurl *models.ShURL) error {
	return s.repo.Update(shurl)
}

func (s *ShURLService) Delete(token string) error {
	return s.repo.Delete(token)
}
