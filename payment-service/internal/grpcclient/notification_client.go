package grpcclient

import (
	"context"
	"time"

	notificationpb "payment-service/gen/notification"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NotificationClient отправляет статусы платежей в notification-service.
type NotificationClient struct {
	conn   *grpc.ClientConn
	client notificationpb.NotificationServiceClient
}

// NewNotificationClient создает gRPC-подключение к notification-service.
func NewNotificationClient(addr string) (*NotificationClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &NotificationClient{
		conn:   conn,
		client: notificationpb.NewNotificationServiceClient(conn),
	}, nil
}

// Close закрывает gRPC-подключение клиента.
func (c *NotificationClient) Close() error {
	return c.conn.Close()
}

// SendPaymentStatus отправляет в notification-service один статус платежа.
func (c *NotificationClient) SendPaymentStatus(ctx context.Context, paymentID int64, orderID int64, status string, provider string, eventTime time.Time) error {
	_, err := c.client.SendPaymentStatus(ctx, &notificationpb.SendPaymentStatusRequest{
		PaymentId: paymentID,
		OrderId:   orderID,
		Status:    status,
		Provider:  provider,
		EventTime: eventTime.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	return nil
}
