package main

import (
	"flag"
)

// неэкспортированная переменная flagShortenerAddr содержит адрес и порт для запуска сервера
var flagShortenerAddr string

// неэкспортированная переменная flagShURLBaseAddr содержит базовый адрес результирующего сокращённого URL (часть перед токеном)
var flagShURLBaseAddr string

// parseFlags обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagShortenerAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagShURLBaseAddr, "b", ":8080", "base address and port for shortened URLs")
	flag.Parse()
}
