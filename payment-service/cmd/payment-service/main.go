package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"payment-service/internal/config"
	appdb "payment-service/internal/db"
	"payment-service/internal/grpcclient"
	"payment-service/internal/handler"
	"payment-service/internal/repository"
	"payment-service/internal/service"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	dbPool, err := appdb.NewPostgresPoolWithRetry(ctx, cfg, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	notificationClient, err := grpcclient.NewNotificationClient(cfg.NotificationGRPCAddr)
	if err != nil {
		log.Printf("failed to connect notification-service grpc (%s), continue without notifications: %v", cfg.NotificationGRPCAddr, err)
	}
	if notificationClient != nil {
		defer notificationClient.Close()
	}

	paymentRepo := repository.NewPaymentRepository(dbPool)
	paymentService := service.NewPaymentService(paymentRepo, notificationClient)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	healthHandler := handler.NewHealthHandler(dbPool, cfg.AppVersion)
	wsDemoHandler := handler.NewWSDemoHandler()

	r := chi.NewRouter()
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Get("/payments", paymentHandler.GetPayments)
	r.Get("/payments/{id}", paymentHandler.GetPaymentByID)
	r.Post("/payments", paymentHandler.CreatePayment)
	r.Get("/ws", wsDemoHandler.WS)
	r.Get("/ws/status", wsDemoHandler.Status)
	r.Post("/test/change-pair", wsDemoHandler.ChangePair)

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("payment-service started on port %s", cfg.AppPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server error: %v", err)
	}
}
