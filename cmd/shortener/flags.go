package main

import (
	"flag"
)

// неэкспортированная переменная flagShortenerRouterAddr содержит адрес и порт для запуска сервера
var flagShortenerRouterAddr string

// неэкспортированная переменная flagRedirectRouterAddr содержит базовый адрес результирующего сокращённого URL (часть перед токеном)
var flagRedirectRouterAddr string

// parseFlags обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagShortenerRouterAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagRedirectRouterAddr, "b", ":8080", "base address and port for shortened URLs")
	flag.Parse()

	flagShortenerRouterAddr = normalizeAddress(flagShortenerRouterAddr)
	flagRedirectRouterAddr = normalizeAddress(flagRedirectRouterAddr)
}
