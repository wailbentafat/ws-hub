package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/wailbentafat/ws-hub/auth"
	"github.com/wailbentafat/ws-hub/broker"
)

const (
	BackendRequestsChannel  = "backend-requests"
	BackendResponsesChannel = "backend-responses"
	PresenceEventsChannel = "presence-events"
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
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "Token not provided", http.StatusUnauthorized)
		return
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return auth.JwtSecretKey, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}
	clientID, ok := claims["sub"].(string)
	if !ok || clientID == "" {
		http.Error(w, "Invalid token subject", http.StatusUnauthorized)
		return
	}
	log.Printf("Authentication successful for clientID: %s", clientID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	session := NewClientSession(clientID, conn)
	h.manager.AddClient(clientID, session)


	go func() {
		connectMsg := broker.Message{
			Type:     "user_connected",
			ClientID: clientID,
		}
		if err := h.broker.Publish(context.Background(), PresenceEventsChannel, connectMsg); err != nil {
			log.Printf("Failed to publish connect event for client %s: %v", clientID, err)
		} else {
			log.Printf("Published 'user_connected' event for %s", clientID)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn.SetPongHandler(func(string) error { session.UpdateActivity(); return nil })
	go session.StartPingSender(ctx)
	go session.StartActivityChecker(ctx, func() {
		log.Printf("Connection timeout for client %s", clientID)
		h.manager.RemoveClient(clientID)
		cancel()
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error from client %s: %v", clientID, err)
			break 
		}

		session.UpdateActivity()

		h.manager.IncreaseWaitGroup()
		go func(messageData []byte) {
			defer h.manager.DecreaseWaitGroup()

			ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := h.broker.Publish(ctxTimeout, BackendRequestsChannel, broker.Message{
				ClientID: clientID,
				Data:     string(messageData),
			}); err != nil {
				log.Printf("Failed to publish message for client %s: %v", clientID, err)
			}
		}(msg)
	}

	
	log.Printf("Cleaning up connection for client %s", clientID)

	go func() {
		disconnectMsg := broker.Message{
			Type:     "user_disconnected",
			ClientID: clientID,
		}
		if err := h.broker.Publish(context.Background(), PresenceEventsChannel, disconnectMsg); err != nil {
			log.Printf("Failed to publish disconnect event for client %s: %v", clientID, err)
		} else {
			log.Printf("Published 'user_disconnected' event for %s", clientID)
		}
	}()

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
