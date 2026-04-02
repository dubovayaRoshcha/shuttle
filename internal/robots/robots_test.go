// internal/robots/manager_test.go
package robots

import (
	"sync"
	"testing"
)

func TestUpsertAndGet(t *testing.T) {
	m := NewManager()

	r := Robot{ID: "r1", X: 1, Y: 2}
	if err := m.Upsert(r); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	got, ok := m.Get("r1")
	if !ok {
		t.Fatalf("Get() ok=false, want true")
	}
	if got.ID != "r1" {
		t.Fatalf("Get().ID=%q, want %q", got.ID, "r1")
	}
	if got.X != 1 || got.Y != 2 {
		t.Fatalf("Get().pos=(%d,%d), want (1,2)", got.X, got.Y)
	}
}

func TestSetBusySetFree(t *testing.T) {
	m := NewManager()

	if err := m.Upsert(Robot{ID: "r1"}); err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	// нельзя пустой taskID
	if err := m.SetBusy("r1", ""); err == nil {
		t.Fatalf("SetBusy() expected error for empty taskID")
	}

	if err := m.SetBusy("r1", "task-1"); err != nil {
		t.Fatalf("SetBusy() error: %v", err)
	}

	got, ok := m.Get("r1")
	if !ok {
		t.Fatalf("Get() ok=false, want true")
	}
	if got.TaskID != "task-1" {
		t.Fatalf("TaskID=%q, want %q", got.TaskID, "task-1")
	}

	if err := m.SetFree("r1"); err != nil {
		t.Fatalf("SetFree() error: %v", err)
	}

	got, ok = m.Get("r1")
	if !ok {
		t.Fatalf("Get() ok=false, want true")
	}
	if got.TaskID != "" {
		t.Fatalf("TaskID=%q, want empty", got.TaskID)
	}
}

func TestListNotNil(t *testing.T) {
	m := NewManager()

	list := m.List()
	if list == nil {
		t.Fatalf("List() returned nil, want empty slice")
	}

	_ = m.Upsert(Robot{ID: "r1"})
	list = m.List()
	if list == nil {
		t.Fatalf("List() returned nil, want non-nil slice")
	}
	if len(list) != 1 {
		t.Fatalf("List() len=%d, want 1", len(list))
	}
}

func TestConcurrentUpsertGet(t *testing.T) {
	m := NewManager()

	const workers = 20
	const iters = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			id := "r-conc"

			for i := 0; i < iters; i++ {
				_ = m.Upsert(Robot{ID: id, X: w, Y: i})
				_, _ = m.Get(id)
				_ = m.SetBusy(id, "task-1")
				_ = m.SetFree(id)
				_ = m.UpdatePosition(id, i, w)
			}
		}()
	}

	wg.Wait()

	// после всех гонок робот должен существовать
	if _, ok := m.Get("r-conc"); !ok {
		t.Fatalf("expected robot to exist after concurrent operations")
	}
}
