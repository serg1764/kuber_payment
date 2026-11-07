package main

import (
	"context"
	"log"
	"net"
	"time"

	notificationpb "notification-service/gen/notification-service/gen/notification"
	"notification-service/internal/config"
	appdb "notification-service/internal/db"
	"notification-service/internal/grpcserver"
	"notification-service/internal/repository"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// main поднимает gRPC-сервер notification-service и подключает его к Postgres.
func main() {
	cfg := config.Load()
	ctx := context.Background()

	dbPool, err := appdb.NewPostgresPoolWithRetry(ctx, cfg, 10, 2*time.Second)
	// Если не удалось подключиться к БД, завершаем процесс с ошибкой.
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	repo := repository.NewNotificationRepository(dbPool)
	serverImpl := grpcserver.NewNotificationServer(repo)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	// Если не удалось открыть порт для gRPC, завершаем процесс.
	if err != nil {
		log.Fatalf("failed to listen on grpc port %s: %v", cfg.GRPCPort, err)
	}
	defer lis.Close()

	grpcSrv := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(grpcSrv, serverImpl)
	// Включаем gRPC reflection, чтобы удобно вызывать методы через grpcurl.
	reflection.Register(grpcSrv)

	log.Printf("notification-service started on gRPC port %s", cfg.GRPCPort)

	// Запускаем gRPC-сервер и блокируем текущую горутину до остановки процесса.
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
