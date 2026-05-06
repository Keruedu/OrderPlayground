package grpcclients

import (
	"context"
	"time"

	"order-playground/apps/gateway-api/internal/domain"
	"order-playground/pkg/transport/grpcjson"
	"order-playground/pkg/transport/inventorypb"
	"order-playground/pkg/transport/notifierpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InventoryClient struct {
	client inventorypb.InventoryServiceClient
}

type NotifierClient struct {
	client notifierpb.NotifierServiceClient
}

func NewInventoryClient(address string) (*InventoryClient, *grpc.ClientConn, error) {
	grpcjson.Register()
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})),
	)
	if err != nil {
		return nil, nil, err
	}
	return &InventoryClient{client: inventorypb.NewInventoryServiceClient(conn)}, conn, nil
}

func NewNotifierClient(address string) (*NotifierClient, *grpc.ClientConn, error) {
	grpcjson.Register()
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})),
	)
	if err != nil {
		return nil, nil, err
	}
	return &NotifierClient{client: notifierpb.NewNotifierServiceClient(conn)}, conn, nil
}

func (c *InventoryClient) ReserveInventory(ctx context.Context, orderID string, items []domain.CreateOrderItem) (*inventorypb.ReserveInventoryResponse, error) {
	req := &inventorypb.ReserveInventoryRequest{OrderID: orderID}
	for _, item := range items {
		req.Items = append(req.Items, inventorypb.InventoryItem{
			SKU:      item.SKU,
			Quantity: item.Quantity,
		})
	}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.client.ReserveInventory(callCtx, req)
}

func (c *NotifierClient) SendOrderNotification(ctx context.Context, orderID, eventType, recipient string) (*notifierpb.SendOrderNotificationResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.client.SendOrderNotification(callCtx, &notifierpb.SendOrderNotificationRequest{
		OrderID:   orderID,
		EventType: eventType,
		Recipient: recipient,
	})
}
