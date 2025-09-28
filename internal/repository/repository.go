package repository

import (
	"context"
	"errors"
	"strings"

	"april/internal/domain"
)

// ErrNotFound возвращается, когда сущность не найдена
var ErrNotFound = errors.New("not found")

// ProductFilter параметры фильтрации списка товаров
type ProductFilter struct {
	NameSubstring string
	MinPrice      *float64
	MaxPrice      *float64
}

// ProductRepository интерфейс репозитория товаров
type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id int64) (*domain.Product, error)
	Update(ctx context.Context, p *domain.Product) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f ProductFilter) ([]domain.Product, error)
}

// OrderRepository интерфейс репозитория заказов
type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	Update(ctx context.Context, o *domain.Order) error
}

// TxManager абстракция транзакции. Для in-memory — глобальная блокировка записи.
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// helper: case-insensitive contains
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
