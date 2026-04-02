// internal/pathfinding/astar_test.go
package pathfinding

import (
	"testing"

	"shuttle/internal/world"
)

// helper: checks that path starts/ends correctly and every step is walkable + 4-neighbor move
func validatePath(t *testing.T, w *world.World, start, goal world.Point, path []world.Point) {
	t.Helper()

	if len(path) == 0 {
		t.Fatalf("path is empty")
	}
	if path[0] != start {
		t.Fatalf("path must start with %v, got %v", start, path[0])
	}
	if path[len(path)-1] != goal {
		t.Fatalf("path must end with %v, got %v", goal, path[len(path)-1])
	}

	// all points walkable + consecutive are 4-neighbors
	for i := 0; i < len(path); i++ {
		p := path[i]
		if !w.Walkable(p.X, p.Y) {
			t.Fatalf("path contains non-walkable cell %v at index %d", p, i)
		}
		if i == 0 {
			continue
		}
		prev := path[i-1]
		dx := prev.X - p.X
		if dx < 0 {
			dx = -dx
		}
		dy := prev.Y - p.Y
		if dy < 0 {
			dy = -dy
		}
		if dx+dy != 1 {
			t.Fatalf("invalid move: %v -> %v (must be 4-neighbor)", prev, p)
		}
	}
}

func TestFindPath_EmptyMap_PathExists(t *testing.T) {
	// assumes your world has constructor: world.New(width, height int, obstacles []world.Point) *world.World
	w := world.New(10, 10, nil)

	start := world.Point{X: 1, Y: 1}
	goal := world.Point{X: 7, Y: 6}

	path, err := FindPath(start, goal, w)
	if err != nil {
		t.Fatalf("expected path, got error: %v", err)
	}

	validatePath(t, w, start, goal, path)
}

func TestFindPath_WithObstacle_WalksAround(t *testing.T) {
	// vertical wall with a gap, so path must обходить/проходить через просвет
	obstacles := []world.Point{
		{X: 3, Y: 0}, {X: 3, Y: 1}, {X: 3, Y: 2}, {X: 3, Y: 3},
		// gap at (3,4)
		{X: 3, Y: 5}, {X: 3, Y: 6}, {X: 3, Y: 7}, {X: 3, Y: 8}, {X: 3, Y: 9},
	}
	w := world.New(10, 10, obstacles)

	start := world.Point{X: 1, Y: 4}
	goal := world.Point{X: 8, Y: 4}

	path, err := FindPath(start, goal, w)
	if err != nil {
		t.Fatalf("expected path, got error: %v", err)
	}
	validatePath(t, w, start, goal, path)

	// additionally ensure it doesn't go through blocked wall cells
	for _, p := range path {
		if p.X == 3 && p.Y != 4 { // only gap is allowed
			t.Fatalf("path goes through wall cell %v", p)
		}
	}
}

func TestFindPath_NoPath_ReturnsError(t *testing.T) {
	goal := world.Point{X: 5, Y: 5}

	// surround goal with obstacles (4-neighbors blocked)
	obstacles := []world.Point{
		{X: goal.X + 1, Y: goal.Y},
		{X: goal.X - 1, Y: goal.Y},
		{X: goal.X, Y: goal.Y + 1},
		{X: goal.X, Y: goal.Y - 1},
	}
	w := world.New(10, 10, obstacles)

	start := world.Point{X: 0, Y: 0}

	path, err := FindPath(start, goal, w)
	if err == nil {
		t.Fatalf("expected error, got path: %v", path)
	}
	if err != ErrPathNotFound {
		t.Fatalf("expected ErrPathNotFound, got: %v", err)
	}
}
