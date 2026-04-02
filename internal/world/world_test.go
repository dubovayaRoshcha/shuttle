package world

import (
	"testing"
)

func TestWalkable_Bounds(t *testing.T) {
	worldTest := World{Width: 20, Height: 20, obstacles: map[Point]bool{}}
	got := worldTest.Walkable(4, 5)
	want := true
	if got != want {
		t.Errorf("Walkable(4, 5) = %t; want %t", got, want)
	}

	got = worldTest.Walkable(-3, 5)
	want = false
	if got != want {
		t.Errorf("Walkable(-3, 5) = %t; want %t", got, want)
	}

	got = worldTest.Walkable(5, -4)
	want = false
	if got != want {
		t.Errorf("Walkable(5, -4) = %t; want %t", got, want)
	}

	got = worldTest.Walkable(23, 6)
	want = false
	if got != want {
		t.Errorf("Walkable(23, 6) = %t; want %t", got, want)
	}

	got = worldTest.Walkable(2, 64)
	want = false
	if got != want {
		t.Errorf("Walkable(2, 64) = %t; want %t", got, want)
	}
}

func TestWalkable_Obstacle(t *testing.T) {
	worldTest := New(20, 20, []Point{Point{X: 10, Y: 10}, Point{X: 14, Y: 14}})
	got := worldTest.Walkable(10, 10)
	want := false
	if got != want {
		t.Errorf("Walkable(10, 10) = %t; want %t", got, want)
	}

	got = worldTest.Walkable(12, 12)
	want = true
	if got != want {
		t.Errorf("Walkable(12, 12) = %t; want %t", got, want)
	}
}

func TestNeighbors_Center_NoObstacles(t *testing.T) {
	worldTest := World{Width: 3, Height: 3, obstacles: map[Point]bool{}}
	got := worldTest.Neighbors(1, 1)
	want := []Point{Point{1, 0}, Point{2, 1}, Point{1, 2}, Point{0, 1}}
	if len(got) != len(want) {
		t.Fatalf("different length")
	}

	for i, _ := range got {
		if got[i] != want[i] {
			t.Errorf("Neighbors(1, 1) = %v; want %v", got, want)
		}
	}
}

func TestNeighbors_Edge(t *testing.T) {
	worldTest := World{Width: 3, Height: 3, obstacles: map[Point]bool{}}
	got := worldTest.Neighbors(0, 0)
	want := []Point{Point{1, 0}, Point{0, 1}}
	if len(got) != len(want) {
		t.Fatalf("different length")
	}

	for i, _ := range got {
		if got[i] != want[i] {
			t.Errorf("Neighbors(0, 0) = %v; want %v", got, want)
		}
	}
}

func TestNeighbors_WithObstacle(t *testing.T) {
	worldTest := World{Width: 3, Height: 3, obstacles: map[Point]bool{Point{X: 2, Y: 1}: true}}
	got := worldTest.Neighbors(1, 1)
	want := []Point{Point{1, 0}, Point{1, 2}, Point{0, 1}}
	if len(got) != len(want) {
		t.Fatalf("different length")
	}

	for i, _ := range got {
		if got[i] != want[i] {
			t.Errorf("Neighbors(1, 1) = %v; want %v", got, want)
		}
	}
}
