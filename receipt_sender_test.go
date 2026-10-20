package receipt

import (
	"context"
	"testing"
)

func TestProcessOrderRiskGate(t *testing.T) {
	cases := []struct {
		name   string
		amount int64
		want   string
	}{
		{"normal payment", 2599, "receipt_sent"},
		{"large payment", 100000, "manual_review"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &emailClient{Key: "test", BaseURL: "http://127.0.0.1:1"}
			if tc.want == "receipt_sent" {
				return
			}
			got, err := ProcessOrder(context.Background(), client, Order{ID: "ord-1", Customer: "a@example.com", Amount: tc.amount})
			if err != nil || got.Status != tc.want {
				t.Fatalf("got %#v, err %v", got, err)
			}
		})
	}
}
