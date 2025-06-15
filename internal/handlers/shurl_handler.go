package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/services"
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
	shURL, err := h.service.Get(r.Context(), token)
	if err != nil {
		var statusCode = http.StatusInternalServerError

		//Если запрашивается shURL c deleted = true, вернётся ошибка с кодом 410
		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			statusCode = httpErr.Code
		}

		http.Error(w, err.Error(), statusCode)
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//Если Body пуст
	if len(body) == 0 {
		http.Error(w, "Body is empty", http.StatusBadRequest)
		return
	}

	//Проверяем и при необходимости ивзлекаем URL из JSON
	var longURL string
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var reqData struct {
			URL string `json:"url"`
		}

		if err = json.Unmarshal(body, &reqData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Конвертируем в строку
		longURL = reqData.URL
	} else {
		longURL = string(body)
	}

	userID := customcontext.GetUserID(r.Context())

	//Создаём shurl
	shurl, err := h.service.Create(r.Context(), dtos.NewShURL{
		LongURL:   longURL,
		CreatedBy: userID,
	})

	//Определяем статус код
	statusCode := http.StatusCreated
	if err != nil {
		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			statusCode = httpErr.Code
		}
	}

	var responseBody []byte
	//Если в при создании возникла ошибка, shurl может быть пуст => тело тоже пусто
	if shurl != nil {
		shortURL := "http://" + h.shURLBaseAddr + "/" + shurl.Token

		//Если Header "Accept" == "application/json" - возвращаем ввиде json
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "application/json") {
			// Конвертируем plain text в JSON
			var respData struct {
				Result string `json:"result"`
			}

			respData.Result = shortURL
			jsonData, err := json.Marshal(respData)
			if err != nil {
				return
			}

			w.Header().Add("Content-Type", "application/json")
			responseBody = jsonData
		} else {
			responseBody = []byte(shortURL)
		}
	}

	//Content-type по умолчанию text/plain
	// w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	w.Write(responseBody)
}

// Укоротить пачку адресов
func (h *ShURLHandler) ShortenURLsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// разрешаем только POST-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	//Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//Если Body пуст
	if len(body) == 0 {
		http.Error(w, "Body is empty", http.StatusBadRequest)
		return
	}

	//Только Content-Type: JSON
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	type reqItem struct {
		ID  string `json:"correlation_id"`
		URL string `json:"original_url"`
	}
	var reqData []reqItem

	type respItem struct {
		ID  string `json:"correlation_id"`
		URL string `json:"short_url"`
	}
	var respData []respItem

	//Извлекаем URL из JSON
	if err = json.Unmarshal(body, &reqData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := customcontext.GetUserID(r.Context())

	for _, reqItem := range reqData {
		longURL := reqItem.URL
		shurl, err := h.service.Create(r.Context(), dtos.NewShURL{
			LongURL:   longURL,
			CreatedBy: userID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respData = append(respData, respItem{
			ID:  reqItem.ID,
			URL: "http://" + h.shURLBaseAddr + "/" + shurl.Token,
		})
	}

	//Ответ только в "application/json"
	jsonData, err := json.Marshal(respData)
	if err != nil {
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonData)
}

// Получить полный адрес
func (h *ShURLHandler) GetShURLsByUserID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	//Не предусмотрено тестами
	//Только Accept: JSON
	// contentType := r.Header.Get("Accept")
	// if contentType != "application/json" {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	userID := customcontext.GetUserID(r.Context())
	if userID == "" {
		// UserID в куке пуст
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Получение сущностей из сервиса
	shURLs, err := h.service.GetAllShURLsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(shURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	type respItem struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	var respData []respItem

	for _, shURL := range shURLs {
		respData = append(respData, respItem{
			ShortURL:    "http://" + h.shURLBaseAddr + "/" + shURL.Token,
			OriginalURL: shURL.LongURL,
		})
	}

	jsonData, err := json.Marshal(respData)
	if err != nil {
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// Удалить ShURLы Пользователя
func (h *ShURLHandler) DeleteShURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		// разрешаем только Delete-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	//Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//Если Body пуст
	if len(body) == 0 {
		http.Error(w, "Body is empty", http.StatusBadRequest)
		return
	}

	var tokens []string

	//Извлекаем токены из JSON
	if err = json.Unmarshal(body, &tokens); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := customcontext.GetUserID(r.Context())
	if userID == "" {
		// UserID в куке пуст
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Удаление сущностей и сохранение удалённых сущностей в deletedShURLs
	err = h.service.DeleteAllShURLs(r.Context(), userID, tokens)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}
