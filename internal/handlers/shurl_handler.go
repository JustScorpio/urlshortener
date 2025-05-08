package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/JustScorpio/urlshortener/internal/services"
	"github.com/jaevor/go-nanoid"
)

type ShURLHandler struct {
	service       *services.ShURLService
	shURLBaseAddr string
}

func NewShURLHandler(service *services.ShURLService, shURLBaseAddr string) *ShURLHandler {
	return &ShURLHandler{
		service:       service,
		shURLBaseAddr: shURLBaseAddr,
	}
}

// Получить полный адрес
func (h *ShURLHandler) GetFullURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/")
	//token := chi.URLParam(r, "token") //Not works. Known chi issue (https://github.com/go-chi/chi/issues/938)

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

	//Читаем тело запроса
	longURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//Если Body пуст
	if len(longURL) == 0 {
		http.Error(w, "Body is empty", http.StatusBadRequest)
		return
	}

	// Проверяем наличие урла в БД
	existedURLs, err := h.service.GetAll()
	if err != nil {
		http.Error(w, "Failed to check existed urls: "+err.Error(), http.StatusBadRequest)
		return
	}

	token := ""
	for _, existedURL := range existedURLs {
		if existedURL.LongURL == string(longURL) {
			token = existedURL.Token
			break
		}
	}

	if token == "" {
		//Добавление shurl в БД
		generate, _ := nanoid.CustomASCII("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
		token = generate() // Пример: "EwHXdJfB"

		shurl := models.ShURL{
			Token:   token,
			LongURL: string(longURL),
		}
		err = h.service.Create(&shurl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte("http://" + h.shURLBaseAddr + "/" + token))

	//Пишем Content-type После w.Write чтобы jsonpacker первым записал туда "application/json"
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
}
