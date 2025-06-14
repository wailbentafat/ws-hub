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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messageBroker, err := broker.NewRedisBroker("redis:6379")
	if err != nil {
		log.Fatalf("Failed to create Redis broker: %v", err)
	}
	defer messageBroker.Close()

	clientManager := websocket.NewClientManager()

	handler := websocket.NewHandler(clientManager, messageBroker)

	srv := server.NewServer(":8080", handler.HandleWebSocket)

	go handler.ListenForResponses(ctx)

	go srv.Start()
	log.Println("WebSocket pooler started on :8080")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutdown signal received")

	srv.Shutdown(ctx, clientManager, messageBroker)
}
