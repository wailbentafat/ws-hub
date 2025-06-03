package websocket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"
)

const (
	pingInterval        = 30 * time.Second
	activityTimeout     = 60 * time.Second
	writeWait           = 5 * time.Second
	websocketRetryDelay = 200 * time.Millisecond
)

// ClientSession represents a connected websocket client
type ClientSession struct {
	ID           string
	conn         *websocket.Conn
	lastActivity int64 // UnixNano timestamp
	mu           sync.Mutex
}

// NewClientSession creates a new client session
func NewClientSession(id string, conn *websocket.Conn) *ClientSession {
	return &ClientSession{
		ID:           id,
		conn:         conn,
		lastActivity: time.Now().UnixNano(),
	}
}

// SafeWriteJSON writes data to the websocket with retry capability
func (s *ClientSession) SafeWriteJSON(data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	operation := func() error {
		return s.conn.WriteJSON(data)
	}

	backoffStrategy := backoff.WithContext(
		backoff.NewConstantBackOff(websocketRetryDelay),
		context.Background(),
	)

	return backoff.RetryNotify(operation, backoffStrategy, func(err error, d time.Duration) {
		log.Printf("Retrying WebSocket write: %v (next attempt in %s)", err, d)
	})
}

// UpdateActivity updates the last activity timestamp
func (s *ClientSession) UpdateActivity() {
	atomic.StoreInt64(&s.lastActivity, time.Now().UnixNano())
}

// LastActivityTime returns the time of last activity
func (s *ClientSession) LastActivityTime() time.Time {
	return time.Unix(0, atomic.LoadInt64(&s.lastActivity))
}

// StartPingSender begins sending ping messages to keep the connection alive
func (s *ClientSession) StartPingSender(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.conn.WriteControl(
				websocket.PingMessage,
				nil,
				time.Now().Add(writeWait),
			)
		case <-ctx.Done():
			return
		}
	}
}

// StartActivityChecker monitors the connection for timeouts
func (s *ClientSession) StartActivityChecker(ctx context.Context, onTimeout func()) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(s.LastActivityTime()) > activityTimeout {
				s.conn.Close()
				onTimeout()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// Close closes the websocket connection
func (s *ClientSession) Close(code int, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, text),
		time.Now().Add(writeWait),
	)
	if err != nil {
		log.Printf("Error sending close message: %v", err)
		return err
	}

	// Close the connection regardless of whether the close message was sent
	return s.conn.Close()
}
