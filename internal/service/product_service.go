package service

import (
	"context"
	"errors"

	"april/internal/domain"
	"april/internal/repository"
)

// ProductService инкапсулирует бизнес-логику вокруг товаров
type ProductService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

var ErrInvalidInput = errors.New("invalid input")

func (s *ProductService) Create(ctx context.Context, p domain.Product) (*domain.Product, error) {
	if p.Name == "" || p.SKU == "" || p.Price < 0 || p.Stock < 0 {
		return nil, ErrInvalidInput
	}
	cp := p
	if err := s.repo.Create(ctx, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

func (s *ProductService) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) Update(ctx context.Context, p domain.Product) (*domain.Product, error) {
	if p.ID <= 0 || p.Name == "" || p.Price < 0 || p.Stock < 0 {
		return nil, ErrInvalidInput
	}
	cp := p
	if err := s.repo.Update(ctx, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

func (s *ProductService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) List(ctx context.Context, f repository.ProductFilter) ([]domain.Product, error) {
	return s.repo.List(ctx, f)
}
