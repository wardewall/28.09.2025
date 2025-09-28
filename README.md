Сервис для управления товарами и заказами.

## Запуск

```bash
go mod tidy
go run ./cmd
# сервер поднимется на :9091
```

## Эндпоинты

- POST /api/v1/products
- GET /api/v1/products/:id
- PUT /api/v1/products/:id
- DELETE /api/v1/products/:id
- GET /api/v1/products?q=строка&min_price=0&max_price=100

- POST /api/v1/orders
- GET /api/v1/orders/:id
- POST /api/v1/orders/:id/cancel
- POST /api/v1/orders/:id/partial-return

## Примеры curl

```bash
# Создать товар
curl -s -X POST http://localhost:9091/api/v1/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"Aspirin","sku":"ASP-100","price":199.9,"stock":50}'

# Получить товар
curl -s http://localhost:9091/api/v1/products/1

# Обновить товар
curl -s -X PUT http://localhost:9091/api/v1/products/1 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Aspirin","price":189.9,"stock":60}'

# Список товаров с фильтрами
curl -s 'http://localhost:9091/api/v1/products?q=asp&min_price=100&max_price=200'

# Создать заказ
curl -s -X POST http://localhost:9091/api/v1/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer_name":"John","items":[{"product_id":1,"quantity":2}]}'

# Получить заказ
curl -s http://localhost:9091/api/v1/orders/1

# Отменить заказ
curl -s -X POST http://localhost:9091/api/v1/orders/1/cancel

# Частичный возврат
curl -s -X POST http://localhost:9091/api/v1/orders/1/partial-return \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"product_id":1,"quantity":1}]}'
```

## Тесты

```bash
go test ./... -v
```

## Архитектура

- internal/domain — модели и статусы
- internal/repository — интерфейсы и in-memory реализация с TxManager
- internal/service — бизнес-логика продуктов и заказов
- internal/http — HTTP-слой на Gin
- cmd — точка входа


