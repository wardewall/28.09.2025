package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"april/internal/repository"
	"april/internal/service"
)

func setupServer(t *testing.T) *Server {
	t.Helper()
	store := repository.NewMemoryStore()
	ordersRepo := repository.NewMemoryOrders(store)
	tx := repository.NewMemoryTx(store)
	productsSvc := service.NewProductService(store)
	ordersSvc := service.NewOrderService(store, ordersRepo, tx)
	return NewServer(productsSvc, ordersSvc)
}

func doJSON(t *testing.T, s *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Engine().ServeHTTP(w, req)
	return w
}

func TestProductFlow(t *testing.T) {
	s := setupServer(t)
	// create
	w := doJSON(t, s, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Aspirin", "sku": "S1", "price": 10, "stock": 5,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create code %v", w.Code)
	}
	// get
	w = doJSON(t, s, http.MethodGet, "/api/v1/products/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get code %v", w.Code)
	}
	// update
	w = doJSON(t, s, http.MethodPut, "/api/v1/products/1", map[string]any{
		"name": "A+", "price": 12, "stock": 7,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update code %v", w.Code)
	}
	// list
	w = doJSON(t, s, http.MethodGet, "/api/v1/products?q=asp", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list code %v", w.Code)
	}
	// delete
	w = doJSON(t, s, http.MethodDelete, "/api/v1/products/1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete code %v", w.Code)
	}
}

func TestOrderFlow(t *testing.T) {
	s := setupServer(t)
	// prepare product
	w := doJSON(t, s, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Aspirin", "sku": "S1", "price": 10, "stock": 5,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create product %v", w.Code)
	}

	// create order
	w = doJSON(t, s, http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_name": "John",
		"items":         []map[string]any{{"product_id": 1, "quantity": 3}},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create order %v", w.Code)
	}

	// get order
	w = doJSON(t, s, http.MethodGet, "/api/v1/orders/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get order %v", w.Code)
	}

	// partial return 1
	w = doJSON(t, s, http.MethodPost, "/api/v1/orders/1/partial-return", map[string]any{
		"items": []map[string]any{{"product_id": 1, "quantity": 1}},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("partial return %v", w.Code)
	}

	// cancel
	w = doJSON(t, s, http.MethodPost, "/api/v1/orders/1/cancel", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("cancel %v", w.Code)
	}
}

func TestHTTP_BadRequests(t *testing.T) {
	s := setupServer(t)
	// invalid product body
	w := doJSON(t, s, http.MethodPost, "/api/v1/products", map[string]any{"name": ""})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", w.Code)
	}

	// invalid id
	w = doJSON(t, s, http.MethodGet, "/api/v1/products/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", w.Code)
	}
}

func TestHTTP_NotFound_Conflict(t *testing.T) {
	s := setupServer(t)
	// not found
	w := doJSON(t, s, http.MethodGet, "/api/v1/products/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", w.Code)
	}

	// create product and order, then cancel twice -> conflict
	_ = doJSON(t, s, http.MethodPost, "/api/v1/products", map[string]any{"name": "A", "sku": "S1", "price": 1, "stock": 1})
	_ = doJSON(t, s, http.MethodPost, "/api/v1/orders", map[string]any{"customer_name": "C", "items": []map[string]any{{"product_id": 1, "quantity": 1}}})
	_ = doJSON(t, s, http.MethodPost, "/api/v1/orders/1/cancel", nil)
	w = doJSON(t, s, http.MethodPost, "/api/v1/orders/1/cancel", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %v", w.Code)
	}
}
