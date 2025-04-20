package sqlite

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed config.json
var configContent []byte

type DbConfiguration struct {
	Host     string
	User     string
	Password string
	DbName   string
	Port     string
	SslMode  string
}

func NewDB() (*sql.DB, error) {
	var conf DbConfiguration
	if err := json.Unmarshal(configContent, &conf); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	var defaultConnString = fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s", conf.Host, conf.User, conf.Password, conf.Port, conf.SslMode)
	defaultDB, err := sql.Open("postgres", defaultConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default database: %w", err)
	}
	defer defaultDB.Close()

	// // Проверка и создание базы данных
	// var dbExists bool
	// err = defaultDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", conf.DbName).Scan(&dbExists)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to check database existence: %w", err)
	// }

	// // Создание базы данных, если она не существует
	// if !dbExists {
	// 	_, err = defaultDB.Exec(fmt.Sprintf("CREATE DATABASE %s", conf.DbName))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create database: %w", err)
	// 	}
	// }

	// Подключение к созданной базе данных
	connString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", conf.Host, conf.User, conf.Password, conf.DbName, conf.Port, conf.SslMode)
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	//Проверка подключения
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// // Создание таблицы Countries, если её нет
	// _, err = db.Exec(`
	// 	CREATE TABLE IF NOT EXISTS countries (
	// 		id SERIAL PRIMARY KEY,
	// 		name TEXT NOT NULL,
	// 		code VARCHAR(2) UNIQUE NOT NULL
	// 		population INT
	// 	);
	// `)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create table Countries: %w", err)
	// }

	return db, nil
}
