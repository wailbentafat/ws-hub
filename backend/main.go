// backend/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/wailbentafat/ws-hub/backend/broker"
)

const (
    BackendRequestsChannel  = "backend-requests"
    BackendResponsesChannel = "backend-responses"
)

func main() {
    log.Println("Starting Backend Service...")
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // --- Initialize Components ---
    rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
    log.Println("Connected to Redis.")
    
    messageBroker, err := broker.NewRedisBrokerFromClient(rdb)
    if err != nil {
        log.Fatalf("Failed to create broker: %v", err)
    }
    defer messageBroker.Close()

    store := NewStore(rdb)

    log.Println("Starting listeners...")
    go ListenForPresenceEvents(ctx, messageBroker, store)
    go ListenForRequests(ctx, messageBroker, store)
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutdown signal received. Cleaning up.")
}