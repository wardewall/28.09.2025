package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"april/internal/domain"
	"april/internal/repository"
	"april/internal/service"
)

type Server struct {
	engine   *gin.Engine
	products *service.ProductService
	orders   *service.OrderService
}

func NewServer(products *service.ProductService, orders *service.OrderService) *Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	s := &Server{engine: r, products: products, orders: orders}
	s.registerRoutes()
	return s
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) registerRoutes() {
	// Swagger UI
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := s.engine.Group("/api/v1")
	{
		products := v1.Group("/products")
		products.POST("", s.createProduct)
		products.GET(":id", s.getProduct)
		products.PUT(":id", s.updateProduct)
		products.DELETE(":id", s.deleteProduct)
		products.GET("", s.listProducts)

		orders := v1.Group("/orders")
		orders.POST("", s.createOrder)
		orders.GET(":id", s.getOrder)
		orders.POST(":id/cancel", s.cancelOrder)
		orders.POST(":id/partial-return", s.partialReturn)
	}
}

// Product handlers
type createProductReq struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock int64   `json:"stock"`
}

// @Summary Create product
// @Tags products
// @Accept json
// @Produce json
// @Param input body createProductReq true "Product"
// @Success 201 {object} domain.Product
// @Failure 400 {object} map[string]string
// @Router /products [post]
func (s *Server) createProduct(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	p, err := s.products.Create(c, domain.Product{Name: req.Name, SKU: req.SKU, Price: req.Price, Stock: req.Stock})
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// @Summary Get product by id
// @Tags products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} domain.Product
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id} [get]
func (s *Server) getProduct(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := s.products.GetByID(c, id)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

type updateProductReq struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int64   `json:"stock"`
}

// @Summary Update product
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param input body updateProductReq true "Update"
// @Success 200 {object} domain.Product
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id} [put]
func (s *Server) updateProduct(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	p, err := s.products.Update(c, domain.Product{ID: id, Name: req.Name, Price: req.Price, Stock: req.Stock})
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

// @Summary Delete product
// @Tags products
// @Param id path int true "Product ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id} [delete]
func (s *Server) deleteProduct(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.products.Delete(c, id); err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary List products
// @Tags products
// @Produce json
// @Param q query string false "Name contains"
// @Param min_price query number false "Min price"
// @Param max_price query number false "Max price"
// @Success 200 {array} domain.Product
// @Router /products [get]
func (s *Server) listProducts(c *gin.Context) {
	var f repository.ProductFilter
	if q := c.Query("q"); q != "" {
		f.NameSubstring = q
	}
	if v := c.Query("min_price"); v != "" {
		if x, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinPrice = &x
		}
	}
	if v := c.Query("max_price"); v != "" {
		if x, err := strconv.ParseFloat(v, 64); err == nil {
			f.MaxPrice = &x
		}
	}
	list, err := s.products.List(c, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Order handlers
type createOrderReq struct {
	CustomerName string             `json:"customer_name"`
	Items        []domain.OrderItem `json:"items"`
}

// @Summary Create order
// @Tags orders
// @Accept json
// @Produce json
// @Param input body createOrderReq true "Order"
// @Success 201 {object} domain.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /orders [post]
func (s *Server) createOrder(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	o, err := s.orders.CreateOrder(c, req.CustomerName, req.Items)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, o)
}

// @Summary Get order by id
// @Tags orders
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /orders/{id} [get]
func (s *Server) getOrder(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := s.orders.GetOrder(c, id)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

// @Summary Cancel order
// @Tags orders
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /orders/{id}/cancel [post]
func (s *Server) cancelOrder(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := s.orders.CancelOrder(c, id)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

type partialReturnReq struct {
	Items []domain.OrderItem `json:"items"`
}

// @Summary Partial return
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Param input body partialReturnReq true "Return items"
// @Success 200 {object} domain.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /orders/{id}/partial-return [post]
func (s *Server) partialReturn(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req partialReturnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	o, err := s.orders.PartialReturn(c, id, req.Items)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func mapErrorToStatus(err error) int {
	switch err {
	case service.ErrInvalidInput:
		return http.StatusBadRequest
	case service.ErrNotEnoughStock:
		return http.StatusBadRequest
	case repository.ErrNotFound:
		return http.StatusNotFound
	case service.ErrInvalidState:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
