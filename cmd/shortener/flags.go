package main

import (
	"flag"
	"strings"
)

// flagShortenerRouterAddr содержит адрес и порт для запуска сервера
var flagShortenerRouterAddr string

// flagRedirectRouterAddr содержит базовый адрес результирующего сокращённого URL (часть перед токеном)
var flagRedirectRouterAddr string

// flagDbFilePath содержит путь до файла базы данных (для .json БД)
var flagDBFilePath string

// flagDBConnStr содержит строку подключения к БД (для postgres)
var flagDBConnStr string

// parseFlags обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagShortenerRouterAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagRedirectRouterAddr, "b", ":8080", "base address and port for shortened URLs")
	flag.StringVar(&flagDBFilePath, "f", "data/shortener.json", "path to .json database file (only for .json database)")
	flag.StringVar(&flagDBConnStr, "d", "", "postgresql connection string (only for postgresql)")
	flag.Parse()

	flagShortenerRouterAddr = normalizeAddress(flagShortenerRouterAddr)
	flagRedirectRouterAddr = normalizeAddress(flagRedirectRouterAddr)
}

// Нормализация адресов
func normalizeAddress(addr string) string {

	// Добавляем порт, если его нет
	if !strings.Contains(addr, ":") {
		addr += ":8080"
	}

	// Убираем часть http://
	if strings.HasPrefix(addr, "http://") {
		addr = strings.Replace(addr, "http://", "", 1)
	}

	// Убираем 127.0.0.1 и localhost
	if strings.HasPrefix(addr, "127.0.0.1:") {
		addr = strings.Replace(addr, "127.0.0.1", "", 1)
	}
	if strings.HasPrefix(addr, "localhost:") {
		addr = strings.Replace(addr, "localhost", "", 1)
	}

	return addr
}
