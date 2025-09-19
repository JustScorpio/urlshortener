// Пакет services содержит структуры и методы, реализующие бизнес-логику приложения
package services

import (
	"context"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository"
	"github.com/jaevor/go-nanoid"
	"github.com/pkg/errors"
)

// ShURLService - сервис-укорачиватель ссылок
type ShURLService struct {
	//ВАЖНО: В Go интерфейсы УЖЕ ЯВЛЯЮТСЯ ССЫЛОЧНЫМ ТИПОМ (под капотом — указатель на структуру)
	repo      repository.IRepository[entities.ShURL]
	taskQueue chan Task // канал-очередь задач
}

// TaskType - алиас вокруг int, для описания типов задачи в очереди задач
type TaskType int

// Типы задач TaskType
const (
	TaskGetAll TaskType = iota
	TaskGet
	TaskCreate
	TaskUpdate
	TaskDelete
	TaskGetByUserID
)

// Task - задача в очереди задач на обработку сервисом
type Task struct {
	Type     TaskType
	Context  context.Context
	Payload  interface{}
	ResultCh chan TaskResult
}

// TaskResult - результат обработки задачи Task
type TaskResult struct {
	Result interface{}
	Err    error
}

// Кастомные типы ошибок, возвращаемых некоторыми из функций пакета
var (
	alreadyExistsError = customerrors.NewAlreadyExistsError(errors.New("shurl already exists"))
)

// NewShURLService - инициализация сервиса-укорачивателя ссылок
func NewShURLService(repo repository.IRepository[entities.ShURL]) *ShURLService {
	service := &ShURLService{
		repo:      repo,
		taskQueue: make(chan Task, 300),
	}

	go service.taskProcessor()

	return service
}

// taskProcessor - обработчик очереди задач в составе ShURLService
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
			result, err = s.create(task.Context, *shURL)
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
			switch task.Type {
			case TaskGetAll, TaskGet, TaskGetByUserID, TaskCreate:
				task.ResultCh <- TaskResult{
					Result: result,
					Err:    err,
				}
			case TaskUpdate, TaskDelete:
				task.ResultCh <- TaskResult{
					Err: err,
				}
			}
			close(task.ResultCh)
		}
	}
}

// enqueueTask - поставить задачу в очередь
func (s *ShURLService) enqueueTask(task Task) (interface{}, error) {
	if task.ResultCh == nil {
		task.ResultCh = make(chan TaskResult, 1)
	}

	s.taskQueue <- task

	select {
	case <-task.Context.Done():
		return nil, task.Context.Err()
	case res := <-task.ResultCh:
		return res.Result, res.Err
	}
}

// GetAll - получить все ShURL
func (s *ShURLService) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGetAll,
		Context: ctx,
	})

	return res.([]entities.ShURL), err
}

// Get - получить ShURL по ID (токену)
func (s *ShURLService) Get(ctx context.Context, token string) (*entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGet,
		Context: ctx,
		Payload: token,
	})

	return res.(*entities.ShURL), err
}

// Create - создать ShURL
func (s *ShURLService) Create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskCreate,
		Context: ctx,
		Payload: &newURL,
	})

	return res.(*entities.ShURL), err
}

// Update - обновить ShURL
// func (s *ShURLService) Update(ctx context.Context, shurl *entities.ShURL) error {
// 	_, err := s.enqueueTask(Task{
// 		Type:    TaskUpdate,
// 		Context: ctx,
// 		Payload: shurl,
// 	})

// 	return err
// }

// Delete - удалить ShURL
func (s *ShURLService) Delete(ctx context.Context, tokens []string, userID string) error {
	_, err := s.enqueueTask(Task{
		Type:    TaskDelete,
		Context: ctx,
		Payload: struct {
			tokens []string
			userID string
		}{tokens, userID},
	})

	return err
}

// GetAllShURLsByUserID - получить все ShURL конкретного пользователя
func (s *ShURLService) GetAllShURLsByUserID(ctx context.Context, userID string) ([]entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGetByUserID,
		Context: ctx,
		Payload: userID,
	})

	return res.([]entities.ShURL), err
}

// create - создать ShURL (инкапсулирует все проверки бизнес-логику)
func (s *ShURLService) create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	// Проверка наличие урла в БД
	existedURLs, err := s.repo.GetAll(ctx)
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

// GetAllShURLsByUserID - получить все ShURL конкретного пользователя (инкапсулирует все проверки бизнес-логику)
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
