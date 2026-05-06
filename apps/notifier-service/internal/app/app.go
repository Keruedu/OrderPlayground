package app

import (
	"context"
	"log"
	"net"
	"os"

	"order-playground/pkg/transport/grpcjson"
	"order-playground/pkg/transport/notifierpb"

	"google.golang.org/grpc"
)

type server struct {
	notifierpb.NotifierServiceServer
}

func Run(ctx context.Context) error {
	grpcjson.Register()
	port := getenvDefault("GRPC_PORT", "9092")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer lis.Close()

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))
	notifierpb.RegisterNotifierServiceServer(grpcServer, &server{})

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
}

func (s *server) SendOrderNotification(ctx context.Context, req *notifierpb.SendOrderNotificationRequest) (*notifierpb.SendOrderNotificationResponse, error) {
	log.Printf("notification accepted order_id=%s recipient=%s event=%s", req.OrderID, req.Recipient, req.EventType)
	return &notifierpb.SendOrderNotificationResponse{
		Success:   true,
		MessageID: "msg-" + req.OrderID,
	}, nil
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
