// Пакет Main
package main

import (
	"encoding/json"
	"os"
)

// parseAppConfig - обрабатывает параметры запуска приложения из конфигурационного файла
func parseAppConfig(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var appConfig struct {
		ServerAddress   string `json:"server_address"`
		BaseURL         string `json:"base_url"`
		FileStoragePath string `json:"file_storage_path"`
		DatabaseDSN     string `json:"database_dsn"`
		EnableHTTPS     bool   `json:"enable_https"`
	}

	err = json.Unmarshal(content, &appConfig)
	if err != nil {
		return err
	}

	flagShortenerRouterAddr = appConfig.ServerAddress
	flagRedirectRouterAddr = appConfig.BaseURL
	flagDBFilePath = appConfig.FileStoragePath
	flagDBConnStr = appConfig.DatabaseDSN
	flagEnableHTTPS = appConfig.EnableHTTPS

	return nil
}
