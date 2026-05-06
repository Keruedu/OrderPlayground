package inventorypb

import (
	"context"

	"google.golang.org/grpc"
)

const InventoryServiceReserveInventoryFullMethodName = "/inventory.v1.InventoryService/ReserveInventory"

type InventoryItem struct {
	SKU      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

type ReserveInventoryRequest struct {
	OrderID string          `json:"order_id"`
	Items   []InventoryItem `json:"items"`
}

type ReserveInventoryResponse struct {
	Success       bool   `json:"success"`
	ReservationID string `json:"reservation_id"`
	Message       string `json:"message"`
}

type InventoryServiceClient interface {
	ReserveInventory(ctx context.Context, in *ReserveInventoryRequest, opts ...grpc.CallOption) (*ReserveInventoryResponse, error)
}

type inventoryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewInventoryServiceClient(cc grpc.ClientConnInterface) InventoryServiceClient {
	return &inventoryServiceClient{cc: cc}
}

func (c *inventoryServiceClient) ReserveInventory(ctx context.Context, in *ReserveInventoryRequest, opts ...grpc.CallOption) (*ReserveInventoryResponse, error) {
	out := new(ReserveInventoryResponse)
	err := c.cc.Invoke(ctx, InventoryServiceReserveInventoryFullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type InventoryServiceServer interface {
	ReserveInventory(context.Context, *ReserveInventoryRequest) (*ReserveInventoryResponse, error)
}

func RegisterInventoryServiceServer(s grpc.ServiceRegistrar, srv InventoryServiceServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "inventory.v1.InventoryService",
		HandlerType: (*InventoryServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ReserveInventory",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					in := new(ReserveInventoryRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(InventoryServiceServer).ReserveInventory(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: InventoryServiceReserveInventoryFullMethodName,
					}
					handler := func(ctx context.Context, req any) (any, error) {
						return srv.(InventoryServiceServer).ReserveInventory(ctx, req.(*ReserveInventoryRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
	}, srv)
}
