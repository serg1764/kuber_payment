package grpcserver

import (
	"context"
	"strings"
	"time"

	notification "notification-service/gen/notification-service/gen/notification"
	"notification-service/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotificationServer реализует gRPC-методы NotificationService.
type NotificationServer struct {
	notification.UnimplementedNotificationServiceServer
	repo *repository.NotificationRepository
}

// NewNotificationServer создает gRPC-сервер с зависимостью на repository слой.
func NewNotificationServer(repo *repository.NotificationRepository) *NotificationServer {
	return &NotificationServer{repo: repo}
}

// SendPaymentStatus валидирует вход и сохраняет статус платежа в БД.
func (s *NotificationServer) SendPaymentStatus(ctx context.Context, req *notification.SendPaymentStatusRequest) (*notification.SendPaymentStatusResponse, error) {
	// Проверяем, что запрос вообще передан.
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	// Проверяем корректность payment_id.
	if req.GetPaymentId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "payment_id must be greater than 0")
	}

	// Проверяем корректность order_id.
	if req.GetOrderId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "order_id must be greater than 0")
	}

	statusValue := strings.TrimSpace(req.GetStatus())
	// Проверяем, что статус не пустой после trim.
	if statusValue == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	providerValue := strings.TrimSpace(req.GetProvider())
	// Проверяем, что провайдер не пустой после trim.
	if providerValue == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	eventTime, err := time.Parse(time.RFC3339, strings.TrimSpace(req.GetEventTime()))
	// Проверяем, что время события соответствует формату RFC3339.
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "event_time must be valid RFC3339")
	}

	notificationID, err := s.repo.CreatePaymentStatus(ctx, req.GetPaymentId(), req.GetOrderId(), statusValue, providerValue, eventTime)
	// Если запись в БД не удалась, возвращаем internal ошибку gRPC.
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to save payment status")
	}

	return &notification.SendPaymentStatusResponse{
		Ok:             true,
		NotificationId: notificationID,
	}, nil
}

// GetPaymentStatusesByPaymentId читает из БД всю историю статусов по payment_id.
func (s *NotificationServer) GetPaymentStatusesByPaymentId(ctx context.Context, req *notification.GetPaymentStatusesByPaymentIdRequest) (*notification.GetPaymentStatusesByPaymentIdResponse, error) {
	// Проверяем, что запрос вообще передан.
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	// Проверяем корректность payment_id.
	if req.GetPaymentId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "payment_id must be greater than 0")
	}

	items, err := s.repo.GetPaymentStatusesByPaymentID(ctx, req.GetPaymentId())
	// Если чтение из БД не удалось, возвращаем internal ошибку gRPC.
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load payment statuses")
	}

	result := make([]*notification.PaymentStatusRecord, 0, len(items))
	// Преобразуем записи репозитория в protobuf-модели ответа.
	for _, item := range items {
		result = append(result, &notification.PaymentStatusRecord{
			Id:        item.ID,
			PaymentId: item.PaymentID,
			OrderId:   item.OrderID,
			Status:    item.Status,
			Provider:  item.Provider,
			EventTime: item.EventTime.Format(time.RFC3339),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}

	return &notification.GetPaymentStatusesByPaymentIdResponse{Items: result}, nil
}
