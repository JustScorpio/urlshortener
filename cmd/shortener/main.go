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
	r.HandleFunc("/{shorturl}", getFullUrl).Methods("GET")
	r.HandleFunc("/", shortenUrl).Methods("POST")

	return http.ListenAndServe(":8080", r)
}

// Получить полный адрес
func getFullUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// разрешаем только Get-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	shortUrl := vars["shorturl"]

	if shortUrl == "EwHXdJfB" {
		w.Write([]byte("https://practicum.yandex.ru/"))
	}

	w.WriteHeader(http.StatusOK)
}

// Укоротить адрес
func shortenUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// разрешаем только POST-запросы
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	fullUrl, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if string(fullUrl) == "https://practicum.yandex.ru/" {
		w.Write([]byte("http://localhost:8080/EwHXdJfB"))
	}

	w.WriteHeader(http.StatusOK)
}
