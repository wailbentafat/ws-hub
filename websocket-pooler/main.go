package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailbentafat/ws-hub/broker"
	"github.com/wailbentafat/ws-hub/server"
	"github.com/wailbentafat/ws-hub/websocket"
)

func main() {
	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create message broker
	messageBroker, err := broker.NewRedisBroker("localhost:6379")
	if err != nil {
		log.Fatalf("Failed to create Redis broker: %v", err)
	}
	defer messageBroker.Close()

	// Create client manager
	clientManager := websocket.NewClientManager()

	// Initialize handlers
	handler := websocket.NewHandler(clientManager, messageBroker)

	// Create and configure server
	srv := server.NewServer(":8080", handler.HandleWebSocket)

	// Start message listener
	go handler.ListenForResponses(ctx)

	// Start server
	go srv.Start()
	log.Println("WebSocket pooler started on :8080")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutdown signal received")

	// Graceful shutdown
	srv.Shutdown(ctx, clientManager, messageBroker)
}
