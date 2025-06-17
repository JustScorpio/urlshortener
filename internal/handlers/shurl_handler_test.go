package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	mock.Mock
	db map[string]ShURLEntry
}

type ShURLEntry struct {
	ShURL   entities.ShURL
	Deleted bool
}

func (r *MockRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	var result []entities.ShURL
	for _, entry := range r.db {
		if !entry.Deleted {
			result = append(result, entry.ShURL)
		}
	}

	return result, nil
}

func (r *MockRepository) Get(ctx context.Context, id string) (*entities.ShURL, error) {
	val, exists := r.db[id]
	if !exists {
		return nil, errors.New("Entry not found")
	}

	return &val.ShURL, nil
}

func (r *MockRepository) Create(ctx context.Context, shurl *entities.ShURL) error {
	if _, exists := r.db[shurl.Token]; exists && !r.db[shurl.Token].Deleted {
		return errors.New("Entry with such id already exists")
	}

	newEntry := ShURLEntry{ShURL: *shurl, Deleted: false}
	r.db[shurl.Token] = newEntry
	return nil
}

func (r *MockRepository) Update(ctx context.Context, shurl *entities.ShURL) error {
	if _, exists := r.db[shurl.Token]; !exists && !r.db[shurl.Token].Deleted {
		return nil
	}

	r.db[shurl.Token] = ShURLEntry{ShURL: *shurl, Deleted: false}
	return nil
}

func (r *MockRepository) Delete(ctx context.Context, ids []string, userID string) error {
	for _, id := range ids {
		entry := r.db[id]
		if entry.ShURL.CreatedBy == userID {
			r.db[id] = ShURLEntry{ShURL: r.db[id].ShURL, Deleted: true}
		}
	}

	return nil
}

func (r *MockRepository) CloseConnection() {
	//Nothing
}

func (r *MockRepository) PingDB() bool {
	return true
}

func TestShURLHandler_GetFullURL(t *testing.T) {

	type want struct {
		statusCode int
		location   string
	}

	shurl1 := entities.ShURL{Token: "acbdefgh", LongURL: "https://practicum.yandex.ru/"}
	shurl2 := entities.ShURL{Token: "bcdefghi", LongURL: "https://www.google.com/"}
	shurl3 := entities.ShURL{Token: "cdefghij", LongURL: "https://vk.com/"}
	shurlsEntries := map[string]ShURLEntry{
		shurl1.Token: {shurl1, false},
		shurl2.Token: {shurl2, false},
		shurl3.Token: {shurl3, false},
	}
	mockRepo := MockRepository{db: shurlsEntries}
	mockService := services.NewShURLService(&mockRepo)
	mockHandler := NewShURLHandler(mockService, "localhost:8080")

	type args struct {
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Test #1: positive",
			args: args{
				r: httptest.NewRequest(http.MethodGet, "/"+shurl1.Token, nil),
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   shurl1.LongURL,
			},
		},
		{
			name: "Test #2: token not exists",
			args: args{
				r: httptest.NewRequest(http.MethodGet, "/incorrecttoken", nil),
			},
			want: want{
				statusCode: http.StatusInternalServerError,
				location:   "", //при не 307 не имеет значения
			},
		},
		{
			name: "Test #3: method not allowed",
			args: args{
				r: httptest.NewRequest(http.MethodPost, "/"+shurl1.Token, nil),
			},
			want: want{
				statusCode: http.StatusMethodNotAllowed,
				location:   "", //при не 307 не имеет значения
			},
		},
		{
			name: "Test #4: bad request",
			args: args{
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			want: want{
				statusCode: http.StatusBadRequest,
				location:   "", //при не 307 не имеет значения
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			mockHandler.GetFullURL(recorder, tt.args.r)
			result := recorder.Result()
			defer result.Body.Close() //Важно - не забывать закрывать!

			require.Equal(t, tt.want.statusCode, result.StatusCode)
			if result.StatusCode == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.want.location, result.Header.Get("location"))
			}
		})
	}
}

func TestShURLHandler_ShortenURL(t *testing.T) {

	type want struct {
		statusCode int
		token      string
	}

	shurl1 := entities.ShURL{Token: "acbdefgh", LongURL: "https://practicum.yandex.ru/"}
	shurl2 := entities.ShURL{Token: "bcdefghi", LongURL: "https://www.google.com/"}
	shurl3 := entities.ShURL{Token: "cdefghij", LongURL: "https://vk.com/"}
	shurlsEntries := map[string]ShURLEntry{
		shurl1.Token: {shurl1, false},
		shurl2.Token: {shurl2, false},
		shurl3.Token: {shurl3, false},
	}
	mockRepo := MockRepository{db: shurlsEntries}
	mockService := services.NewShURLService(&mockRepo)
	mockHandler := NewShURLHandler(mockService, "localhost:8080")

	type args struct {
		method string
		url    string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Test #1: positive",
			args: args{
				method: http.MethodPost,
				url:    "https://metanit.com/",
			},
			want: want{
				statusCode: http.StatusCreated,
				token:      "", //при создании нового не имеет значения
			},
		},
		{
			name: "Test #2: URL already exists",
			args: args{
				method: http.MethodPost,
				url:    shurl1.LongURL,
			},
			want: want{
				statusCode: http.StatusConflict,
				token:      shurl1.Token,
			},
		},
		{
			name: "Test #3: method not allowed",
			args: args{
				method: http.MethodGet,
				url:    "https://metanit.com/",
			},
			want: want{
				statusCode: http.StatusMethodNotAllowed,
				token:      "", //при не 201 не имеет значения
			},
		},
		{
			name: "Test #4: body is empty",
			args: args{
				method: http.MethodPost,
				url:    "",
			},
			want: want{
				statusCode: http.StatusBadRequest,
				token:      "", //при не 201 не имеет значения
			},
		},
		//TODO: Тесты для json-варианта запросов и ответов - придётся переписывать все тесты + сделать для каждого теста свой мок БД
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.args.method, "/", strings.NewReader(tt.args.url))
			mockHandler.ShortenURL(recorder, request)
			result := recorder.Result()
			defer result.Body.Close() //Важно - не забывать закрывать!

			require.Equal(t, tt.want.statusCode, result.StatusCode)
			if result.StatusCode == http.StatusCreated {
				//Проверить что в map есть один и только один заданный урл
				count := 0
				for entry := range shurlsEntries {
					if shurlsEntries[entry].ShURL.LongURL == tt.args.url {
						count++
					}
				}

				assert.Equal(t, 1, count)
			}
		})
	}
}
