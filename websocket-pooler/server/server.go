package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/abdelmounim-dev/websocket-pooler/broker"
	"github.com/abdelmounim-dev/websocket-pooler/websocket"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new HTTP server
func NewServer(addr string, wsHandler http.HandlerFunc) *Server {
	// Register handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)

	// Create server
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		httpServer: srv,
	}
}

// Start starts the HTTP server
func (s *Server) Start() {
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// Shutdown gracefully shuts down the server and cleans up resources
func (s *Server) Shutdown(ctx context.Context, clientManager *websocket.ClientManager, broker broker.MessageBroker) {
	// Create shutdown context with 15s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Step 1: Stop accepting new connections
	log.Println("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Step 2: Close all active WebSocket connections
	log.Println("Closing WebSocket connections...")
	clientManager.CloseAllConnections("Server shutting down")

	// Step 3: Wait for in-flight messages to process
	log.Println("Waiting for pending operations...")
	done := make(chan struct{})
	go func() {
		clientManager.WaitForCompletion()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All operations completed")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
	}

	// Step 4: Close message broker
	log.Println("Closing message broker...")
	if err := broker.Close(); err != nil {
		log.Printf("Broker closure error: %v", err)
	}

	log.Println("Shutdown complete")
}
