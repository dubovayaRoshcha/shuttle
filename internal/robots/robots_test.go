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

func TestMultipleRobotsIsolation(t *testing.T) {
	m := NewManager()

	if err := m.Upsert(Robot{ID: "r1", X: 0, Y: 0}); err != nil {
		t.Fatalf("Upsert r1 error: %v", err)
	}

	if err := m.Upsert(Robot{ID: "r2", X: 8, Y: 8}); err != nil {
		t.Fatalf("Upsert r2 error: %v", err)
	}

	if err := m.SetBusy("r1", "T-1"); err != nil {
		t.Fatalf("SetBusy r1 error: %v", err)
	}

	if err := m.Step("r1", 1, 0, 0); err != nil {
		t.Fatalf("Step r1 error: %v", err)
	}

	r1, err := m.GetState("r1")
	if err != nil {
		t.Fatalf("GetState r1 error: %v", err)
	}

	r2, err := m.GetState("r2")
	if err != nil {
		t.Fatalf("GetState r2 error: %v", err)
	}

	if r1.X != 1 || r1.Y != 0 {
		t.Fatalf("r1 position = (%d,%d), want (1,0)", r1.X, r1.Y)
	}

	if r1.State != StateBusy {
		t.Fatalf("r1 state = %q, want %q", r1.State, StateBusy)
	}

	if r2.X != 8 || r2.Y != 8 {
		t.Fatalf("r2 position changed = (%d,%d), want (8,8)", r2.X, r2.Y)
	}

	if r2.State != StateIdle {
		t.Fatalf("r2 state = %q, want %q", r2.State, StateIdle)
	}

	list := m.List()
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	if err := m.SetFree("r1"); err != nil {
		t.Fatalf("SetFree r1 error: %v", err)
	}

	r1, err = m.GetState("r1")
	if err != nil {
		t.Fatalf("GetState r1 after SetFree error: %v", err)
	}

	if r1.State != StateIdle {
		t.Fatalf("r1 state after SetFree = %q, want %q", r1.State, StateIdle)
	}
}
