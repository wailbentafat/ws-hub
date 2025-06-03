package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-redis/redis/v8"
)

const (
	maxRetries     = 3
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

// RedisBroker implements MessageBroker using Redis pub/sub
type RedisBroker struct {
	client *redis.Client
}

// NewRedisBroker creates a new Redis message broker
func NewRedisBroker(addr string) (*RedisBroker, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisBroker{client: client}, nil
}

// MarshalBinary implements encoding.BinaryMarshaler interface
func (m Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface
func (m *Message) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

// Publish sends a message to the specified channel with retry capability
func (b *RedisBroker) Publish(ctx context.Context, channel string, message Message) error {
	operation := func() error {
		return b.client.Publish(ctx, channel, message).Err()
	}

	backoffStrategy := backoff.WithContext(
		backoff.WithMaxRetries(
			backoff.NewExponentialBackOff(
				backoff.WithInitialInterval(initialBackoff),
				backoff.WithMaxInterval(maxBackoff),
			),
			maxRetries,
		),
		ctx,
	)

	return backoff.RetryNotify(operation, backoffStrategy, func(err error, d time.Duration) {
		log.Printf("Retrying Redis publish for %s: %v (next attempt in %s)", message.ClientID, err, d)
	})
}

// Subscribe starts listening for messages on the specified channel
func (b *RedisBroker) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	pubsub := b.client.Subscribe(ctx, channel)

	// Test subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return nil, fmt.Errorf("failed to subscribe to %s: %w", channel, err)
	}

	messages := make(chan Message)

	go func() {
		defer pubsub.Close()
		defer close(messages)

		msgChan := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}

				var message Message
				if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
					log.Printf("Message decode error: %v", err)
					continue
				}

				messages <- message
			}
		}
	}()

	return messages, nil
}

// Close cleans up resources
func (b *RedisBroker) Close() error {
	return b.client.Close()
}
