
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailbentafat/ws-hub/backend/broker"
)

const (
    BackendRequestsChannel  = "backend-requests"
    BackendResponsesChannel = "backend-responses"
)

func main() {
    log.Println("Starting Backend Echo Service...")
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    messageBroker, err := broker.NewRedisBroker("localhost:6379")
    if err != nil {
        log.Fatalf("Failed to create Redis broker: %v", err)
    }
    defer messageBroker.Close()
    log.Println("Connected to Redis Broker.")
    requestsChan, err := messageBroker.Subscribe(ctx, BackendRequestsChannel)
    if err != nil {
        log.Fatalf("Failed to subscribe to '%s': %v", BackendRequestsChannel, err)
    }
    log.Printf("Subscribed to '%s' channel.", BackendRequestsChannel)
    log.Println("Waiting for messages...")
    for {
        select {
        case <-ctx.Done():
            log.Println("Context cancelled. Shutting down service.")
            return
        case msg, ok := <-requestsChan:
            if !ok {
                log.Println("Requests channel closed. Shutting down.")
                return
            }
            log.Printf("Received message for client %s: %s", msg.ClientID, msg.Data)
            log.Printf("Echoing message back to client %s", msg.ClientID)
            err := messageBroker.Publish(ctx, BackendResponsesChannel, msg)
            if err != nil {
                log.Printf("ERROR: Failed to publish echo message for client %s: %v", msg.ClientID, err)
            }
        }
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("Shutdown signal received. Cleaning up.")
	cancel()
    
					
}