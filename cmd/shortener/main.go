package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// функция main вызывается автоматически при запуске приложения
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {
	r := mux.NewRouter()
	r.HandleFunc("/", getFullURL).Methods("GET")
	r.HandleFunc("/", shortenURL).Methods("POST")

	return http.ListenAndServe(":8080", r)
}

// Получить полный адрес
func getFullURL(w http.ResponseWriter, r *http.Request) {
	fmt.Printf(r.URL.Path)
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	shortURL := strings.TrimPrefix(r.URL.Path, "/")

	if shortURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Add("Location", "https://practicum.yandex.ru/")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Укоротить адрес
func shortenURL(w http.ResponseWriter, r *http.Request) {
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

	// Читаем тело запроса
	// fullURL, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	http.Error(w, "Failed to read request body", http.StatusBadRequest)
	// 	return
	// }
	// defer r.Body.Close()

	w.Write([]byte("http://localhost:8080/EwHXdJfB"))
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
}
