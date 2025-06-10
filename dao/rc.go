package dao

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

// Init Initialize Redis client
func InitRedis(addr, password string, db int) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	if _, err := Client.Ping(Ctx).Result(); err != nil {
		return errors.Wrap(err, "failed to connect to redis")
	}
	return nil
}

// SetJSON Set JSON data
func SetJSON(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal value")
	}
	return Client.Set(Ctx, key, data, expiration).Err()
}

// GetJSON Get JSON data
func GetJSON(key string, dest any) error {
	data, err := Client.Get(Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return errors.Wrap(err, "failed to get value")
	}
	return json.Unmarshal(data, dest)
}

// Del Delete key
func Del(key string) error {
	return Client.Del(Ctx, key).Err()
}

// Exists Check if key exists
func Exists(key string) (bool, error) {
	n, err := Client.Exists(Ctx, key).Result()
	return n > 0, err
}

// KeysByPrefix Get all matching keys by prefix
func KeysByPrefix(prefix string) ([]string, error) {
	var keys []string
	var cursor uint64
	var err error

	for {
		// Safely iterate keys using SCAN command
		var partialKeys []string
		partialKeys, cursor, err = Client.Scan(Ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan keys")
		}

		keys = append(keys, partialKeys...)

		if cursor == 0 { // Iteration completed
			break
		}
	}

	return keys, nil
}
