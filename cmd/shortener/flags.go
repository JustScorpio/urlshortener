// Пакет Main
package main

import (
	"flag"
	"strings"
)

var (
	// flagShortenerRouterAddr - адрес и порт для запуска сервера
	flagShortenerRouterAddr string

	// flagRedirectRouterAddr - базовый адрес сокращённого URL (часть перед токеном)
	flagRedirectRouterAddr string

	// flagDbFilePath - файл базы данных (для .json БД)
	flagDBFilePath string

	// flagDBConnStr - строка подключения к БД (для postgres)
	flagDBConnStr string

	// flagDBConnStr - включение HTTPS
	flagEnableHTTPS bool

	// flagConfigPath - путь до конфигурационного файла
	flagConfigPath string

	// flagTrustedSubnet - адрес доверенного сервера
	flagTrustedSubnet string
)

// parseFlags - обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagShortenerRouterAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagRedirectRouterAddr, "b", ":8080", "base address and port for shortened URLs")
	flag.StringVar(&flagDBFilePath, "f", "data/shortener.json", "path to .json database file (only for .json database)")
	flag.StringVar(&flagDBConnStr, "d", "", "postgresql connection string (only for postgresql)")
	flag.BoolVar(&flagEnableHTTPS, "s", false, "enable https")
	flag.StringVar(&flagConfigPath, "c", "", "path to application config file")
	flag.StringVar(&flagTrustedSubnet, "t", "", "trusted subnet")
	flag.Parse()

	flagShortenerRouterAddr = normalizeAddress(flagShortenerRouterAddr)
	flagRedirectRouterAddr = normalizeAddress(flagRedirectRouterAddr)
}

// normalizeAddress - нормализация адресов
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
