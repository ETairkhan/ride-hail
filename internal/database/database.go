package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"ride-hail/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
)

type DB struct {
	Pool *pgxpool.Pool
}

type RabbitMQ struct {
	Conn *amqp091.Connection
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.HealthCheckPeriod = 1 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func NewRabbitMQ(cfg config.RabbitMQConfig) (*RabbitMQ, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)

	conn, err := amqp091.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	log.Println("Successfully connected to RabbitMQ")
	return &RabbitMQ{Conn: conn}, nil
}

func (rmq *RabbitMQ) Close() {
	if rmq.Conn != nil {
		rmq.Conn.Close()
	}
}
