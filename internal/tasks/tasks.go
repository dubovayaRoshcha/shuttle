// in-memory очередь задач для этапа 2.1
package tasks

import (
	"errors"
	"shuttle/internal/world"
	"strconv"
	"sync"
	"time"
)

type Task struct {
	ID        string
	Type      string // move | pickup | dropoff
	From      world.Point
	To        world.Point
	Status    string // pending | assigned | in_progress | done | failed
	CreatedAt time.Time
	UpdatedAt time.Time
	Route     []world.Point
}

type Queue struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	order  []string
	nextID int
}

func NewQueue() *Queue {
	return &Queue{
		tasks: make(map[string]*Task),
		order: make([]string, 0),
	}
}

func (q *Queue) AddTask(t *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if t == nil {
		return errors.New("The task is empty")
	}

	q.nextID++
	t.ID = "T-" + strconv.Itoa(q.nextID)
	t.Status = "pending"
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt

	q.tasks[t.ID] = t
	q.order = append(q.order, t.ID)

	return nil
}

func (q *Queue) GetTask(id string) (*Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	task, ok := q.tasks[id]
	if !ok {
		return nil, false
	}

	return task, true
}

func (q *Queue) ListTasks() []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()
	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (q *Queue) NextPending() (*Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, id := range q.order {
		task, ok := q.tasks[id]
		if !ok {
			continue
		}
		if task.Status == "pending" {
			return task, true
		}
	}

	return nil, false
}

func (q *Queue) UpdateStatus(id string, status string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.tasks[id]
	if !ok {
		return errors.New("There is no such task")
	}
	task.Status = status
	task.UpdatedAt = time.Now()
	return nil
}
