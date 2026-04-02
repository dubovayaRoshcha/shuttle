package replanner

import (
	"testing"

	"shuttle/internal/reservations"
	"shuttle/internal/world"
)

func newTestWorld(width, height int) *world.World {
	return world.New(width, height, nil)
}

func TestService_Replan_NoReservations(t *testing.T) {
	w := newTestWorld(5, 5)
	r := reservations.NewManager()
	svc := NewService(w, r)

	start := world.Point{X: 0, Y: 0}
	goal := world.Point{X: 4, Y: 0}

	path, err := svc.Replan(start, goal, "r1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(path) == 0 {
		t.Fatal("expected non-empty path")
	}

	if path[0] != start {
		t.Fatalf("expected path to start at %+v, got %+v", start, path[0])
	}

	if path[len(path)-1] != goal {
		t.Fatalf("expected path to end at %+v, got %+v", goal, path[len(path)-1])
	}
}

func TestService_Replan_AvoidsReservedCellsOfAnotherOwner(t *testing.T) {
	w := newTestWorld(5, 5)
	r := reservations.NewManager()
	svc := NewService(w, r)

	blocked := world.Point{X: 2, Y: 0}
	if err := r.Reserve(blocked, "r2"); err != nil {
		t.Fatalf("failed to reserve point: %v", err)
	}

	start := world.Point{X: 0, Y: 0}
	goal := world.Point{X: 4, Y: 0}

	path, err := svc.Replan(start, goal, "r1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(path) == 0 {
		t.Fatal("expected non-empty path")
	}

	if path[0] != start {
		t.Fatalf("expected path to start at %+v, got %+v", start, path[0])
	}

	if path[len(path)-1] != goal {
		t.Fatalf("expected path to end at %+v, got %+v", goal, path[len(path)-1])
	}

	for _, p := range path {
		if p == blocked {
			t.Fatalf("expected path to avoid reserved point %+v, but it was included", blocked)
		}
	}
}

func TestService_Replan_AllowsReservedCellsOfSameOwner(t *testing.T) {
	w := newTestWorld(5, 5)
	r := reservations.NewManager()
	svc := NewService(w, r)

	owned := world.Point{X: 2, Y: 0}
	if err := r.Reserve(owned, "r1"); err != nil {
		t.Fatalf("failed to reserve point: %v", err)
	}

	start := world.Point{X: 0, Y: 0}
	goal := world.Point{X: 4, Y: 0}

	path, err := svc.Replan(start, goal, "r1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(path) == 0 {
		t.Fatal("expected non-empty path")
	}

	foundOwnedPoint := false
	for _, p := range path {
		if p == owned {
			foundOwnedPoint = true
			break
		}
	}

	if !foundOwnedPoint {
		t.Fatalf("expected path to allow reserved point of same owner %+v", owned)
	}
}

func TestService_Replan_NoPathBecauseOfReservations(t *testing.T) {
	w := newTestWorld(3, 3)
	r := reservations.NewManager()
	svc := NewService(w, r)

	blockedPoints := []world.Point{
		{X: 1, Y: 0},
		{X: 1, Y: 1},
		{X: 1, Y: 2},
	}

	for _, p := range blockedPoints {
		if err := r.Reserve(p, "r2"); err != nil {
			t.Fatalf("failed to reserve point %+v: %v", p, err)
		}
	}

	start := world.Point{X: 0, Y: 1}
	goal := world.Point{X: 2, Y: 1}

	path, err := svc.Replan(start, goal, "r1")
	if err == nil {
		t.Fatalf("expected error, got nil and path %+v", path)
	}
}
