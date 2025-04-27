package main

import (
	"flag"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера
var flagRunAddr string

// неэкспортированная переменная flagShURLBaseAddr содержит базовый адрес результирующего сокращённого URL (часть перед токеном)
var flagShURLBaseAddr string

// parseFlags обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", ":8000", "address and port to run server")
	flag.StringVar(&flagShURLBaseAddr, "b", ":8000", "base address and port for shortened URLs")
	flag.Parse()
}
