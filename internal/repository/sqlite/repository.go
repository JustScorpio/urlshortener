package sqlite

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JustScorpio/urlshortener/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed config.json
var configContent []byte

type DBConfiguration struct {
	Path string `json:"path"`
}

type SQLiteShURLRepository struct {
	db *sql.DB
}

func NewSQLiteShURLRepository() (*SQLiteShURLRepository, error) {
	var conf DBConfiguration
	if err := json.Unmarshal(configContent, &conf); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Создаем директорию для БД, если ее нет
	if err := os.MkdirAll(filepath.Dir(conf.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Открываем (или создаем) базу данных
	db, err := sql.Open("sqlite", "file:"+conf.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Включаем foreign keys и другие настройки SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	// Создаем таблицу
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shurls (
			token TEXT PRIMARY KEY,
			longurl TEXT NOT NULL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table shurls: %w", err)
	}

	return &SQLiteShURLRepository{db: db}, nil
}

func (r *SQLiteShURLRepository) GetAll() ([]models.ShURL, error) {
	rows, err := r.db.Query("SELECT token, longurl FROM shurls")
	if err != nil {
		return nil, err
	}
	if err = rows.Err(); err != nil {
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

func (r *SQLiteShURLRepository) Get(id string) (*models.ShURL, error) {
	var shurl models.ShURL
	err := r.db.QueryRow(
		"SELECT token, longurl FROM shurls WHERE token = ?",
		id,
	).Scan(&shurl.Token, &shurl.LongURL)

	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

func (r *SQLiteShURLRepository) Create(shurl *models.ShURL) error {
	_, err := r.db.Exec(
		"INSERT INTO shurls (token, longurl) VALUES (?, ?)",
		shurl.Token,
		shurl.LongURL,
	)
	return err
}

func (r *SQLiteShURLRepository) Update(shurl *models.ShURL) error {
	_, err := r.db.Exec(
		"UPDATE shurls SET longurl = ? WHERE token = ?",
		shurl.LongURL,
		shurl.Token,
	)
	return err
}

func (r *SQLiteShURLRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM shurls WHERE token = ?", id)
	return err
}

func (r *SQLiteShURLRepository) CloseConnection() {
	r.db.Close()
}

func (r *SQLiteShURLRepository) PingDB() bool {
	err := r.db.Ping()
	return err == nil
}
