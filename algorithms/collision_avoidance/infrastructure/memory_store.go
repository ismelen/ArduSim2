package infrastructure

import (
	"sync"
	"time"

	"collision_avoidance/domain"
)

type MemoryTelemetryStore struct {
	mu           sync.RWMutex
	externalBeacons map[int64]domain.Beacon
	ownBeacon    domain.Beacon
	expTime      int64
}

func NewMemoryTelemetryStore(expirationTimeNs int64) *MemoryTelemetryStore {
	return &MemoryTelemetryStore{
		externalBeacons: make(map[int64]domain.Beacon),
		expTime:      expirationTimeNs,
	}
}

func (m *MemoryTelemetryStore) UpdateExternalBeacon(b domain.Beacon) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.externalBeacons[b.UavID] = b
}

func (m *MemoryTelemetryStore) GetExternalBeacons() map[int64]domain.Beacon {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[int64]domain.Beacon)
	now := time.Now().UnixNano()

	for id, b := range m.externalBeacons {
		if now-b.Time < m.expTime {
			res[id] = b
		}
	}
	return res
}

func (m *MemoryTelemetryStore) UpdateOwnBeacon(b domain.Beacon) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ownBeacon = b
}

func (m *MemoryTelemetryStore) GetOwnBeacon() *domain.Beacon {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If uninitialized (time is 0), return nil
	if m.ownBeacon.Time == 0 {
		return nil
	}

	cpy := m.ownBeacon
	return &cpy
}
