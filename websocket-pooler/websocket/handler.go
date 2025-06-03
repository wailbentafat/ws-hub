package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/wailbentafat/ws-hub/broker"
)

const (
	BackendRequestsChannel  = "backend-requests"
	BackendResponsesChannel = "backend-responses"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	manager *ClientManager
	broker  broker.MessageBroker
}

func NewHandler(manager *ClientManager, broker broker.MessageBroker) *Handler {
	return &Handler{
		manager: manager,
		broker:  broker,
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := uuid.New().String()

	session := NewClientSession(clientID, conn)

	h.manager.AddClient(clientID, session)

	conn.SetPongHandler(func(string) error {
		session.UpdateActivity()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go session.StartPingSender(ctx)
	go session.StartActivityChecker(ctx, func() {
		log.Printf("Connection timeout for client %s", clientID)
		h.manager.RemoveClient(clientID)
		cancel()
	})

	if err := session.SafeWriteJSON(map[string]string{"client_id": clientID}); err != nil {
		log.Printf("Failed to send client ID: %v", err)
		conn.Close()
		h.manager.RemoveClient(clientID)
		return
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error from client %s: %v", clientID, err)
			break
		}

		session.UpdateActivity()

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

	h.manager.RemoveClient(clientID)
}

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
