package services

import (
	"context"
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository"
	"github.com/jaevor/go-nanoid"
	"github.com/pkg/errors"
)

type ShURLService struct {
	//ВАЖНО: В Go интерфейсы УЖЕ ЯВЛЯЮТСЯ ССЫЛОЧНЫМ ТИПОМ (под капотом — указатель на структуру)
	repo      repository.IRepository[entities.ShURL]
	taskQueue chan Task // канал-очередь задач
}

type TaskType int

const (
	TaskGetAll TaskType = iota
	TaskGet
	TaskCreate
	TaskUpdate
	TaskDelete
	TaskGetByUserID
)

type Task struct {
	Type     TaskType
	Context  context.Context
	Payload  interface{}
	ResultCh chan interface{}
}

var alreadyExistsError = customerrors.NewAlreadyExistsError(errors.New("shurl already exists"))

func NewShURLService(repo repository.IRepository[entities.ShURL]) *ShURLService {
	service := &ShURLService{
		repo:      repo,
		taskQueue: make(chan Task, 300),
	}

	go service.taskProcessor()

	return service
}

func (s *ShURLService) taskProcessor() {
	for task := range s.taskQueue {

		var result interface{}
		var err error

		switch task.Type {
		case TaskGetAll:
			result, err = s.repo.GetAll(task.Context)
		case TaskGet:
			token := task.Payload.(string)
			result, err = s.repo.Get(task.Context, token)
		case TaskCreate:
			shURL := task.Payload.(*dtos.NewShURL)
			result, err = s.сreate(task.Context, *shURL)
		case TaskUpdate:
			shURL := task.Payload.(*entities.ShURL)
			err = s.repo.Update(task.Context, shURL)
		case TaskDelete:
			payload := task.Payload.(struct {
				tokens []string
				userID string
			})
			err = s.repo.Delete(task.Context, payload.tokens, payload.userID)
		case TaskGetByUserID:
			userID := task.Payload.(string)
			result, err = s.getAllByUserID(task.Context, userID)
		}

		if task.ResultCh != nil {
			if err != nil {
				task.ResultCh <- err
			} else {
				task.ResultCh <- result
			}
			close(task.ResultCh)
		}
	}
}

// Поставить задачу в очередь
func (s *ShURLService) enqueueTask(task Task) interface{} {
	if task.ResultCh == nil {
		task.ResultCh = make(chan interface{}, 1)
	}

	s.taskQueue <- task

	select {
	case <-task.Context.Done():
		return task.Context.Err()
	case result := <-task.ResultCh:
		return result
	}
}

func (s *ShURLService) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	result := s.enqueueTask(Task{
		Type:    TaskGetAll,
		Context: ctx,
	})

	switch v := result.(type) {
	case error:
		return nil, v
	case []entities.ShURL:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected result type")
	}
}

func (s *ShURLService) Get(ctx context.Context, token string) (*entities.ShURL, error) {
	result := s.enqueueTask(Task{
		Type:    TaskGet,
		Context: ctx,
		Payload: token,
	})

	switch v := result.(type) {
	case error:
		return nil, v
	case *entities.ShURL:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected result type")
	}
}

func (s *ShURLService) Create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	result := s.enqueueTask(Task{
		Type:    TaskCreate,
		Context: ctx,
		Payload: &newURL,
	})

	switch v := result.(type) {
	case error:
		return nil, v
	case *entities.ShURL:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected result type")
	}
}

func (s *ShURLService) сreate(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	// Проверка наличие урла в БД
	existedURLs, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	longURL := newURL.LongURL
	createdBy := newURL.CreatedBy

	for _, existedURL := range existedURLs {
		// Проверяем не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		//TODO: если разные пользователи укоротили один урл, дубль должен писаться? По идее да
		if existedURL.LongURL == longURL && existedURL.CreatedBy == createdBy {
			return &existedURL, alreadyExistsError
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
	result := s.enqueueTask(Task{
		Type:    TaskUpdate,
		Context: ctx,
		Payload: shurl,
	})

	if err, ok := result.(error); ok {
		return err
	}
	return nil
}

func (s *ShURLService) Delete(ctx context.Context, tokens []string, userID string) error {
	result := s.enqueueTask(Task{
		Type:    TaskDelete,
		Context: ctx,
		Payload: struct {
			tokens []string
			userID string
		}{tokens, userID},
	})

	if err, ok := result.(error); ok {
		return err
	}
	return nil
}

func (s *ShURLService) GetAllShURLsByUserID(ctx context.Context, userID string) ([]entities.ShURL, error) {
	result := s.enqueueTask(Task{
		Type:    TaskGetByUserID,
		Context: ctx,
		Payload: userID,
	})

	switch v := result.(type) {
	case error:
		return nil, v
	case []entities.ShURL:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected result type")
	}
}

func (s *ShURLService) getAllByUserID(ctx context.Context, userID string) ([]entities.ShURL, error) {
	allShURLs, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []entities.ShURL
	for _, shURL := range allShURLs {
		if shURL.CreatedBy == userID {
			result = append(result, shURL)
		}
	}

	return result, nil
}
