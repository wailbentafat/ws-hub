package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/abdelmounim-dev/websocket-pooler/broker"
)

// Constants for channel names
const (
	BackendRequestsChannel  = "backend-requests"
	BackendResponsesChannel = "backend-responses"
)

// Upgrader for websocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler manages websocket connections and message routing
type Handler struct {
	manager *ClientManager
	broker  broker.MessageBroker
}

// NewHandler creates a new websocket handler
func NewHandler(manager *ClientManager, broker broker.MessageBroker) *Handler {
	return &Handler{
		manager: manager,
		broker:  broker,
	}
}

// HandleWebSocket handles incoming websocket connections
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Generate a unique client ID
	clientID := uuid.New().String()

	// Create a new client session
	session := NewClientSession(clientID, conn)

	// Add client to manager
	h.manager.AddClient(clientID, session)

	// Configure pong handler
	conn.SetPongHandler(func(string) error {
		session.UpdateActivity()
		return nil
	})

	// Create a context for this connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start connection monitoring
	go session.StartPingSender(ctx)
	go session.StartActivityChecker(ctx, func() {
		log.Printf("Connection timeout for client %s", clientID)
		h.manager.RemoveClient(clientID)
		cancel()
	})

	// Send client ID to client
	if err := session.SafeWriteJSON(map[string]string{"client_id": clientID}); err != nil {
		log.Printf("Failed to send client ID: %v", err)
		conn.Close()
		h.manager.RemoveClient(clientID)
		return
	}

	// Read messages from client
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error from client %s: %v", clientID, err)
			break
		}

		// Update activity timestamp
		session.UpdateActivity()

		// Forward message to backend
		h.manager.IncreaseWaitGroup()
		go func() {
			defer h.manager.DecreaseWaitGroup()

			ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := h.broker.Publish(ctxTimeout, BackendRequestsChannel, broker.Message{
				ClientID: clientID,
				Data:     string(msg),
			}); err != nil {
				log.Printf("Failed to publish message for client %s: %v", clientID, err)
			}
		}()
	}

	// Clean up when connection ends
	h.manager.RemoveClient(clientID)
}

// ListenForResponses listens for messages from backend and routes them to clients
func (h *Handler) ListenForResponses(ctx context.Context) {
	messageChan, err := h.broker.Subscribe(ctx, BackendResponsesChannel)
	if err != nil {
		log.Fatalf("Failed to subscribe to %s: %v", BackendResponsesChannel, err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messageChan:
			if !ok {
				log.Println("Backend response channel closed")
				return
			}

			clientID := message.ClientID

			if session, ok := h.manager.GetClient(clientID); ok {
				if err := session.SafeWriteJSON(message.Data); err != nil {
					log.Printf("Failed to send message to client %s: %v", clientID, err)
					session.Close(websocket.CloseInternalServerErr, "Failed to send message")
					h.manager.RemoveClient(clientID)
				}
			}
		}
	}
}
