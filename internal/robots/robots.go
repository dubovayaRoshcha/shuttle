package robots

import (
	"errors"
	"sync"
	"time"
)

type Robot struct {
	ID        string    `json:"id"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Theta     float64   `json:"theta"`
	Battery   int       `json:"battery"`
	State     string    `json:"state"`
	TaskID    string    `json:"task_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Manager struct {
	mu     sync.RWMutex
	robots map[string]Robot
}

func NewManager() *Manager {
	return &Manager{
		robots: make(map[string]Robot),
	}
}

func (m *Manager) Upsert(r Robot) error {
	if r.ID == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.robots[r.ID] = r
	return nil
}

func (m *Manager) Get(id string) (Robot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	robot, ok := m.robots[id]
	if ok {
		return robot, ok
	}

	return Robot{}, ok
}

func (m *Manager) List() []Robot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	listRobots := make([]Robot, 0)

	for _, robot := range m.robots {
		listRobots = append(listRobots, robot)
	}

	return listRobots
}

func (m *Manager) SetBusy(id, taskID string) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	if taskID == "" {
		return errors.New("id of task is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	robot, ok := m.robots[id]
	if !ok {
		return errors.New("There is no robot with such an ID")
	}

	robot.TaskID = taskID
	robot.State = "busy"
	m.robots[id] = robot
	return nil
}

func (m *Manager) SetFree(id string) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	robot, ok := m.robots[id]
	if !ok {
		return errors.New("There is no robot with such an ID")
	}

	robot.TaskID = ""
	robot.State = "idle"
	m.robots[id] = robot
	return nil
}

func (m *Manager) UpdatePosition(id string, x, y int) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	robot, ok := m.robots[id]
	if !ok {
		return errors.New("There is no robot with such an ID")
	}

	robot.X = x
	robot.Y = y
	m.robots[id] = robot
	return nil
}
