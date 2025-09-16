// Пакет sqlite содержит репозиторий, который хранит данные в SQLite-базе данных
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	_ "modernc.org/sqlite"
)

// configContent - содержимое конфигурационного файла подключения к базе данных
//
//go:embed config.json
var configContent []byte

// DBConfiguration - конфигурация базы данных считанная из файла
type DBConfiguration struct {
	Path string `json:"path"`
}

// SQLiteShURLRepository - репозиторий
type SQLiteShURLRepository struct {
	db *sql.DB
}

// Кастомные типы ошибок, возвращаемых некоторыми из функций пакета
var (
	errGone = customerrors.NewGoneError(errors.New("shurl has been deleted"))
)

// NewSQLiteShURLRepository - инициализация репозитория
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
			longurl TEXT NOT NULL,
			createdby TEXT NOT NULL,
			deleted BOOLEAN DEFAULT FALSE
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table shurls: %w", err)
	}

	return &SQLiteShURLRepository{db: db}, nil
}

// GetAll - получить все ShURL
func (r *SQLiteShURLRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT token, longurl, createdby FROM shurls WHERE deleted = FALSE")
	if err != nil {
		return nil, err
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	defer rows.Close()

	var shurls []entities.ShURL
	for rows.Next() {
		// Проверяем не отменен ли контекст
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var shurl entities.ShURL
		err := rows.Scan(&shurl.Token, &shurl.LongURL, &shurl.CreatedBy)
		if err != nil {
			return nil, err
		}
		shurls = append(shurls, shurl)
	}

	return shurls, nil
}

// Get - получить ShURL по ID (токену)
func (r *SQLiteShURLRepository) Get(ctx context.Context, id string) (*entities.ShURL, error) {
	var shurl entities.ShURL
	var deleted bool
	err := r.db.QueryRowContext(
		ctx,
		"SELECT token, longurl, createdby, deleted FROM shurls WHERE token = ?",
		id,
	).Scan(&shurl.Token, &shurl.LongURL, &shurl.CreatedBy, &deleted)

	if deleted {
		return nil, errGone
	}

	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

// Create - создать ShURL
func (r *SQLiteShURLRepository) Create(ctx context.Context, shurl *entities.ShURL) error {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO shurls (token, longurl, createdby) VALUES (?, ?, ?)",
		shurl.Token,
		shurl.LongURL,
		shurl.CreatedBy,
	)
	return err
}

// Update - обновить ShURL
func (r *SQLiteShURLRepository) Update(ctx context.Context, shurl *entities.ShURL) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE shurls SET longurl = ?, createdby = ? WHERE token = ?",
		shurl.LongURL,
		shurl.CreatedBy,
		shurl.Token,
	)
	return err
}

// Delete - удалить ShURL
func (r *SQLiteShURLRepository) Delete(ctx context.Context, ids []string, userID string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE shurls SET deleted = TRUE WHERE token = ANY(?) AND createdby = ?", ids, userID)
	return err
}

// CloseConnection - закрыть соединение с базой данных
func (r *SQLiteShURLRepository) CloseConnection() {
	r.db.Close()
}

// PingDB - проверить подключение к базе данных
func (r *SQLiteShURLRepository) PingDB() bool {
	err := r.db.Ping()
	return err == nil
}
