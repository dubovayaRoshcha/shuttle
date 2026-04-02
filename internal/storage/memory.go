package storage

import (
	"context"
	"shuttle/internal/robots"
	"sync"
)

type Memory struct {
	mu     sync.RWMutex
	robots map[string]robots.Robot
}

func NewMemory() *Memory {
	return &Memory{
		robots: make(map[string]robots.Robot),
	}
}

func (m *Memory) UpsertRobot(ctx context.Context, robot robots.Robot) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.robots[robot.ID] = robot

	return nil
}

func (m *Memory) GetRobot(ctx context.Context, id string) (robots.Robot, bool, error) {
	if ctx.Err() != nil {
		return robots.Robot{}, false, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if robot, ok := m.robots[id]; ok {
		return robot, true, nil
	}

	return robots.Robot{}, false, nil
}

func (m *Memory) ListRobots(ctx context.Context) ([]robots.Robot, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	// делаем пустой срез, а не nil
	list := make([]robots.Robot, 0, len(m.robots))
	for _, r := range m.robots {
		list = append(list, r)
	}
	return list, nil
}

func (m *Memory) Close() error {
	return nil
}

// компиляционная проверка интерфейса
var _ Storage = (*Memory)(nil)
