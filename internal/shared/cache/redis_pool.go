package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisPool struct {
	addr     string
	password string
	mu       sync.RWMutex
	clients  map[int]*redis.Client
}

func NewRedisPool(addr, password string, defaultDB int) (*RedisPool, error) {
	pool := &RedisPool{
		addr:     addr,
		password: password,
		clients:  make(map[int]*redis.Client),
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       defaultDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	pool.clients[defaultDB] = client
	return pool, nil
}

func (p *RedisPool) GetClient(db int) *redis.Client {
	p.mu.RLock()
	client, exists := p.clients[db]
	p.mu.RUnlock()

	if exists {
		return client
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists = p.clients[db]; exists {
		return client
	}

	client = redis.NewClient(&redis.Options{
		Addr:     p.addr,
		Password: p.password,
		DB:       db,
	})
	p.clients[db] = client
	return client
}

func (p *RedisPool) Ping() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for db, client := range p.clients {
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis db %d ping failed: %w", db, err)
		}
	}
	return nil
}

func (p *RedisPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for db, client := range p.clients {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("redis db %d close failed: %w", db, err)
		}
	}
	p.clients = make(map[int]*redis.Client)
	return firstErr
}
