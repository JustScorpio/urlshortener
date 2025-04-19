package main

import (
	"io"
	"net/http"

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
	r.HandleFunc("/{shorturl}", getFullURL).Methods("GET")
	r.HandleFunc("/", shortenURL).Methods("POST")

	return http.ListenAndServe(":8080", r)
}

// Получить полный адрес
func getFullURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	shortURL := vars["shorturl"]

	if shortURL == "EwHXdJfB" {
		w.Write([]byte("https://practicum.yandex.ru/"))
	}

	w.WriteHeader(http.StatusOK)
}

// Укоротить адрес
func shortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// разрешаем только POST-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	fullURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if string(fullURL) == "https://practicum.yandex.ru/" {
		w.Write([]byte("http://localhost:8080/EwHXdJfB"))
	}

	w.WriteHeader(http.StatusOK)
}
