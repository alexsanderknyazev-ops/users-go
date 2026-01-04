package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"users/database"
	"users/grpc"
	"users/router"
)

func main() {
	// Канал для graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Инициализируем БД
	database.InitDB()
	log.Default().Println("main - Init DB")

	// Запускаем GRPC сервер в отдельной горутине
	grpcErr := make(chan error, 1)
	go func() {
		port := "50063" // GRPC порт
		log.Printf("Starting Users GRPC server on :%s", port)
		grpcErr <- grpc.StartServer(port)
	}()

	// Даем время на запуск GRPC сервера
	time.Sleep(100 * time.Millisecond)

	// Запускаем HTTP сервер
	httpErr := make(chan error, 1)
	go func() {
		r := router.Route()
		log.Default().Println("main - Init Route")

		port := ":8072" // HTTP порт
		log.Printf("HTTP Server starting on %s", port)

		httpErr <- http.ListenAndServe(port, r)
	}()

	// Ждем сигнал завершения или ошибку
	select {
	case sig := <-stopChan:
		log.Printf("Received signal: %v. Shutting down...", sig)
		time.Sleep(2 * time.Second)
	case err := <-httpErr:
		log.Printf("HTTP server error: %v", err)
	case err := <-grpcErr:
		log.Printf("GRPC server error: %v", err)
	}

	log.Println("Users service shutdown complete")
}