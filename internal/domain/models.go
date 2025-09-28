package domain

import "time"

// Product представляет товар в аптеке
type Product struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock int64   `json:"stock"`
}

// OrderStatus тип статуса заказа
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "Pending"
	OrderStatusConfirmed OrderStatus = "Confirmed"
	OrderStatusCancelled OrderStatus = "Cancelled"
)

// OrderItem позиция в заказе
type OrderItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}

// Order сущность заказа
type Order struct {
	ID           int64       `json:"id"`
	CustomerName string      `json:"customer_name"`
	Items        []OrderItem `json:"items"`
	Status       OrderStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
