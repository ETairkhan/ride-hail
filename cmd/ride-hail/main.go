package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"ride-hail/internal/ride"
	"ride-hail/pkg/db"
	"ride-hail/pkg/rabbitmq"
	"ride-hail/pkg/utils"
	"syscall"
)

func main() {
	// Загружаем конфигурацию
	ctx := context.Background()
	config := utils.LoadConfig()

	// Инициализация базы данных
	dbConnection, err := db.InitDB(config, ctx)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer dbConnection.Close(ctx)

	// Инициализация RabbitMQ
	rabbitConnection, err := rabbitmq.InitRabbitMQ(config)
	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ: %v", err)
	}
	defer rabbitConnection.Close()

	// Запуск сервисов
	go ride.StartService(config, dbConnection, rabbitConnection)

	// Ожидаем завершения программы по сигналу
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down services...")
}
