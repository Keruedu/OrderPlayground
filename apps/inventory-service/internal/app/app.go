package app

import (
	"context"
	"fmt"
	"net"
	"os"

	"order-playground/pkg/transport/grpcjson"
	"order-playground/pkg/transport/inventorypb"

	"google.golang.org/grpc"
)

type server struct {
	inventorypb.InventoryServiceServer
}

func Run(ctx context.Context) error {
	grpcjson.Register()
	port := getenvDefault("GRPC_PORT", "9091")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer lis.Close()

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))
	inventorypb.RegisterInventoryServiceServer(grpcServer, &server{})

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
}

func (s *server) ReserveInventory(ctx context.Context, req *inventorypb.ReserveInventoryRequest) (*inventorypb.ReserveInventoryResponse, error) {
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return &inventorypb.ReserveInventoryResponse{
				Success: false,
				Message: "quantity must be positive",
			}, nil
		}
		if item.Quantity > 5 {
			return &inventorypb.ReserveInventoryResponse{
				Success: false,
				Message: fmt.Sprintf("inventory rejected for sku %s", item.SKU),
			}, nil
		}
	}
	return &inventorypb.ReserveInventoryResponse{
		Success:       true,
		ReservationID: "resv-" + req.OrderID,
		Message:       "inventory reserved",
	}, nil
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
