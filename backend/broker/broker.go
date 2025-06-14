package broker

import (
	"context"
)

type Message struct {
	Type     string      `json:"type,omitempty"` 
	ClientID string      `json:"client_id"`
	Data     interface{} `json:"data"`
}

type MessageBroker interface {
	Publish(ctx context.Context, channel string, message Message) error
	
	Subscribe(ctx context.Context, channel string) (<-chan Message, error)
	
	Close() error
}
