package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/wailbentafat/ws-hub/backend/broker"
)

type RequestPayload struct {
	Type string `json:"type"`
}

func ListenForRequests(ctx context.Context, messageBroker broker.MessageBroker, store *Store) {
	requestsChan, err := messageBroker.Subscribe(ctx, BackendRequestsChannel)
	if err != nil {
		log.Fatalf("Failed to subscribe to requests: %v", err)
	}
	log.Printf("Subscribed to '%s' channel.", BackendRequestsChannel)

	for msg := range requestsChan {
		log.Printf("Received request for client %s: %s", msg.ClientID, msg.Data)

		var payload RequestPayload
		if err := json.Unmarshal([]byte(msg.Data.(string)), &payload); err != nil {
			log.Printf("Message is not structured JSON, echoing back.")
			publishResponse(ctx, messageBroker, msg.ClientID, msg.Data)
			continue
		}

		switch payload.Type {
		case "get_online_users":
			log.Printf("Handling 'get_online_users' request for client %s", msg.ClientID)
			users, err := store.GetOnlineUsers(ctx)
			if err != nil {
				log.Printf("ERROR: Failed to get online users: %v", err)
				continue
			}
			response := map[string]interface{}{
				"type": "online_users_list",
				"users": users,
			}
			publishResponse(ctx, messageBroker, msg.ClientID, response)
		default:
			log.Printf("Unknown request type '%s', echoing back.", payload.Type)
			publishResponse(ctx, messageBroker, msg.ClientID, msg.Data)
		}
	}
}

func publishResponse(ctx context.Context, mb broker.MessageBroker, clientID string, data interface{}) {
	responseMsg := broker.Message{
		ClientID: clientID,
		Data:     data,
	}
	if err := mb.Publish(ctx, BackendResponsesChannel, responseMsg); err != nil {
		log.Printf("ERROR: Failed to publish response for client %s: %v", clientID, err)
	}
}