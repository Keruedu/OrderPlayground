package domain

import "testing"

func TestCreateOrderRequestValidate(t *testing.T) {
	req := CreateOrderRequest{
		CustomerName: "Alice",
		Currency:     "USD",
		Items: []CreateOrderItem{
			{SKU: "ABC", Quantity: 1, Price: 9.99},
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error %v", err)
	}
}

func TestCreateOrderRequestValidateFails(t *testing.T) {
	req := CreateOrderRequest{}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}
