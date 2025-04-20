package postgres

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/models"
	_ "github.com/lib/pq"
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

type PostgresShURLRepository struct {
	Db *sql.DB
}

func NewPostgresShURLRepository() (*PostgresShURLRepository, error) {
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

	// Проверка и создание базы данных
	var dbExists bool
	err = defaultDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", conf.DbName).Scan(&dbExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	}

	// Создание базы данных, если она не существует
	if !dbExists {
		_, err = defaultDB.Exec(fmt.Sprintf("CREATE DATABASE %s", conf.DbName))
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	}

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

	// Создание таблицы shurls, если её нет
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shurls (
			token VARCHAR(8) PRIMARY KEY,
			longurl TEXT NOT NULL,
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table shurls: %w", err)
	}

	return &PostgresShURLRepository{Db: db}, nil
}

func (r *PostgresShURLRepository) GetAll() ([]models.ShURL, error) {
	rows, err := r.Db.Query("SELECT token, longurl FROM shurls")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shurls []models.ShURL
	for rows.Next() {
		var shurl models.ShURL
		err := rows.Scan(&shurl.Token, &shurl.LongURL)
		if err != nil {
			return nil, err
		}
		shurls = append(shurls, shurl)
	}

	return shurls, nil
}

func (r *PostgresShURLRepository) Get(id string) (*models.ShURL, error) {
	var shurl models.ShURL
	err := r.Db.QueryRow("SELECT token, longurl, FROM shurls WHERE token = $1", id).Scan(&shurl.Token, &shurl.LongURL)
	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

func (r *PostgresShURLRepository) Create(shurl *models.ShURL) error {
	err := r.Db.QueryRow("INSERT INTO shurls (token, longurl) VALUES ($1, $2) RETURNING token", shurl.Token, shurl.LongURL).Scan(&shurl.Token)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostgresShURLRepository) Update(shurl *models.ShURL) error {
	_, err := r.Db.Exec("UPDATE shurls SET longurl = $2 WHERE token = $1", shurl.Token, shurl.LongURL)
	return err
}

func (r *PostgresShURLRepository) Delete(id string) error {
	_, err := r.Db.Exec("DELETE FROM countries WHERE token = $1", id)
	return err
}
