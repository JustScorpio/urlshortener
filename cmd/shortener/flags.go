// Пакет Main
package main

import (
	"flag"
	"strings"
)

var (
	// flagShortenerRouterAddr - адрес и порт для запуска сервера
	flagShortenerRouterAddr string

	// flagGRPCRouterAddr - адрес и порт для запуска Grpc сервера
	flagGRPCRouterAddr string

	// flagRedirectRouterAddr - базовый адрес сокращённого URL (часть перед токеном)
	flagRedirectRouterAddr string

	// flagDbFilePath - файл базы данных (для .json БД)
	flagDBFilePath string

	// flagDBConnStr - строка подключения к БД (для postgres)
	flagDBConnStr string

	// flagDBConnStr - включение HTTPS
	flagEnableHTTPS bool //UNDONE: нет проверки что flagShortenerRouterAddr и flagRedirectRouterAddr начинаются с http/https

	// flagConfigPath - путь до конфигурационного файла
	flagConfigPath string

	// flagTrustedSubnet - адрес доверенного сервера
	flagTrustedSubnet string
)

// parseFlags - обрабатывает аргументы командной строки и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagShortenerRouterAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&flagRedirectRouterAddr, "b", "localhost:8080", "base address and port for shortened URLs")
	flag.StringVar(&flagGRPCRouterAddr, "g", "localhost:5051", "address and port for grpc server")
	flag.StringVar(&flagDBFilePath, "f", "data/shortener.json", "path to .json database file (only for .json database)")
	flag.StringVar(&flagDBConnStr, "d", "", "postgresql connection string (only for postgresql)")
	flag.BoolVar(&flagEnableHTTPS, "s", false, "enable https")
	flag.StringVar(&flagConfigPath, "c", "", "path to application config file")
	flag.StringVar(&flagTrustedSubnet, "t", "", "trusted subnet")
	flag.Parse()

	flagShortenerRouterAddr = normalizeHTTPAddress(flagShortenerRouterAddr)
	flagRedirectRouterAddr = normalizeHTTPAddress(flagRedirectRouterAddr)
	flagGRPCRouterAddr = normalizeGRPCAddress(flagGRPCRouterAddr)
}

// normalizeHTTPAddress - нормализация адресов
func normalizeHTTPAddress(addr string) string {

	// Добавляем порт, если его нет
	if !strings.Contains(addr, ":") {
		addr += ":8080"
	}
	// Меняем 127.0.0.1 на localhost
	addr = strings.Replace(addr, "127.0.0.1", "localhost", 1)

	return addr
}

// normalizeHTTPAddress - нормализация адресов
func normalizeGRPCAddress(addr string) string {

	// Убираем часть http:// и https://
	if strings.HasPrefix(addr, "http://") {
		addr = strings.Replace(addr, "http://", "", 1)
	}
	if strings.HasPrefix(addr, "https://") {
		addr = strings.Replace(addr, "https://", "", 1)
	}

	// Добавляем порт, если его нет
	if !strings.Contains(addr, ":") {
		addr += ":5051"
	}
	// Меняем 127.0.0.1 на localhost
	addr = strings.Replace(addr, "127.0.0.1", "localhost", 1)

	return addr
}
