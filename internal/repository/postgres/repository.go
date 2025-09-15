package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/jackc/pgx/v5"
)

//КАК ЗАКОММЕНТИРОВАТЬ КОММЕНТАРИЙ go:embed config.json
// var configContent []byte

// type DBConfiguration struct {
// 	Host     string `json:"host"`
// 	User     string `json:"user"`
// 	Password string `json:"password"`
// 	DBName   string `json:"dbname"`
// 	Port     string `json:"port"`
// 	SslMode  string `json:"sslmode"`
// }

type PostgresShURLRepository struct {
	db *pgx.Conn
}

var errGone = customerrors.NewGoneError(errors.New("shurl has been deleted"))

func NewPostgresShURLRepository(connStr string) (*PostgresShURLRepository, error) {
	//Если передана пустая строка - парсим конфиг
	// var conf DBConfiguration
	// if connStr == "" {
	// 	if err := json.Unmarshal(configContent, &conf); err != nil {
	// 		return nil, fmt.Errorf("failed to decode config: %w", err)
	// 	}

	// 	connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", conf.Host, conf.User, conf.Password, conf.DBName, conf.Port, conf.SslMode)
	// }

	// // Создание базы данных (Закомментировано т.к. в тестах используется уже созданная)
	// defaultDB, err := pgx.Connect(context.Background(), connStr)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to default database: %w", err)
	// }
	// defer defaultDB.Close(context.Background())

	// // Проверка и создание базы данных
	// var dbExists bool
	// err = defaultDB.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", conf.DBName).Scan(&dbExists)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to check database existence: %w", err)
	// }

	// // Создание базы данных, если она не существует
	// if !dbExists {
	// 	_, err = defaultDB.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", conf.DBName))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create database: %w", err)
	// 	}
	// }

	// Подключение к базе данных
	db, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	//Проверка подключения
	if err = db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Создание таблицы shurls, если её нет
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS shurls (
			token VARCHAR(8) PRIMARY KEY,
			longurl TEXT NOT NULL UNIQUE,
			createdby TEXT NOT NULL,
			deleted BOOLEAN DEFAULT false
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table shurls: %w", err)
	}

	return &PostgresShURLRepository{db: db}, nil
}

func (r *PostgresShURLRepository) GetAll(ctx context.Context) ([]entities.ShURL, error) {
	rows, err := r.db.Query(ctx, "SELECT token, longurl, createdby FROM shurls WHERE deleted = false")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	var shurls []entities.ShURL
	for rows.Next() {
		var shurl entities.ShURL
		err := rows.Scan(&shurl.Token, &shurl.LongURL, &shurl.CreatedBy)
		if err != nil {
			return nil, err
		}
		shurls = append(shurls, shurl)
	}

	return shurls, nil
}

func (r *PostgresShURLRepository) GetByCondition(ctx context.Context, key string, value string) ([]entities.ShURL, error) {
	rows, err := r.db.Query(ctx, "SELECT token, longurl, createdby FROM shurls WHERE deleted = false AND $1 = $2", key, value)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	var shurls []entities.ShURL
	for rows.Next() {
		var shurl entities.ShURL
		err := rows.Scan(&shurl.Token, &shurl.LongURL, &shurl.CreatedBy)
		if err != nil {
			return nil, err
		}
		shurls = append(shurls, shurl)
	}

	return shurls, nil
}

func (r *PostgresShURLRepository) GetById(ctx context.Context, id string) (*entities.ShURL, error) {
	var shurl entities.ShURL
	var deleted bool
	err := r.db.QueryRow(ctx, "SELECT token, longurl, createdby, deleted FROM shurls WHERE token = $1", id).Scan(&shurl.Token, &shurl.LongURL, &shurl.CreatedBy, &deleted)

	if deleted {
		return nil, errGone
	}

	if err != nil {
		return nil, err
	}
	return &shurl, nil
}

func (r *PostgresShURLRepository) Create(ctx context.Context, shurl *entities.ShURL) error {
	_, err := r.db.Exec(ctx, "INSERT INTO shurls (token, longurl, createdBy) VALUES ($1, $2, $3)", shurl.Token, shurl.LongURL, shurl.CreatedBy)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostgresShURLRepository) Update(ctx context.Context, shurl *entities.ShURL) error {
	_, err := r.db.Exec(ctx, "UPDATE shurls SET longurl = $2, createdby = $3 WHERE token = $1", shurl.Token, shurl.LongURL, shurl.CreatedBy)
	return err
}

func (r *PostgresShURLRepository) Delete(ctx context.Context, ids []string, userID string) error {
	_, err := r.db.Exec(ctx, "UPDATE shurls SET deleted = true WHERE token = ANY($1) AND createdby = $2", ids, userID)
	return err
}

func (r *PostgresShURLRepository) CloseConnection() {
	r.db.Close(context.Background())
}

func (r *PostgresShURLRepository) PingDB() bool {
	err := r.db.Ping(context.Background())
	return err == nil
}
