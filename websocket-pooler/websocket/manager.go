package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// ClientManager manages connected websocket clients
type ClientManager struct {
	clients sync.Map
	wg      sync.WaitGroup
}

// NewClientManager creates a new client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: sync.Map{},
	}
}

// AddClient adds a client to the manager
func (m *ClientManager) AddClient(clientID string, session *ClientSession) {
	m.clients.Store(clientID, session)
}

// RemoveClient removes a client from the manager
func (m *ClientManager) RemoveClient(clientID string) {
	m.clients.Delete(clientID)
}

// GetClient retrieves a client by ID
func (m *ClientManager) GetClient(clientID string) (*ClientSession, bool) {
	if client, ok := m.clients.Load(clientID); ok {
		return client.(*ClientSession), true
	}
	return nil, false
}

// IncreaseWaitGroup increases the wait group counter
func (m *ClientManager) IncreaseWaitGroup() {
	m.wg.Add(1)
}

// DecreaseWaitGroup decreases the wait group counter
func (m *ClientManager) DecreaseWaitGroup() {
	m.wg.Done()
}

// WaitForCompletion waits for all operations to complete
func (m *ClientManager) WaitForCompletion() {
	m.wg.Wait()
}

// CloseAllConnections sends close messages to all clients and removes them
func (m *ClientManager) CloseAllConnections(reason string) {
	m.clients.Range(func(key, value interface{}) bool {
		clientID := key.(string)
		session := value.(*ClientSession)

		log.Printf("Closing connection for client %s: %s", clientID, reason)
		session.Close(websocket.CloseGoingAway, reason)
		m.RemoveClient(clientID)

		return true
	})
}
