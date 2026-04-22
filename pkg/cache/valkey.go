package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type ValkeyClient struct {
	client *redis.Client
	ctx    context.Context
	enabled bool
}

func NewValkeyClient(addr string, password string, db int) *ValkeyClient {
	ctx := context.Background()

	if addr == "" {
		log.Println("⚠️  Valkey not configured, using in-memory fallback")
		return &ValkeyClient{enabled: false, ctx: ctx}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Valkey connection failed: %v, using fallback", err)
		return &ValkeyClient{enabled: false, ctx: ctx}
	}

	log.Println("✅ Valkey connected")
	return &ValkeyClient{
		client:  client,
		ctx:     ctx,
		enabled: true,
	}
}

func (v *ValkeyClient) Set(key string, value interface{}, ttl time.Duration) error {
	if !v.enabled {
		// TODO: in-memory fallback with TTL
		return nil
	}
	return v.client.Set(v.ctx, key, value, ttl).Err()
}

func (v *ValkeyClient) Get(key string) (string, error) {
	if !v.enabled {
		return "", nil
	}
	return v.client.Get(v.ctx, key).Result()
}

func (v *ValkeyClient) Delete(keys ...string) error {
	if !v.enabled {
		return nil
	}
	return v.client.Del(v.ctx, keys...).Err()
}

func (v *ValkeyClient) IsEnabled() bool {
	return v.enabled
}
