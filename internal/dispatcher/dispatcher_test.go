package dispatcher

import (
	"context"
	"testing"

	"shuttle/internal/pathfinding"
	"shuttle/internal/replanner"
	"shuttle/internal/reservations"
	"shuttle/internal/robots"
	"shuttle/internal/tasks"
	"shuttle/internal/world"
)

func newTestDispatcher(w *world.World) (*Dispatcher, *tasks.Queue, *robots.Manager, *reservations.Manager) {
	queue := tasks.NewQueue()
	manager := robots.NewManager()
	res := reservations.NewManager()
	repl := replanner.NewService(w, res)

	d := New(Options{
		Queue:        queue,
		Manager:      manager,
		World:        w,
		Reservations: res,
		Replanner:    repl,
	})

	return d, queue, manager, res
}

func mustAddTask(t *testing.T, queue *tasks.Queue, task *tasks.Task) {
	t.Helper()
	if err := queue.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}
}

func mustUpsertRobot(t *testing.T, manager *robots.Manager, robot robots.Robot) {
	t.Helper()
	if err := manager.Upsert(robot); err != nil {
		t.Fatalf("failed to upsert robot %s: %v", robot.ID, err)
	}
}

func mustReservePoint(t *testing.T, res *reservations.Manager, p world.Point, owner string) {
	t.Helper()
	if err := res.Reserve(p, owner); err != nil {
		t.Fatalf("failed to reserve point %+v: %v", p, err)
	}
}

func containsPoint(path []world.Point, target world.Point) bool {
	for _, p := range path {
		if p == target {
			return true
		}
	}
	return false
}

func indexOfPoint(path []world.Point, target world.Point) int {
	for i, p := range path {
		if p == target {
			return i
		}
	}
	return -1
}

func TestDispatcher_RunOnce_TwoPhaseSuccess(t *testing.T) {
	w := world.New(6, 6, nil)
	d, queue, manager, _ := newTestDispatcher(w)

	mustUpsertRobot(t, manager, robots.Robot{
		ID: "r1",
		X:  0,
		Y:  0,
	})

	task := &tasks.Task{
		Type: "move",
		From: world.Point{X: 2, Y: 0},
		To:   world.Point{X: 4, Y: 0},
	}
	mustAddTask(t, queue, task)

	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	gotTask, ok := queue.GetTask(task.ID)
	if !ok {
		t.Fatal("task not found")
	}

	if gotTask.Status != "assigned" {
		t.Fatalf("expected task status assigned, got %s", gotTask.Status)
	}

	if len(gotTask.Route) == 0 {
		t.Fatal("expected non-empty route")
	}

	if gotTask.Route[0] != (world.Point{X: 0, Y: 0}) {
		t.Fatalf("expected route to start at robot position, got %+v", gotTask.Route[0])
	}

	last := gotTask.Route[len(gotTask.Route)-1]
	if last != task.To {
		t.Fatalf("expected route to end at %+v, got %+v", task.To, last)
	}

	fromIdx := indexOfPoint(gotTask.Route, task.From)
	if fromIdx == -1 {
		t.Fatalf("expected route to pass through task.From %+v", task.From)
	}

	toIdx := indexOfPoint(gotTask.Route, task.To)
	if toIdx == -1 || fromIdx > toIdx {
		t.Fatalf("expected route to pass through From before To")
	}

	r1, ok := manager.Get("r1")
	if !ok {
		t.Fatal("robot r1 not found")
	}

	if r1.TaskID != task.ID {
		t.Fatalf("expected robot to be busy with task %s, got %s", task.ID, r1.TaskID)
	}
}

func TestDispatcher_RunOnce_Phase1Conflict_ReplanSuccess(t *testing.T) {
	w := world.New(6, 6, nil)
	d, queue, manager, res := newTestDispatcher(w)

	mustUpsertRobot(t, manager, robots.Robot{
		ID: "r1",
		X:  0,
		Y:  0,
	})

	task := &tasks.Task{
		Type: "move",
		From: world.Point{X: 2, Y: 0},
		To:   world.Point{X: 4, Y: 0},
	}
	mustAddTask(t, queue, task)

	// Ломаем обычный путь первой фазы: (0,0) -> (2,0) обычно идёт через (1,0).
	mustReservePoint(t, res, world.Point{X: 1, Y: 0}, "other")

	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	gotTask, ok := queue.GetTask(task.ID)
	if !ok {
		t.Fatal("task not found")
	}

	if gotTask.Status != "assigned" {
		t.Fatalf("expected task status assigned, got %s", gotTask.Status)
	}

	if len(gotTask.Route) == 0 {
		t.Fatal("expected non-empty route")
	}

	if !containsPoint(gotTask.Route, task.From) {
		t.Fatalf("expected route to pass through task.From %+v", task.From)
	}

	if gotTask.Route[len(gotTask.Route)-1] != task.To {
		t.Fatalf("expected route to end at %+v, got %+v", task.To, gotTask.Route[len(gotTask.Route)-1])
	}

	if containsPoint(gotTask.Route, world.Point{X: 1, Y: 0}) {
		t.Fatal("expected replanned route to avoid reserved point on phase 1")
	}

	r1, ok := manager.Get("r1")
	if !ok {
		t.Fatal("robot r1 not found")
	}

	if r1.TaskID != task.ID {
		t.Fatalf("expected robot to be assigned task %s, got %s", task.ID, r1.TaskID)
	}
}

