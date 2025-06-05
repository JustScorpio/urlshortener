package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/models"
	"github.com/jackc/pgx/v5"
)

//go:embed config.json
var configContent []byte

type DBConfiguration struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	Port     string `json:"port"`
	SslMode  string `json:"sslmode"`
}

type PostgresShURLRepository struct {
	db *pgx.Conn
}

func NewPostgresShURLRepository(ctx context.Context, connStr string) (*PostgresShURLRepository, error) {
	//Если передана пустая строка - парсим конфиг
	// var conf DBConfiguration
	// if connStr == "" {
	// 	if err := json.Unmarshal(configContent, &conf); err != nil {
	// 		return nil, fmt.Errorf("failed to decode config: %w", err)
	// 	}

	// 	connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", conf.Host, conf.User, conf.Password, conf.DBName, conf.Port, conf.SslMode)
	// }

	//Создание базы данных (Закомментировано т.к. в тестах используется уже созданная)
	// defaultDB, err := pgx.Connect(ctx, connStr)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to default database: %w", err)
	// }
	// defer defaultDB.Close(ctx)

	// // Проверка и создание базы данных
	// var dbExists bool
	// err = defaultDB.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", conf.DBName).Scan(&dbExists)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to check database existence: %w", err)
	// }

	// // Создание базы данных, если она не существует
	// if !dbExists {
	// 	_, err = defaultDB.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", conf.DBName))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create database: %w", err)
	// 	}
	// }

	// Подключение к базе данных
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	//Проверка подключения
	if err = db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Создание таблицы shurls, если её нет
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS shurls (
			token VARCHAR(8) PRIMARY KEY,
			longurl TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table shurls: %w", err)
	}

	return &PostgresShURLRepository{db: db}, nil
}

func (r *PostgresShURLRepository) GetAll(ctx context.Context) ([]models.ShURL, error) {
	rows, err := r.db.Query(ctx, "SELECT token, longurl FROM shurls")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		return nil, err
	}

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

func (r *PostgresShURLRepository) Get(ctx context.Context, id string) (*models.ShURL, error) {
	var shurl models.ShURL
	err := r.db.QueryRow(ctx, "SELECT token, longurl FROM shurls WHERE token = $1", id).Scan(&shurl.Token, &shurl.LongURL)
	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

func (r *PostgresShURLRepository) Create(ctx context.Context, shurl *models.ShURL) error {
	_, err := r.db.Exec(ctx, "INSERT INTO shurls (token, longurl) VALUES ($1, $2)", shurl.Token, shurl.LongURL)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostgresShURLRepository) Update(ctx context.Context, shurl *models.ShURL) error {
	_, err := r.db.Exec(ctx, "UPDATE shurls SET longurl = $2 WHERE token = $1", shurl.Token, shurl.LongURL)
	return err
}

func (r *PostgresShURLRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM countries WHERE token = $1", id)
	return err
}

func (r *PostgresShURLRepository) CloseConnection(ctx context.Context) {
	r.db.Close(ctx)
}

func (r *PostgresShURLRepository) PingDB(ctx context.Context) bool {
	err := r.db.Ping(ctx)
	return err == nil
}
