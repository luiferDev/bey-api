package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = fmt.Errorf("cache miss")

type CacheService struct {
	client  *redis.Client
	ttl     time.Duration
	metrics *CacheMetrics
}

func NewCacheService(client *redis.Client, ttl time.Duration, metrics *CacheMetrics) *CacheService {
	return &CacheService{
		client:  client,
		ttl:     ttl,
		metrics: metrics,
	}
}

func (s *CacheService) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		s.metrics.Miss()
		return false, nil
	}
	if err != nil {
		s.metrics.Error()
		return false, err
	}

	s.metrics.Hit()
	if err := json.Unmarshal(data, dest); err != nil {
		s.metrics.Error()
		return false, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	return true, nil
}

func (s *CacheService) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		s.metrics.Error()
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		s.metrics.Error()
		return err
	}

	s.metrics.Set()
	return nil
}

func (s *CacheService) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, key).Err(); err != nil {
		s.metrics.Error()
		return err
	}

	s.metrics.Delete()
	return nil
}

func (s *CacheService) InvalidatePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	batchSize := int64(100)
	var deleted int64

	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			s.metrics.Error()
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := s.client.Del(ctx, keys...).Err(); err != nil {
				s.metrics.Error()
				return fmt.Errorf("failed to delete keys: %w", err)
			}
			deleted += int64(len(keys))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	s.metrics.Delete()
	return nil
}

func (s *CacheService) Key(parts ...string) string {
	return strings.Join(parts, ":")
}
