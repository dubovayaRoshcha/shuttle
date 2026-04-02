package storage

import (
	"context"
	"errors"
	"testing"

	"shuttle/internal/robots"
)

func TestUpsertAndGetRobot(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	r1 := robots.Robot{ID: "r1", Battery: 90}
	if err := m.UpsertRobot(ctx, r1); err != nil {
		t.Fatalf("UpsertRobot failed: %v", err)
	}

	got, ok, err := m.GetRobot(ctx, "r1")
	if err != nil {
		t.Fatalf("GetRobot unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("GetRobot: want ok=true, got false")
	}
	if got.ID != r1.ID || got.Battery != r1.Battery {
		t.Fatalf("GetRobot: mismatch: got=%+v want=%+v", got, r1)
	}

	// update
	r1.Battery = 75
	if err := m.UpsertRobot(ctx, r1); err != nil {
		t.Fatalf("UpsertRobot(update) failed: %v", err)
	}
	got, ok, err = m.GetRobot(ctx, "r1")
	if err != nil || !ok {
		t.Fatalf("GetRobot after update: err=%v ok=%v", err, ok)
	}
	if got.Battery != 75 {
		t.Fatalf("GetRobot after update: want Battery=75, got %d", got.Battery)
	}
}

func TestListRobots(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	rs := []robots.Robot{
		{ID: "a", Battery: 10},
		{ID: "b", Battery: 20},
		{ID: "c", Battery: 30},
	}
	for _, r := range rs {
		if err := m.UpsertRobot(ctx, r); err != nil {
			t.Fatalf("UpsertRobot %s failed: %v", r.ID, err)
		}
	}

	list, err := m.ListRobots(ctx)
	if err != nil {
		t.Fatalf("ListRobots error: %v", err)
	}
	if len(list) != len(rs) {
		t.Fatalf("ListRobots: want %d, got %d", len(rs), len(list))
	}

	want := map[string]bool{"a": true, "b": true, "c": true}
	for _, r := range list {
		if !want[r.ID] {
			t.Fatalf("ListRobots: unexpected ID %q", r.ID)
		}
		delete(want, r.ID)
	}
	if len(want) != 0 {
		t.Fatalf("ListRobots: missing IDs: %v", want)
	}
}

func TestContextCanceled(t *testing.T) {
	m := NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем до вызова

	err := m.UpsertRobot(ctx, robots.Robot{ID: "x", Battery: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("UpsertRobot with canceled ctx: want context.Canceled, got %v", err)
	}

	// Для полноты — Get и List тоже должны уважать ctx.
	if _, _, err := m.GetRobot(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetRobot with canceled ctx: want context.Canceled, got %v", err)
	}
	if _, err := m.ListRobots(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListRobots with canceled ctx: want context.Canceled, got %v", err)
	}
}
