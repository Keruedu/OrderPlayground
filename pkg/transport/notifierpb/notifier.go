package notifierpb

import (
	"context"

	"google.golang.org/grpc"
)

const NotifierServiceSendOrderNotificationFullMethodName = "/notifier.v1.NotifierService/SendOrderNotification"

type SendOrderNotificationRequest struct {
	OrderID   string `json:"order_id"`
	EventType string `json:"event_type"`
	Recipient string `json:"recipient"`
}

type SendOrderNotificationResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
}

type NotifierServiceClient interface {
	SendOrderNotification(ctx context.Context, in *SendOrderNotificationRequest, opts ...grpc.CallOption) (*SendOrderNotificationResponse, error)
}

type notifierServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNotifierServiceClient(cc grpc.ClientConnInterface) NotifierServiceClient {
	return &notifierServiceClient{cc: cc}
}

func (c *notifierServiceClient) SendOrderNotification(ctx context.Context, in *SendOrderNotificationRequest, opts ...grpc.CallOption) (*SendOrderNotificationResponse, error) {
	out := new(SendOrderNotificationResponse)
	err := c.cc.Invoke(ctx, NotifierServiceSendOrderNotificationFullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type NotifierServiceServer interface {
	SendOrderNotification(context.Context, *SendOrderNotificationRequest) (*SendOrderNotificationResponse, error)
}

func RegisterNotifierServiceServer(s grpc.ServiceRegistrar, srv NotifierServiceServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "notifier.v1.NotifierService",
		HandlerType: (*NotifierServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "SendOrderNotification",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					in := new(SendOrderNotificationRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(NotifierServiceServer).SendOrderNotification(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: NotifierServiceSendOrderNotificationFullMethodName,
					}
					handler := func(ctx context.Context, req any) (any, error) {
						return srv.(NotifierServiceServer).SendOrderNotification(ctx, req.(*SendOrderNotificationRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
	}, srv)
}
