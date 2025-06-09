package handlers

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
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
	db map[string]entities.ShURL
}

func (r *MockRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	return slices.Collect(maps.Values(r.db)), nil
}

func (r *MockRepository) Get(ctx context.Context, id string) (*entities.ShURL, error) {
	val, exists := r.db[id]
	if !exists {
		return nil, errors.New("Entry not found")
	}

	return &val, nil
}

func (r *MockRepository) Create(ctx context.Context, shurl *entities.ShURL) error {
	if _, exists := r.db[shurl.Token]; exists {
		return errors.New("Entry with such id already exists")
	}

	r.db[shurl.Token] = *shurl
	return nil
}

func (r *MockRepository) Update(ctx context.Context, shurl *entities.ShURL) error {
	if _, exists := r.db[shurl.Token]; !exists {
		return nil
	}

	r.db[shurl.Token] = *shurl
	return nil
}

func (r *MockRepository) Delete(ctx context.Context, id string) error {
	if _, exists := r.db[id]; !exists {
		return nil
	}

	delete(r.db, id)
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
	shurls := map[string]entities.ShURL{
		shurl1.Token: shurl1,
		shurl2.Token: shurl2,
		shurl3.Token: shurl3,
	}
	mockRepo := MockRepository{db: shurls}
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
	shurls := map[string]entities.ShURL{
		shurl1.Token: shurl1,
		shurl2.Token: shurl2,
		shurl3.Token: shurl3,
	}
	mockRepo := MockRepository{db: shurls}
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
				for entry := range shurls {
					if shurls[entry].LongURL == tt.args.url {
						count++
					}
				}

				assert.Equal(t, 1, count)
			}
		})
	}
}
