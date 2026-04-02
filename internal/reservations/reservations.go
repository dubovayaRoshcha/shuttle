package reservations

import (
	"fmt"
	"sync"

	"shuttle/internal/world"
)

type Manager struct {
	mu       sync.RWMutex
	reserved map[world.Point]string
}

func NewManager() *Manager {
	return &Manager{
		reserved: make(map[world.Point]string),
	}
}

func (m *Manager) Reserve(point world.Point, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentOwner, ok := m.reserved[point]
	if !ok {
		m.reserved[point] = owner
		return nil
	}

	if currentOwner == owner {
		return nil
	}

	return fmt.Errorf("point %+v already reserved by %s", point, currentOwner)
}

func (m *Manager) IsReserved(point world.Point) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.reserved[point]
	return ok
}

func (m *Manager) Owner(point world.Point) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	owner, ok := m.reserved[point]
	return owner, ok
}

func (m *Manager) Release(point world.Point, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentOwner, ok := m.reserved[point]
	if !ok {
		return nil
	}

	if currentOwner != owner {
		return fmt.Errorf("point %+v reserved by %s, not %s", point, currentOwner, owner)
	}

	delete(m.reserved, point)
	return nil
}

func (m *Manager) ReservePath(path []world.Point, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Проверка всего пути
	for _, p := range path {
		currentOwner, ok := m.reserved[p]
		if ok && currentOwner != owner {
			return fmt.Errorf("point %+v already reserved by %s", p, currentOwner)
		}
	}

	// 2. Резервирование
	for _, p := range path {
		m.reserved[p] = owner
	}

	return nil
}

func (m *Manager) ReleasePath(path []world.Point, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range path {
		currentOwner, ok := m.reserved[p]
		if !ok {
			continue
		}

		if currentOwner != owner {
			return fmt.Errorf("point %+v reserved by %s, not %s", p, currentOwner, owner)
		}

		delete(m.reserved, p)
	}

	return nil
}
