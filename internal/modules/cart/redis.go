package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"bey/internal/config"
)

type RedisCartRepository struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

func NewRedisCartRepository(cfg config.CartConfig) (*RedisCartRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	ttlDays := cfg.TTLDays
	if ttlDays == 0 {
		ttlDays = 7
	}

	return &RedisCartRepository{
		client: client,
		ctx:    ctx,
		ttl:    time.Duration(ttlDays) * 24 * time.Hour,
	}, nil
}

func (r *RedisCartRepository) cartKey(userID uint) string {
	return fmt.Sprintf("cart:%d", userID)
}

func (r *RedisCartRepository) GetCart(userID uint) (*Cart, error) {
	key := r.cartKey(userID)
	data, err := r.client.Get(r.ctx, key).Bytes()
	if err == redis.Nil {
		return &Cart{
			UserID: userID,
			Items:  []CartItem{},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var cart Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, err
	}

	return &cart, nil
}

func (r *RedisCartRepository) SaveCart(cart *Cart) error {
	key := r.cartKey(cart.UserID)
	data, err := json.Marshal(cart)
	if err != nil {
		return err
	}

	if err := r.client.Set(r.ctx, key, data, r.ttl).Err(); err != nil {
		return err
	}

	return nil
}

func (r *RedisCartRepository) DeleteCart(userID uint) error {
	key := r.cartKey(userID)
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisCartRepository) ExtendTTL(userID uint) error {
	key := r.cartKey(userID)
	return r.client.Expire(r.ctx, key, r.ttl).Err()
}
