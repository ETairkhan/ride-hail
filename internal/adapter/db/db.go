package db

import (
	"context"
	"fmt"
	"log"
	"ride-hail/pkg/utils"

	"github.com/jackc/pgx/v5"
)

// Инициализация подключения к базе данных
func InitDB(config utils.Config, ctx context.Context) (*pgx.Conn, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	log.Println("Connected to the database")
	return conn, nil
}
