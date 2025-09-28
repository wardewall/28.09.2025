package service

import (
	"context"
	"testing"

	"april/internal/domain"
	"april/internal/repository"
)

func setup(t *testing.T) (*ProductService, *OrderService) {
	t.Helper()
	store := repository.NewMemoryStore()
	ordersRepo := repository.NewMemoryOrders(store)
	tx := repository.NewMemoryTx(store)
	ps := NewProductService(store)
	os := NewOrderService(store, ordersRepo, tx)
	return ps, os
}

func TestCreateOrderAndCancel(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	// create products
	p1, err := ps.Create(ctx, domain.Product{Name: "A", SKU: "SKU1", Price: 10, Stock: 5})
	if err != nil {
		t.Fatalf("create p1: %v", err)
	}
	p2, err := ps.Create(ctx, domain.Product{Name: "B", SKU: "SKU2", Price: 20, Stock: 2})
	if err != nil {
		t.Fatalf("create p2: %v", err)
	}

	// create order
	o, err := os.CreateOrder(ctx, "John", []domain.OrderItem{{ProductID: p1.ID, Quantity: 3}, {ProductID: p2.ID, Quantity: 2}})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if o.Status != domain.OrderStatusConfirmed {
		t.Fatalf("expected confirmed")
	}

	// stocks decreased
	p1After, _ := ps.GetByID(ctx, p1.ID)
	p2After, _ := ps.GetByID(ctx, p2.ID)
	if p1After.Stock != 2 || p2After.Stock != 0 {
		t.Fatalf("stock not decreased: %v %v", p1After.Stock, p2After.Stock)
	}

	// cancel
	o2, err := os.CancelOrder(ctx, o.ID)
	if err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if o2.Status != domain.OrderStatusCancelled {
		t.Fatalf("expected cancelled")
	}

	// stocks restored
	p1R, _ := ps.GetByID(ctx, p1.ID)
	p2R, _ := ps.GetByID(ctx, p2.ID)
	if p1R.Stock != 5 || p2R.Stock != 2 {
		t.Fatalf("stock not restored: %v %v", p1R.Stock, p2R.Stock)
	}
}

func TestCreateOrder_NotEnoughStock(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	p1, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "SKU1", Price: 10, Stock: 1})
	_, err := os.CreateOrder(ctx, "John", []domain.OrderItem{{ProductID: p1.ID, Quantity: 2}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrNotEnoughStock {
		t.Fatalf("expected not enough stock, got %v", err)
	}
}

func TestPartialReturn(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	p1, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "SKU1", Price: 10, Stock: 10})
	p2, _ := ps.Create(ctx, domain.Product{Name: "B", SKU: "SKU2", Price: 15, Stock: 5})
	o, err := os.CreateOrder(ctx, "Jane", []domain.OrderItem{{ProductID: p1.ID, Quantity: 4}, {ProductID: p2.ID, Quantity: 3}})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// return 2 of product1 and 1 of product2
	o2, err := os.PartialReturn(ctx, o.ID, []domain.OrderItem{{ProductID: p1.ID, Quantity: 2}, {ProductID: p2.ID, Quantity: 1}})
	if err != nil {
		t.Fatalf("partial return: %v", err)
	}

	// stocks after return
	p1a, _ := ps.GetByID(ctx, p1.ID)
	p2a, _ := ps.GetByID(ctx, p2.ID)
	if p1a.Stock != 8 {
		t.Fatalf("p1 stock expected 8, got %v", p1a.Stock)
	}
	if p2a.Stock != 3 {
		t.Fatalf("p2 stock expected 3, got %v", p2a.Stock)
	}

	// order items updated: p1 2 left, p2 2 left
	// verify by summing
	sum1, sum2 := int64(0), int64(0)
	for _, it := range o2.Items {
		if it.ProductID == p1.ID {
			sum1 += it.Quantity
		}
		if it.ProductID == p2.ID {
			sum2 += it.Quantity
		}
	}
	if sum1 != 2 || sum2 != 2 {
		t.Fatalf("order items not updated: %v %v", sum1, sum2)
	}
}

func TestCancelOrder_InvalidState(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	p1, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "SKU1", Price: 10, Stock: 10})
	o, err := os.CreateOrder(ctx, "Jane", []domain.OrderItem{{ProductID: p1.ID, Quantity: 2}})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := os.CancelOrder(ctx, o.ID); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if _, err := os.CancelOrder(ctx, o.ID); err == nil {
		t.Fatalf("expected invalid state on second cancel")
	}
}

func TestPartialReturn_Exceed(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	p1, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "SKU1", Price: 10, Stock: 10})
	o, err := os.CreateOrder(ctx, "Jane", []domain.OrderItem{{ProductID: p1.ID, Quantity: 2}})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := os.PartialReturn(ctx, o.ID, []domain.OrderItem{{ProductID: p1.ID, Quantity: 3}}); err == nil {
		t.Fatalf("expected validation error on exceed return")
	}
}

func TestCreateOrder_InvalidInput(t *testing.T) {
	ctx := context.Background()
	ps, os := setup(t)
	p1, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "S1", Price: 10, Stock: 5})
	if _, err := os.CreateOrder(ctx, "", []domain.OrderItem{{ProductID: p1.ID, Quantity: 1}}); err == nil {
		t.Fatalf("expected invalid input for empty customer")
	}
	if _, err := os.CreateOrder(ctx, "John", []domain.OrderItem{{ProductID: p1.ID, Quantity: 0}}); err == nil {
		t.Fatalf("expected invalid quantity")
	}
}
