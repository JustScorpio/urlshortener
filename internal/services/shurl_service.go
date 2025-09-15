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

type ShURLService struct {
	//ВАЖНО: В Go интерфейсы УЖЕ ЯВЛЯЮТСЯ ССЫЛОЧНЫМ ТИПОМ (под капотом — указатель на структуру)
	repo      repository.IRepository[entities.ShURL]
	taskQueue chan Task // канал-очередь задач
}

type TaskType int

const (
	TaskGetAll TaskType = iota
	TaskGetByCondition
	TaskGetById
	TaskCreate
	TaskUpdate
	TaskDelete
)

type Task struct {
	Type     TaskType
	Context  context.Context
	Payload  interface{}
	ResultCh chan TaskResult
}

type TaskGetByConditionPayload struct {
	Key   string
	Value string
}

type TaskResult struct {
	Result interface{}
	Err    error
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
		case TaskGetByCondition:
			kayValuePair := task.Payload.(TaskGetByConditionPayload)
			result, err = s.repo.GetByCondition(task.Context, kayValuePair.Key, kayValuePair.Value)
		case TaskGetById:
			token := task.Payload.(string)
			result, err = s.repo.GetById(task.Context, token)
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
		}

		if task.ResultCh != nil {
			switch task.Type {
			case TaskGetAll, TaskGetByCondition, TaskGetById, TaskCreate:
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

// Поставить задачу в очередь
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

func (s *ShURLService) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGetAll,
		Context: ctx,
	})

	return res.([]entities.ShURL), err
}

func (s *ShURLService) GetByCondition(ctx context.Context, key string, value string) ([]entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGetByCondition,
		Context: ctx,
		Payload: TaskGetByConditionPayload{key, value},
	})

	return res.([]entities.ShURL), err
}

func (s *ShURLService) GetById(ctx context.Context, token string) (*entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskGetById,
		Context: ctx,
		Payload: token,
	})

	return res.(*entities.ShURL), err
}

func (s *ShURLService) Create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	res, err := s.enqueueTask(Task{
		Type:    TaskCreate,
		Context: ctx,
		Payload: &newURL,
	})

	return res.(*entities.ShURL), err
}

func (s *ShURLService) create(ctx context.Context, newURL dtos.NewShURL) (*entities.ShURL, error) {
	// Проверка наличие урла в БД
	existedURLs, err := s.repo.GetByCondition(ctx, entities.ShURLLongURLFieldName, newURL.LongURL)
	if err != nil {
		return nil, err
	}

	//Если есть дубли - отработает один раз только для первого
	for _, existedURL := range existedURLs {
		//TODO: если разные пользователи укоротили один урл, дубль должен писаться? По идее да
		return &existedURL, alreadyExistsError
	}

	//Добавление shurl в БД
	generate, _ := nanoid.CustomASCII("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	token := generate() // Пример: "EwHXdJfB"
	shurl := entities.ShURL{
		Token:     token,
		LongURL:   newURL.LongURL,
		CreatedBy: newURL.CreatedBy,
	}

	err = s.repo.Create(ctx, &shurl)
	if err != nil {
		return nil, err
	}

	return &shurl, nil
}

// func (s *ShURLService) Update(ctx context.Context, shurl *entities.ShURL) error {
// 	_, err := s.enqueueTask(Task{
// 		Type:    TaskUpdate,
// 		Context: ctx,
// 		Payload: shurl,
// 	})

// 	return err
// }

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
