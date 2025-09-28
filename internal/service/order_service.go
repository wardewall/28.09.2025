package service

import (
	"context"
	"errors"

	"april/internal/domain"
	"april/internal/repository"
)

// OrderService реализует логику заказов: создание, отмена, частичный возврат
type OrderService struct {
	products repository.ProductRepository
	orders   repository.OrderRepository
	tx       repository.TxManager
}

func NewOrderService(products repository.ProductRepository, orders repository.OrderRepository, tx repository.TxManager) *OrderService {
	return &OrderService{products: products, orders: orders, tx: tx}
}

var (
	ErrNotEnoughStock = errors.New("not enough stock")
	ErrInvalidState   = errors.New("invalid state")
)

// CreateOrder проверяет наличие товара и атомарно списывает запас
func (s *OrderService) CreateOrder(ctx context.Context, customer string, items []domain.OrderItem) (*domain.Order, error) {
	if customer == "" || len(items) == 0 {
		return nil, ErrInvalidInput
	}
	// validate items
	for _, it := range items {
		if it.ProductID <= 0 || it.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
	}

	var created *domain.Order
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		// load and check stock
		// accumulate updates to avoid partial state
		productCopies := make(map[int64]*domain.Product)
		for _, it := range items {
			p, err := s.products.GetByID(ctx, it.ProductID)
			if err != nil {
				return err
			}
			if p.Stock < it.Quantity {
				return ErrNotEnoughStock
			}
			// reserve
			p.Stock -= it.Quantity
			productCopies[p.ID] = p
		}
		// persist product stock updates
		for _, p := range productCopies {
			if err := s.products.Update(ctx, p); err != nil {
				return err
			}
		}

		// create order
		o := domain.Order{
			CustomerName: customer,
			Items:        items,
			Status:       domain.OrderStatusConfirmed,
		}
		if err := s.orders.Create(ctx, &o); err != nil {
			return err
		}
		created = &o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// GetOrder возвращает заказ по id
func (s *OrderService) GetOrder(ctx context.Context, id int64) (*domain.Order, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	return s.orders.GetByID(ctx, id)
}

// CancelOrder если Confirmed — возвращаем товары на склад и ставим Cancelled
func (s *OrderService) CancelOrder(ctx context.Context, id int64) (*domain.Order, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	var updated *domain.Order
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		o, err := s.orders.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if o.Status != domain.OrderStatusConfirmed {
			return ErrInvalidState
		}
		// return stock
		for _, it := range o.Items {
			p, err := s.products.GetByID(ctx, it.ProductID)
			if err != nil {
				return err
			}
			p.Stock += it.Quantity
			if err := s.products.Update(ctx, p); err != nil {
				return err
			}
		}
		o.Status = domain.OrderStatusCancelled
		if err := s.orders.Update(ctx, o); err != nil {
			return err
		}
		updated = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// PartialReturn уменьшает количество в заказе и возвращает часть на склад
func (s *OrderService) PartialReturn(ctx context.Context, id int64, returns []domain.OrderItem) (*domain.Order, error) {
	if id <= 0 || len(returns) == 0 {
		return nil, ErrInvalidInput
	}
	// validate returns
	for _, r := range returns {
		if r.ProductID <= 0 || r.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
	}

	var updated *domain.Order
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		o, err := s.orders.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if o.Status != domain.OrderStatusConfirmed {
			return ErrInvalidState
		}
		// map current quantities
		qtyByProduct := make(map[int64]int64)
		for _, it := range o.Items {
			qtyByProduct[it.ProductID] += it.Quantity
		}
		// validate not exceeding
		for _, r := range returns {
			if qtyByProduct[r.ProductID] < r.Quantity {
				return ErrInvalidInput
			}
		}
		// apply returns to order items and restore stock
		newItems := make([]domain.OrderItem, 0, len(o.Items))
		consumed := make(map[int64]int64)
		for _, it := range o.Items {
			ret := consumed[it.ProductID]
			remainingToReturn := qtyByProduct[it.ProductID]
			_ = remainingToReturn
			// how much to return for this product overall
			totalReturn := int64(0)
			for _, r := range returns {
				if r.ProductID == it.ProductID {
					totalReturn += r.Quantity
				}
			}
			if ret >= totalReturn {
				// already returned enough in previous items of same product
				newItems = append(newItems, it)
				continue
			}
			// available to return from this item
			canReturn := it.Quantity
			needReturn := totalReturn - ret
			if needReturn < canReturn {
				// partially reduce
				it.Quantity -= needReturn
				consumed[it.ProductID] += needReturn
				newItems = append(newItems, it)
				// restore stock
				p, err := s.products.GetByID(ctx, it.ProductID)
				if err != nil {
					return err
				}
				p.Stock += needReturn
				if err := s.products.Update(ctx, p); err != nil {
					return err
				}
			} else if needReturn == canReturn {
				// drop this item fully
				consumed[it.ProductID] += needReturn
				// restore stock
				p, err := s.products.GetByID(ctx, it.ProductID)
				if err != nil {
					return err
				}
				p.Stock += needReturn
				if err := s.products.Update(ctx, p); err != nil {
					return err
				}
			} else { // needReturn > canReturn
				// consume whole item and continue
				consumed[it.ProductID] += canReturn
				p, err := s.products.GetByID(ctx, it.ProductID)
				if err != nil {
					return err
				}
				p.Stock += canReturn
				if err := s.products.Update(ctx, p); err != nil {
					return err
				}
				// item removed
			}
		}
		o.Items = newItems
		if err := s.orders.Update(ctx, o); err != nil {
			return err
		}
		updated = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
