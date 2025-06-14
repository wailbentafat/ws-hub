// backend/events.go

package main

import (
	"context"
	"log"

	"github.com/wailbentafat/ws-hub/backend/broker"
)

const PresenceEventsChannel = "presence-events"

func ListenForPresenceEvents(ctx context.Context, messageBroker broker.MessageBroker, store *Store) {
	eventsChan, err := messageBroker.Subscribe(ctx, PresenceEventsChannel)
	if err != nil {
		log.Fatalf("Failed to subscribe to presence events: %v", err)
	}
	log.Printf("Subscribed to '%s' channel.", PresenceEventsChannel)

	for msg := range eventsChan {

		switch msg.Type {
		case "user_connected":
			log.Printf("EVENT: User connected: %s", msg.ClientID)
			if err := store.AddOnlineUser(ctx, msg.ClientID); err != nil {
				log.Printf("ERROR: Failed to add online user %s: %v", msg.ClientID, err)
			}
		case "user_disconnected":
			log.Printf("EVENT: User disconnected: %s", msg.ClientID)
			if err := store.RemoveOnlineUser(ctx, msg.ClientID); err != nil {
				log.Printf("ERROR: Failed to remove online user %s: %v", msg.ClientID, err)
			}
		default:
			// This default case was already correctly looking at msg.Type!
			log.Printf("WARNING: Unknown event type received: %s", msg.Type)
		}
	}
}