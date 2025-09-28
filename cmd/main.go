package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "april/internal/http"
	"april/internal/repository"
	"april/internal/service"

	_ "april/docs"
)

func main() {
	store := repository.NewMemoryStore()
	ordersRepo := repository.NewMemoryOrders(store)
	tx := repository.NewMemoryTx(store)

	productsSvc := service.NewProductService(store)
	ordersSvc := service.NewOrderService(store, ordersRepo, tx)

	srv := httpapi.NewServer(productsSvc, ordersSvc)

	httpServer := &http.Server{
		Addr:    ":9091",
		Handler: srv.Engine(),
	}

	go func() {
		log.Printf("HTTP server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
