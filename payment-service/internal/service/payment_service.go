package service

import (
	"context"
	"log"
	"time"

	"payment-service/internal/grpcclient"
	"payment-service/internal/repository"
)

type PaymentService struct {
	repo               *repository.PaymentRepository
	notificationClient *grpcclient.NotificationClient
}

type NotificationStatus string

const (
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
	NotificationStatusSkipped NotificationStatus = "skipped"
)

func NewPaymentService(repo *repository.PaymentRepository, notificationClient *grpcclient.NotificationClient) *PaymentService {
	return &PaymentService{
		repo:               repo,
		notificationClient: notificationClient,
	}
}

func (s *PaymentService) GetPayments(ctx context.Context) ([]repository.Payment, error) {
	return s.repo.GetPayments(ctx)
}

func (s *PaymentService) GetPaymentByID(ctx context.Context, id int64) (repository.Payment, error) {
	return s.repo.GetPaymentByID(ctx, id)
}

func (s *PaymentService) CreatePayment(ctx context.Context, req repository.CreatePaymentRequest) (repository.Payment, NotificationStatus, error) {
	payment, err := s.repo.CreatePayment(ctx, req)
	if err != nil {
		return repository.Payment{}, "", err
	}

	if s.notificationClient != nil {
		sendErr := s.notificationClient.SendPaymentStatus(ctx, payment.ID, payment.OrderID, payment.Status, payment.Provider, time.Now().UTC())
		if sendErr != nil {
			log.Printf("failed to send payment status to notification-service: %v", sendErr)
			return payment, NotificationStatusFailed, nil
		}
		return payment, NotificationStatusSent, nil
	}

	return payment, NotificationStatusSkipped, nil
}
