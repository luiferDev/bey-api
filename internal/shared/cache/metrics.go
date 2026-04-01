package cache

import "sync/atomic"

type CacheMetrics struct {
	hits    atomic.Int64
	misses  atomic.Int64
	sets    atomic.Int64
	deletes atomic.Int64
	errors  atomic.Int64
}

func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

func (m *CacheMetrics) Hit() {
	m.hits.Add(1)
}

func (m *CacheMetrics) Miss() {
	m.misses.Add(1)
}

func (m *CacheMetrics) Set() {
	m.sets.Add(1)
}

func (m *CacheMetrics) Delete() {
	m.deletes.Add(1)
}

func (m *CacheMetrics) Error() {
	m.errors.Add(1)
}

func (m *CacheMetrics) HitRate() float64 {
	hits := m.hits.Load()
	misses := m.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func (m *CacheMetrics) Snapshot() map[string]interface{} {
	return map[string]interface{}{
		"hits":     m.hits.Load(),
		"misses":   m.misses.Load(),
		"sets":     m.sets.Load(),
		"deletes":  m.deletes.Load(),
		"errors":   m.errors.Load(),
		"hit_rate": m.HitRate(),
		"hit_pct":  m.HitRate() * 100,
	}
}

func (m *CacheMetrics) Reset() {
	m.hits.Store(0)
	m.misses.Store(0)
	m.sets.Store(0)
	m.deletes.Store(0)
	m.errors.Store(0)
}
