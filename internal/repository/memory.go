package repository

import (
	"context"
	"sync"
	"time"

	"april/internal/domain"
)

// MemoryStore объединённое in-memory хранилище и простой генератор ID
type MemoryStore struct {
	mu           sync.RWMutex
	nextProdID   int64
	nextOrderID  int64
	productsByID map[int64]domain.Product
	ordersByID   map[int64]domain.Order
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextProdID:   1,
		nextOrderID:  1,
		productsByID: make(map[int64]domain.Product),
		ordersByID:   make(map[int64]domain.Order),
	}
}

// transaction-aware locking helpers
type txKey struct{}

func isTx(ctx context.Context) bool {
	v := ctx.Value(txKey{})
	if v == nil {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func (m *MemoryStore) rlock(ctx context.Context) {
	if !isTx(ctx) {
		m.mu.RLock()
	}
}
func (m *MemoryStore) runlock(ctx context.Context) {
	if !isTx(ctx) {
		m.mu.RUnlock()
	}
}
func (m *MemoryStore) wlock(ctx context.Context) {
	if !isTx(ctx) {
		m.mu.Lock()
	}
}
func (m *MemoryStore) wunlock(ctx context.Context) {
	if !isTx(ctx) {
		m.mu.Unlock()
	}
}

// Ensure interfaces
var _ ProductRepository = (*MemoryStore)(nil)

// OrderRepository будет реализован отдельным типом MemoryOrders

// ProductRepository implementation
func (m *MemoryStore) Create(ctx context.Context, p *domain.Product) error {
	m.wlock(ctx)
	defer m.wunlock(ctx)
	p.ID = m.nextProdID
	m.nextProdID++
	m.productsByID[p.ID] = *p
	return nil
}

func (m *MemoryStore) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	m.rlock(ctx)
	defer m.runlock(ctx)
	p, ok := m.productsByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	// return copy
	cp := p
	return &cp, nil
}

func (m *MemoryStore) Update(ctx context.Context, p *domain.Product) error {
	m.wlock(ctx)
	defer m.wunlock(ctx)
	if _, ok := m.productsByID[p.ID]; !ok {
		return ErrNotFound
	}
	m.productsByID[p.ID] = *p
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, id int64) error {
	m.wlock(ctx)
	defer m.wunlock(ctx)
	if _, ok := m.productsByID[id]; !ok {
		return ErrNotFound
	}
	delete(m.productsByID, id)
	return nil
}

func (m *MemoryStore) List(ctx context.Context, f ProductFilter) ([]domain.Product, error) {
	m.rlock(ctx)
	defer m.runlock(ctx)
	out := make([]domain.Product, 0)
	for _, p := range m.productsByID {
		if !containsIgnoreCase(p.Name, f.NameSubstring) {
			continue
		}
		if f.MinPrice != nil && p.Price < *f.MinPrice {
			continue
		}
		if f.MaxPrice != nil && p.Price > *f.MaxPrice {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// OrderRepository implementation on wrapper type
type MemoryOrders struct{ store *MemoryStore }

func NewMemoryOrders(store *MemoryStore) *MemoryOrders { return &MemoryOrders{store: store} }

var _ OrderRepository = (*MemoryOrders)(nil)

func (mo *MemoryOrders) Create(ctx context.Context, o *domain.Order) error {
	mo.store.wlock(ctx)
	defer mo.store.wunlock(ctx)
	o.ID = mo.store.nextOrderID
	mo.store.nextOrderID++
	o.CreatedAt = time.Now().UTC()
	o.UpdatedAt = o.CreatedAt
	mo.store.ordersByID[o.ID] = *o
	return nil
}

func (mo *MemoryOrders) GetByID(ctx context.Context, id int64) (*domain.Order, error) {
	mo.store.rlock(ctx)
	defer mo.store.runlock(ctx)
	o, ok := mo.store.ordersByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := o
	return &cp, nil
}

func (mo *MemoryOrders) Update(ctx context.Context, o *domain.Order) error {
	mo.store.wlock(ctx)
	defer mo.store.wunlock(ctx)
	if _, ok := mo.store.ordersByID[o.ID]; !ok {
		return ErrNotFound
	}
	o.UpdatedAt = time.Now().UTC()
	mo.store.ordersByID[o.ID] = *o
	return nil
}

// Tx manager using write lock to emulate transaction boundary
type MemoryTx struct{ store *MemoryStore }

func NewMemoryTx(store *MemoryStore) *MemoryTx { return &MemoryTx{store: store} }

func (tx *MemoryTx) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Для in-memory используем блокировку записи и помечаем контекст, чтобы репозитории пропускали внутренние локи
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()
	ctx = context.WithValue(ctx, txKey{}, true)
	return fn(ctx)
}