func TestDispatcher_RunOnce_Phase2Conflict_ReplanSuccess(t *testing.T) {
	w := world.New(6, 6, nil)
	d, queue, manager, res := newTestDispatcher(w)

	mustUpsertRobot(t, manager, robots.Robot{
		ID: "r1",
		X:  0,
		Y:  0,
	})

	task := &tasks.Task{
		Type: "move",
		From: world.Point{X: 2, Y: 0},
		To:   world.Point{X: 4, Y: 0},
	}
	mustAddTask(t, queue, task)

	// Phase 1 остаётся свободной.
	// Ломаем обычный путь второй фазы: (2,0) -> (4,0) обычно идёт через (3,0).
	mustReservePoint(t, res, world.Point{X: 3, Y: 0}, "other")

	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	gotTask, ok := queue.GetTask(task.ID)
	if !ok {
		t.Fatal("task not found")
	}

	if gotTask.Status != "assigned" {
		t.Fatalf("expected task status assigned, got %s", gotTask.Status)
	}

	if len(gotTask.Route) == 0 {
		t.Fatal("expected non-empty route")
	}

	fromIdx := indexOfPoint(gotTask.Route, task.From)
	if fromIdx == -1 {
		t.Fatalf("expected route to pass through task.From %+v", task.From)
	}

	if gotTask.Route[len(gotTask.Route)-1] != task.To {
		t.Fatalf("expected route to end at %+v, got %+v", task.To, gotTask.Route[len(gotTask.Route)-1])
	}

	if containsPoint(gotTask.Route, world.Point{X: 3, Y: 0}) {
		t.Fatal("expected replanned route to avoid reserved point on phase 2")
	}

	r1, ok := manager.Get("r1")
	if !ok {
		t.Fatal("robot r1 not found")
	}

	if r1.TaskID != task.ID {
		t.Fatalf("expected robot to be assigned task %s, got %s", task.ID, r1.TaskID)
	}
}

func TestDispatcher_RunOnce_Phase2Fail_ReleasesPhase1(t *testing.T) {
	w := world.New(6, 6, nil)
	d, queue, manager, res := newTestDispatcher(w)

	mustUpsertRobot(t, manager, robots.Robot{
		ID: "r1",
		X:  0,
		Y:  0,
	})

	task := &tasks.Task{
		Type: "move",
		From: world.Point{X: 0, Y: 2},
		To:   world.Point{X: 4, Y: 2},
	}
	mustAddTask(t, queue, task)

	// Делаем фазу 2 невозможной для replanner:
	// полная вертикальная "стена" резервов по x=1.
	for y := 0; y < 6; y++ {
		mustReservePoint(t, res, world.Point{X: 1, Y: y}, "other")
	}

	expectedPhase1, err := pathfinding.FindPath(world.Point{X: 0, Y: 0}, task.From, w)
	if err != nil {
		t.Fatalf("failed to build expected phase1 path: %v", err)
	}

	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	gotTask, ok := queue.GetTask(task.ID)
	if !ok {
		t.Fatal("task not found")
	}

	if gotTask.Status == "assigned" {
		t.Fatalf("task should not be assigned, got status %s", gotTask.Status)
	}

	if len(gotTask.Route) != 0 {
		t.Fatalf("expected empty route, got %d points", len(gotTask.Route))
	}

	r1, ok := manager.Get("r1")
	if !ok {
		t.Fatal("robot r1 not found")
	}

	if r1.TaskID != "" {
		t.Fatalf("expected robot to stay free, got taskID %s", r1.TaskID)
	}

	// Проверяем, что резерв первой фазы откатился.
	for _, p := range expectedPhase1 {
		owner, ok := res.Owner(p)
		if ok && owner == task.ID {
			t.Fatalf("expected phase1 reservation to be released for point %+v", p)
		}
	}
}
