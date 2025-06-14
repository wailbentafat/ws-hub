package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ClientManager struct {
	clients sync.Map
	wg      sync.WaitGroup
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: sync.Map{},
	}
}

func (m *ClientManager) AddClient(clientID string, session *ClientSession) {
	m.clients.Store(clientID, session)
}

func (m *ClientManager) RemoveClient(clientID string) {
	m.clients.Delete(clientID)
}

func (m *ClientManager) GetClient(clientID string) (*ClientSession, bool) {
	if client, ok := m.clients.Load(clientID); ok {
		return client.(*ClientSession), true
	}
	return nil, false
}

func (m *ClientManager) IncreaseWaitGroup() {
	m.wg.Add(1)
}

func (m *ClientManager) DecreaseWaitGroup() {
	m.wg.Done()
}

func (m *ClientManager) WaitForCompletion() {
	m.wg.Wait()
}

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
