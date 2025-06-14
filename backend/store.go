package main

import (
	"context"
	"github.com/go-redis/redis/v8"
)

const onlineUsersSetKey = "online_users"

type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) AddOnlineUser(ctx context.Context, userID string) error {
	return s.rdb.SAdd(ctx, onlineUsersSetKey, userID).Err()
}

func (s *Store) RemoveOnlineUser(ctx context.Context, userID string) error {
	return s.rdb.SRem(ctx, onlineUsersSetKey, userID).Err()
}

func (s *Store) GetOnlineUsers(ctx context.Context) ([]string, error) {
	return s.rdb.SMembers(ctx, onlineUsersSetKey).Result()
}