package reservations

import (
	"testing"

	"shuttle/internal/world"
)

func TestManagerReserve(t *testing.T) {
	m := NewManager()
	point := world.Point{X: 1, Y: 2}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}

	if !m.IsReserved(point) {
		t.Fatalf("IsReserved() = false, want true")
	}

	owner, ok := m.Owner(point)
	if !ok {
		t.Fatalf("Owner() ok = false, want true")
	}
	if owner != "r1" {
		t.Fatalf("Owner() = %q, want %q", owner, "r1")
	}
}

func TestManagerReserveConflict(t *testing.T) {
	m := NewManager()
	point := world.Point{X: 2, Y: 3}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("first Reserve() error = %v, want nil", err)
	}

	err := m.Reserve(point, "r2")
	if err == nil {
		t.Fatalf("second Reserve() error = nil, want conflict error")
	}

	owner, ok := m.Owner(point)
	if !ok {
		t.Fatalf("Owner() ok = false, want true")
	}
	if owner != "r1" {
		t.Fatalf("Owner() = %q, want %q", owner, "r1")
	}
}

func TestManagerReserveSameOwner(t *testing.T) {
	m := NewManager()
	point := world.Point{X: 4, Y: 5}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("first Reserve() error = %v, want nil", err)
	}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("second Reserve() same owner error = %v, want nil", err)
	}

	if !m.IsReserved(point) {
		t.Fatalf("IsReserved() = false, want true")
	}

	owner, ok := m.Owner(point)
	if !ok {
		t.Fatalf("Owner() ok = false, want true")
	}
	if owner != "r1" {
		t.Fatalf("Owner() = %q, want %q", owner, "r1")
	}
}

func TestManagerRelease(t *testing.T) {
	m := NewManager()
	point := world.Point{X: 6, Y: 7}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}

	if err := m.Release(point, "r1"); err != nil {
		t.Fatalf("Release() error = %v, want nil", err)
	}

	if m.IsReserved(point) {
		t.Fatalf("IsReserved() = true, want false")
	}

	owner, ok := m.Owner(point)
	if ok {
		t.Fatalf("Owner() ok = true, want false, got owner = %q", owner)
	}
	if owner != "" {
		t.Fatalf("Owner() = %q, want empty string", owner)
	}
}

func TestManagerReleaseWrongOwner(t *testing.T) {
	m := NewManager()
	point := world.Point{X: 8, Y: 9}

	if err := m.Reserve(point, "r1"); err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}

	err := m.Release(point, "r2")
	if err == nil {
		t.Fatalf("Release() wrong owner error = nil, want error")
	}

	if !m.IsReserved(point) {
		t.Fatalf("IsReserved() = false, want true")
	}

	owner, ok := m.Owner(point)
	if !ok {
		t.Fatalf("Owner() ok = false, want true")
	}
	if owner != "r1" {
		t.Fatalf("Owner() = %q, want %q", owner, "r1")
	}
}

func TestManagerReservePath(t *testing.T) {
	m := NewManager()
	path := []world.Point{
		{X: 0, Y: 0},
		{X: 0, Y: 1},
		{X: 0, Y: 2},
	}

	if err := m.ReservePath(path, "r1"); err != nil {
		t.Fatalf("ReservePath() error = %v, want nil", err)
	}

	for _, point := range path {
		if !m.IsReserved(point) {
			t.Fatalf("point %+v is not reserved, want reserved", point)
		}

		owner, ok := m.Owner(point)
		if !ok {
			t.Fatalf("Owner(%+v) ok = false, want true", point)
		}
		if owner != "r1" {
			t.Fatalf("Owner(%+v) = %q, want %q", point, owner, "r1")
		}
	}
}

func TestManagerReservePathConflict(t *testing.T) {
	m := NewManager()

	conflictPoint := world.Point{X: 1, Y: 1}
	if err := m.Reserve(conflictPoint, "r2"); err != nil {
		t.Fatalf("Reserve() conflict point error = %v, want nil", err)
	}

	path := []world.Point{
		{X: 1, Y: 0},
		{X: 1, Y: 1},
		{X: 1, Y: 2},
	}

	err := m.ReservePath(path, "r1")
	if err == nil {
		t.Fatalf("ReservePath() error = nil, want conflict error")
	}

	for _, point := range path {
		if point == conflictPoint {
			continue
		}
		if m.IsReserved(point) {
			t.Fatalf("point %+v was reserved, want not reserved because ReservePath must be atomic", point)
		}
	}

	owner, ok := m.Owner(conflictPoint)
	if !ok {
		t.Fatalf("Owner(conflictPoint) ok = false, want true")
	}
	if owner != "r2" {
		t.Fatalf("Owner(conflictPoint) = %q, want %q", owner, "r2")
	}
}

func TestManagerReleasePath(t *testing.T) {
	m := NewManager()
	path := []world.Point{
		{X: 3, Y: 0},
		{X: 3, Y: 1},
		{X: 3, Y: 2},
	}

	if err := m.ReservePath(path, "r1"); err != nil {
		t.Fatalf("ReservePath() error = %v, want nil", err)
	}

	if err := m.ReleasePath(path, "r1"); err != nil {
		t.Fatalf("ReleasePath() error = %v, want nil", err)
	}

	for _, point := range path {
		if m.IsReserved(point) {
			t.Fatalf("point %+v still reserved, want released", point)
		}

		owner, ok := m.Owner(point)
		if ok {
			t.Fatalf("Owner(%+v) ok = true, want false, got owner = %q", point, owner)
		}
		if owner != "" {
			t.Fatalf("Owner(%+v) = %q, want empty string", point, owner)
		}
	}
}

func TestManagerReleasePathWrongOwner(t *testing.T) {
	m := NewManager()
	path := []world.Point{
		{X: 5, Y: 0},
		{X: 5, Y: 1},
		{X: 5, Y: 2},
	}

	if err := m.ReservePath(path, "r1"); err != nil {
		t.Fatalf("ReservePath() error = %v, want nil", err)
	}

	err := m.ReleasePath(path, "r2")
	if err == nil {
		t.Fatalf("ReleasePath() wrong owner error = nil, want error")
	}

	for _, point := range path {
		if !m.IsReserved(point) {
			t.Fatalf("point %+v is not reserved, want still reserved", point)
		}

		owner, ok := m.Owner(point)
		if !ok {
			t.Fatalf("Owner(%+v) ok = false, want true", point)
		}
		if owner != "r1" {
			t.Fatalf("Owner(%+v) = %q, want %q", point, owner, "r1")
		}
	}
}
