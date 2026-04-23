package robots

import (
	"errors"
	"sync"
	"time"
)

const (
	StateIdle  = "idle"
	StateBusy  = "busy"
	StateError = "error"
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

func (m *Manager) updateRobotLocked(id string, update func(*Robot) error) error {
	robot, ok := m.robots[id]
	if !ok {
		return errors.New("There is no robot with such an ID")
	}

	if err := update(&robot); err != nil {
		return err
	}

	robot.UpdatedAt = time.Now()
	m.robots[id] = robot

	return nil
}

func isValidState(state string) bool {
	switch state {
	case StateIdle, StateBusy, StateError:
		return true
	default:
		return false
	}
}

func (m *Manager) Upsert(r Robot) error {
	if r.ID == "" {
		return errors.New("id of robot is empty")
	}

	if r.State == "" {
		r.State = StateIdle
	}

	if !isValidState(r.State) {
		return errors.New("invalid robot state")
	}

	r.UpdatedAt = time.Now()

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

	return m.updateRobotLocked(id, func(robot *Robot) error {
		robot.TaskID = taskID
		robot.State = StateBusy
		return nil
	})
}

func (m *Manager) SetFree(id string) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.updateRobotLocked(id, func(robot *Robot) error {
		robot.TaskID = ""
		robot.State = StateIdle
		return nil
	})
}

func (m *Manager) UpdatePosition(id string, x, y int) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.updateRobotLocked(id, func(robot *Robot) error {
		robot.X = x
		robot.Y = y
		return nil
	})
}

func (m *Manager) UpdateState(id string, x, y int, theta float64, state string) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	if !isValidState(state) {
		return errors.New("invalid robot state")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.updateRobotLocked(id, func(robot *Robot) error {
		robot.X = x
		robot.Y = y
		robot.Theta = theta
		robot.State = state
		return nil
	})
}

func (m *Manager) GetState(id string) (Robot, error) {
	if id == "" {
		return Robot{}, errors.New("id of robot is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	robot, ok := m.robots[id]
	if !ok {
		return Robot{}, errors.New("There is no robot with such an ID")
	}

	return robot, nil
}

func (m *Manager) Step(id string, x, y int, theta float64) error {
	if id == "" {
		return errors.New("id of robot is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.updateRobotLocked(id, func(robot *Robot) error {
		robot.X = x
		robot.Y = y
		robot.Theta = theta
		robot.State = StateBusy
		return nil
	})
}
