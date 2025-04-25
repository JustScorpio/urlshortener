package handlers

import (
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/JustScorpio/urlshortener/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
	db map[string]models.ShURL
}

func (r *MockRepository) GetAll() ([]models.ShURL, error) {
	return slices.Collect(maps.Values(r.db)), nil
}

func (r *MockRepository) Get(id string) (*models.ShURL, error) {
	val, exists := r.db[id]
	if !exists {
		return nil, errors.New("Entry not found")
	}

	return &val, nil
}

func (r *MockRepository) Create(shurl *models.ShURL) error {
	if _, exists := r.db[shurl.Token]; exists {
		return errors.New("Entry with such id already exists")
	}

	r.db[shurl.Token] = *shurl
	return nil
}

func (r *MockRepository) Update(shurl *models.ShURL) error {
	if _, exists := r.db[shurl.Token]; !exists {
		return nil
	}

	r.db[shurl.Token] = *shurl
	return nil
}

func (r *MockRepository) Delete(id string) error {
	if _, exists := r.db[id]; !exists {
		return nil
	}

	delete(r.db, id)
	return nil
}

func TestShURLHandler_GetFullURL(t *testing.T) {

	type want struct {
		statusCode int
		fullURL    string
	}

	shurl1 := models.ShURL{Token: "acbdefgh", LongURL: "https://practicum.yandex.ru/"}
	shurl2 := models.ShURL{Token: "bcdefghi", LongURL: "https://www.google.com/"}
	shurl3 := models.ShURL{Token: "cdefghij", LongURL: "https://vk.com/"}
	shurs := map[string]models.ShURL{
		shurl1.Token: shurl1,
		shurl2.Token: shurl2,
		shurl3.Token: shurl3,
	}
	mockRepo := MockRepository{db: shurs}
	mockService := services.NewShURLService(&mockRepo)
	mockHandler := NewShURLHandler(mockService)

	type args struct {
		r *http.Request
	}
	tests := []struct {
		name string
		h    *ShURLHandler
		args args
		want want
	}{
		{
			name: "Positive test #1",
			h:    mockHandler,
			args: args{
				r: httptest.NewRequest(http.MethodGet, "/"+shurl1.Token, nil),
			},
			want: want{
				statusCode: 307,
				fullURL:    shurl1.LongURL,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tt.h.GetFullURL(recorder, tt.args.r)
			result := recorder.Result()

			if !assert.Equal(t, result.StatusCode, tt.want.statusCode) {
				t.Errorf("Status code is not equal to expected")
			}
			if !assert.Equal(t, result.Header.Get("location"), tt.want.fullURL) {
				t.Errorf("Location is not equal to expected")
			}
		})
	}
}
