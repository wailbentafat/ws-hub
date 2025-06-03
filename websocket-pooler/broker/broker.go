package broker

import (
	"context"
)

// Message represents a message passed between clients and backend
type Message struct {
	ClientID string      `json:"client_id"`
	Data     interface{} `json:"data"`
}

// MessageBroker defines the interface for message brokers
type MessageBroker interface {
	// Publish sends a message to the specified channel
	Publish(ctx context.Context, channel string, message Message) error
	
	// Subscribe starts listening for messages on the specified channel
	Subscribe(ctx context.Context, channel string) (<-chan Message, error)
	
	// Close cleans up resources
	Close() error
}
