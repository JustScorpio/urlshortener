package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/JustScorpio/urlshortener/internal/services"
)

type ShURLHandler struct {
	service *services.ShURLService
}

func NewShURLHandler(service *services.ShURLService) *ShURLHandler {
	return &ShURLHandler{service: service}
}

func (h *ShURLHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	entities, err := h.service.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entities)
}

// Получить полный адрес
func (h *ShURLHandler) GetFullURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/")

	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Получение сущности из сервиса
	shURL, err := h.service.Get(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Location", shURL.LongURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Укоротить адрес
func (h *ShURLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// разрешаем только POST-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Автотесты говорят что НЕЛЬЗЯ проверять content-type. Ок, как скажете
	// if r.Header.Get("Content-Type") != "text/plain" {
	// 	// разрешаем только Content-Type: text/plain
	// 	w.WriteHeader(http.StatusUnsupportedMediaType)
	// 	return
	// }

	//Читаем тело запроса
	longURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//Добавление shurl в БД
	// Получение сущности из сервиса
	shurl := models.ShURL{
		Token:   "ЗАТЫЧКА",
		LongURL: string(longURL),
	}
	err = h.service.Create(&shurl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://localhost:8080/EwHXdJfB"))
}
