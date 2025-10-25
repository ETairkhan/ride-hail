package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"ride-hail/internal/adapter/db"
	"ride-hail/internal/adapter/rabbitmq"
	"ride-hail/internal/app"
	"ride-hail/internal/common/config"
	"syscall"
)

func main() {
	// Загружаем конфигурацию
	ctx := context.Background()
	config := config.LoadConfig()
	fmt.Println(config)

	// Инициализация базы данных
	dbConnection, err := db.InitDB(config.DBConfig, ctx)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer dbConnection.Close(ctx)

	// Инициализация RabbitMQ
	rabbitConnection, err := rabbitmq.InitRabbitMQ(config.RabbitMQConfig)
	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ: %v", err)
	}
	defer rabbitConnection.Close()

	// Запуск сервисов
	go app.RideStartService(config, dbConnection, rabbitConnection)

	// Ожидаем завершения программы по сигналу
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down services...")
}
