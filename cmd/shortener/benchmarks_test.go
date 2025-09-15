package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository/inmemory"
	"github.com/JustScorpio/urlshortener/internal/services"
)

func BenchmarkShURLService_Create(b *testing.B) {
	//Пустой сервер в отдельной горутине без хэндлеров для pprof
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newURL := dtos.NewShURL{
			LongURL:   fmt.Sprint("https://example.com/", i),
			CreatedBy: "user1",
		}
		service.Create(ctx, newURL)
	}

	b.StopTimer()

	// Даем время на сбор данных pprof
	time.Sleep(3 * time.Second)
}

func BenchmarkShURLService_Get(b *testing.B) {
	//Пустой сервер в отдельной горутине без хэндлеров для pprof
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	ctx := context.Background()

	// Сначала создаем URL для тестирования
	newURL := dtos.NewShURL{
		LongURL:   "https://example.com",
		CreatedBy: "user1",
	}
	shURL, _ := service.Create(ctx, newURL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetById(ctx, shURL.Token)
	}

	b.StopTimer()

	// Даем время на сбор данных pprof
	time.Sleep(3 * time.Second)
}

func BenchmarkShURLService_GetAllByUserID(b *testing.B) {
	//Пустой сервер в отдельной горутине без хэндлеров для pprof
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	mockRepo := inmemory.NewInMemoryRepository()
	service := services.NewShURLService(mockRepo)
	ctx := context.Background()

	// Создаем несколько URL для пользователя
	for i := 0; i < 100; i++ {
		newURL := dtos.NewShURL{
			LongURL:   "https://example.com",
			CreatedBy: "user1",
		}
		service.Create(ctx, newURL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetByCondition(ctx, entities.ShURLCreatedByFieldName, "user1")
	}

	b.StopTimer()

	// Даем время на сбор данных pprof
	time.Sleep(3 * time.Second)
}
