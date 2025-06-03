package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/wailbentafat/ws-hub/broker"
	"github.com/wailbentafat/ws-hub/websocket"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, wsHandler http.HandlerFunc) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		httpServer: srv,
	}
}

func (s *Server) Start() {
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) Shutdown(ctx context.Context, clientManager *websocket.ClientManager, broker broker.MessageBroker) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	log.Println("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Closing WebSocket connections...")
	clientManager.CloseAllConnections("Server shutting down")
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

	log.Println("Closing message broker...")
	if err := broker.Close(); err != nil {
		log.Printf("Broker closure error: %v", err)
	}

	log.Println("Shutdown complete")
}
