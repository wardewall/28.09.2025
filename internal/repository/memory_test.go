package repository

import (
	"context"
	"testing"

	"april/internal/domain"
)

func TestMemoryStore_ProductCRUD(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	p := domain.Product{Name: "A", SKU: "S1", Price: 10, Stock: 5}
	if err := store.Create(ctx, &p); err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == 0 {
		t.Fatalf("no id")
	}

	got, err := store.GetByID(ctx, p.ID)
	if err != nil || got.ID != p.ID {
		t.Fatalf("get: %v", err)
	}

	p.Price = 12
	if err := store.Update(ctx, &p); err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := store.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetByID(ctx, p.ID); err == nil {
		t.Fatalf("expected not found")
	}
}

func TestMemoryTx_TransactionalUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tx := NewMemoryTx(store)
	orders := NewMemoryOrders(store)

	// seed product
	p := domain.Product{Name: "A", SKU: "S1", Price: 10, Stock: 5}
	if err := store.Create(ctx, &p); err != nil {
		t.Fatal(err)
	}

	// emulate atomic create order with stock decrease
	err := tx.WithTransaction(ctx, func(ctx context.Context) error {
		pp, err := store.GetByID(ctx, p.ID)
		if err != nil {
			return err
		}
		if pp.Stock < 3 {
			t.Fatalf("stock precondition")
		}
		pp.Stock -= 3
		if err := store.Update(ctx, pp); err != nil {
			return err
		}
		o := domain.Order{CustomerName: "John", Items: []domain.OrderItem{{ProductID: p.ID, Quantity: 3}}, Status: domain.OrderStatusConfirmed}
		return orders.Create(ctx, &o)
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}

	// check stock after
	pp, _ := store.GetByID(context.Background(), p.ID)
	if pp.Stock != 2 {
		t.Fatalf("stock expected 2, got %v", pp.Stock)
	}
}

func TestList_Filtering(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	add := func(n string, price float64) {
		p := domain.Product{Name: n, SKU: n, Price: price, Stock: 1}
		if err := store.Create(ctx, &p); err != nil {
			t.Fatal(err)
		}
	}
	add("Aspirin", 100)
	add("Paracetamol", 50)
	add("Ibuprofen", 150)

	// name contains
	list, _ := store.List(ctx, ProductFilter{NameSubstring: "in"})
	if len(list) == 0 {
		t.Fatalf("name filter empty")
	}

	// min
	min := 100.0
	list, _ = store.List(ctx, ProductFilter{MinPrice: &min})
	for _, p := range list {
		if p.Price < min {
			t.Fatalf("min filter fail")
		}
	}

	// max
	max := 100.0
	list, _ = store.List(ctx, ProductFilter{MaxPrice: &max})
	for _, p := range list {
		if p.Price > max {
			t.Fatalf("max filter fail")
		}
	}
}
