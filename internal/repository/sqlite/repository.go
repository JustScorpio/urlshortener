package sqlite

import (
	"context"
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
	//TODO: задействовать context при создании, подключении БД

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

func (r *SQLiteShURLRepository) GetAll(ctx context.Context) ([]models.ShURL, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT token, longurl FROM shurls")
	if err != nil {
		return nil, err
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	defer rows.Close()

	var shurls []models.ShURL
	for rows.Next() {
		// Проверяем не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var shurl models.ShURL
		err := rows.Scan(&shurl.Token, &shurl.LongURL)
		if err != nil {
			return nil, err
		}
		shurls = append(shurls, shurl)
	}

	return shurls, nil
}

func (r *SQLiteShURLRepository) Get(ctx context.Context, id string) (*models.ShURL, error) {
	var shurl models.ShURL
	err := r.db.QueryRowContext(
		ctx,
		"SELECT token, longurl FROM shurls WHERE token = ?",
		id,
	).Scan(&shurl.Token, &shurl.LongURL)

	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

func (r *SQLiteShURLRepository) Create(ctx context.Context, shurl *models.ShURL) error {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO shurls (token, longurl) VALUES (?, ?)",
		shurl.Token,
		shurl.LongURL,
	)
	return err
}

func (r *SQLiteShURLRepository) Update(ctx context.Context, shurl *models.ShURL) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE shurls SET longurl = ? WHERE token = ?",
		shurl.LongURL,
		shurl.Token,
	)
	return err
}

func (r *SQLiteShURLRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM shurls WHERE token = ?", id)
	return err
}

func (r *SQLiteShURLRepository) CloseConnection(ctx context.Context) {
	//TODO: задействовать context при хакрытии соединения с БД (если это уместно при закрытии соединения)
	r.db.Close()
}

func (r *SQLiteShURLRepository) PingDB(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}
