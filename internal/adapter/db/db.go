package db

import (
	"context"
	"fmt"
	"log"
	"ride-hail/internal/common/config"

	"github.com/jackc/pgx/v5"
)

// Инициализация подключения к базе данных
func InitDB(config config.DBConfig, ctx context.Context) (*pgx.Conn, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.User, config.Password, config.Host, config.Port, config.Name)

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	log.Println("Connected to the database")
	return conn, nil
}
