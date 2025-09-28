package service

import (
	"context"
	"testing"

	"april/internal/domain"
	"april/internal/repository"
)

func setupPS(t *testing.T) *ProductService {
	t.Helper()
	store := repository.NewMemoryStore()
	return NewProductService(store)
}

func TestProduct_Create_Valid(t *testing.T) {
	ctx := context.Background()
	ps := setupPS(t)
	p, err := ps.Create(ctx, domain.Product{Name: "Aspirin", SKU: "ASP-1", Price: 100, Stock: 10})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID == 0 {
		t.Fatalf("expected id assigned")
	}
}

func TestProduct_Create_Invalid(t *testing.T) {
	ctx := context.Background()
	ps := setupPS(t)
	if _, err := ps.Create(ctx, domain.Product{Name: "", SKU: "S", Price: 1, Stock: 1}); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := ps.Create(ctx, domain.Product{Name: "N", SKU: "", Price: 1, Stock: 1}); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := ps.Create(ctx, domain.Product{Name: "N", SKU: "S", Price: -1, Stock: 1}); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := ps.Create(ctx, domain.Product{Name: "N", SKU: "S", Price: 1, Stock: -1}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestProduct_Update_Get_Delete(t *testing.T) {
	ctx := context.Background()
	ps := setupPS(t)
	p, _ := ps.Create(ctx, domain.Product{Name: "A", SKU: "S1", Price: 10, Stock: 5})

	// get
	got, err := ps.GetByID(ctx, p.ID)
	if err != nil || got.ID != p.ID {
		t.Fatalf("get failed: %v", err)
	}

	// update
	p.Name = "A+"
	p.Price = 12
	p.Stock = 7
	up, err := ps.Update(ctx, *p)
	if err != nil {
		t.Fatalf("update err: %v", err)
	}
	if up.Name != "A+" || up.Price != 12 || up.Stock != 7 {
		t.Fatalf("not updated")
	}

	// delete
	if err := ps.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if _, err := ps.GetByID(ctx, p.ID); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestProduct_List_Filtering(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()
	ps := NewProductService(store)
	must := func(p *domain.Product, err error) *domain.Product {
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	_ = must(ps.Create(ctx, domain.Product{Name: "Aspirin", SKU: "S1", Price: 100, Stock: 5}))
	_ = must(ps.Create(ctx, domain.Product{Name: "Paracetamol", SKU: "S2", Price: 50, Stock: 5}))
	_ = must(ps.Create(ctx, domain.Product{Name: "Ibuprofen", SKU: "S3", Price: 150, Stock: 5}))

	// substring
	list, err := ps.List(ctx, repository.ProductFilter{NameSubstring: "in"})
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected some items")
	}

	// min price
	min := 100.0
	list, err = ps.List(ctx, repository.ProductFilter{MinPrice: &min})
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	for _, p := range list {
		if p.Price < min {
			t.Fatalf("price filter failed")
		}
	}

	// max price
	max := 100.0
	list, err = ps.List(ctx, repository.ProductFilter{MaxPrice: &max})
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	for _, p := range list {
		if p.Price > max {
			t.Fatalf("price filter failed")
		}
	}
}
