package dispatcher

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"shuttle/internal/pathfinding"
	"shuttle/internal/replanner"
	"shuttle/internal/reservations"
	"shuttle/internal/robots"
	"shuttle/internal/rosbridge"
	"shuttle/internal/tasks"
	"shuttle/internal/world"
	"time"
)

func abs(num int) int {
	if num < 0 {
		return -num
	}

	return num
}

func calcTheta(from, to world.Point) float64 {
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	return math.Atan2(dy, dx)
}

func toRoutePoints(path []world.Point) []rosbridge.RoutePoint {
	points := make([]rosbridge.RoutePoint, 0, len(path))
	for _, p := range path {
		points = append(points, rosbridge.RoutePoint{
			X: p.X,
			Y: p.Y,
		})
	}

	return points
}

func toPathPoints(path []world.Point) []rosbridge.PathPoint {
	points := make([]rosbridge.PathPoint, 0, len(path))
	for _, p := range path {
		points = append(points, rosbridge.PathPoint{
			X: p.X,
			Y: p.Y,
		})
	}

	return points
}

func buildPathMarker(path []world.Point) map[string]interface{} {
	marker := map[string]interface{}{
		"header": map[string]interface{}{
			"frame_id": "map",
		},
		"ns":     "path",
		"id":     1,
		"type":   4, // LINE_STRIP
		"action": 0,
		"scale": map[string]interface{}{
			"x": 0.1,
		},
		"color": map[string]interface{}{
			"r": 0.0,
			"g": 1.0,
			"b": 0.0,
			"a": 1.0,
		},
	}

	points := make([]map[string]interface{}, 0, len(path))
	for _, p := range path {
		points = append(points, map[string]interface{}{
			"x": float64(p.X),
			"y": float64(p.Y),
			"z": 0.0,
		})
	}

	marker["points"] = points
	return marker
}

type RobotStatePublisher interface {
	PublishRobotState(robotID string) error
}

type Options struct {
	Queue        *tasks.Queue
	Manager      *robots.Manager
	World        *world.World
	Reservations *reservations.Manager
	Replanner    *replanner.Service
	ROS          *rosbridge.Client
	Publisher    RobotStatePublisher
}

type Dispatcher struct {
	Queue        *tasks.Queue
	Manager      *robots.Manager
	World        *world.World
	Reservations *reservations.Manager
	Replanner    *replanner.Service
	ROS          *rosbridge.Client
	Publisher    RobotStatePublisher
}

func New(opt Options) *Dispatcher {
	return &Dispatcher{
		Queue:        opt.Queue,
		Manager:      opt.Manager,
		World:        opt.World,
		Reservations: opt.Reservations,
		Replanner:    opt.Replanner,
		ROS:          opt.ROS,
		Publisher:    opt.Publisher,
	}
}

func (d *Dispatcher) executeTask(robotID, taskID string, route []world.Point) {
	slog.Info("task execution started",
		"robot_id", robotID,
		"task_id", taskID,
		"route_len", len(route),
	)

	if len(route) == 0 {
		slog.Warn("empty route for execution",
			"robot_id", robotID,
			"task_id", taskID,
		)
		d.finishTask(robotID, taskID, "failed", route)
		return
	}

	currentRobot, err := d.Manager.GetState(robotID)
	if err != nil {
		slog.Error("failed to get robot state before execution",
			"robot_id", robotID,
			"task_id", taskID,
			"error", err,
		)
		d.finishTask(robotID, taskID, "failed", route)
		return
	}

	startIndex := 0
	if len(route) > 0 && currentRobot.X == route[0].X && currentRobot.Y == route[0].Y {
		startIndex = 1
	}

	currentPoint := world.Point{X: currentRobot.X, Y: currentRobot.Y}

	for i := startIndex; i < len(route); i++ {
		nextPoint := route[i]
		theta := calcTheta(currentPoint, nextPoint)

		err := d.Manager.Step(robotID, nextPoint.X, nextPoint.Y, theta)
		if err != nil {
			slog.Error("failed to step robot",
				"robot_id", robotID,
				"task_id", taskID,
				"step", i,
				"x", nextPoint.X,
				"y", nextPoint.Y,
				"theta", theta,
				"error", err,
			)

			d.finishTask(robotID, taskID, "failed", route)

			slog.Info("task execution failed",
				"robot_id", robotID,
				"task_id", taskID,
				"failed_step", i,
			)
			return
		}

		if err := d.Publisher.PublishRobotState(robotID); err != nil {
			slog.Error("failed to publish robot state after step",
				"robot_id", robotID,
				"task_id", taskID,
				"step", i,
				"x", nextPoint.X,
				"y", nextPoint.Y,
				"theta", theta,
				"error", err,
			)

			d.finishTask(robotID, taskID, "failed", route)

			slog.Info("task execution failed",
				"robot_id", robotID,
				"task_id", taskID,
				"failed_step", i,
			)
			return
		}

		slog.Info("robot moved",
			"robot_id", robotID,
			"task_id", taskID,
			"step", i,
			"x", nextPoint.X,
			"y", nextPoint.Y,
			"theta", theta,
		)

		currentPoint = nextPoint

		time.Sleep(500 * time.Millisecond)
	}

	d.finishTask(robotID, taskID, "completed", route)

	slog.Info("task execution completed",
		"robot_id", robotID,
		"task_id", taskID,
	)
}

func (d *Dispatcher) finishTask(robotID, taskID, status string, route []world.Point) {
	if err := d.Queue.UpdateStatus(taskID, status); err != nil {
		slog.Error("failed to update task status",
			"robot_id", robotID,
			"task_id", taskID,
			"status", status,
			"error", err,
		)
	}

	if err := d.Manager.SetFree(robotID); err != nil {
		slog.Error("failed to free robot",
			"robot_id", robotID,
			"task_id", taskID,
			"status", status,
			"error", err,
		)
	} else {
		if err := d.Publisher.PublishRobotState(robotID); err != nil {
			slog.Error("failed to publish final robot state",
				"robot_id", robotID,
				"task_id", taskID,
				"status", status,
				"error", err,
			)
		}
	}

	if err := d.Reservations.ReleasePath(route, taskID); err != nil {
		slog.Error("failed to release reservations",
			"robot_id", robotID,
			"task_id", taskID,
			"status", status,
			"error", err,
		)
	}
}

func (d *Dispatcher) RunOnce(ctx context.Context) error {
	if d.Queue == nil {
		return errors.New("expected queue")
	}

	if d.Manager == nil {
		return errors.New("expected manager")
	}

	if d.World == nil {
		return errors.New("expected world")
	}

	if d.Reservations == nil {
		return errors.New("expected reservations")
	}

	if d.Replanner == nil {
		return errors.New("expected replanner")
	}

	if d.ROS == nil {
		return errors.New("expected rosbridge")
	}

	if d.Publisher == nil {
		return errors.New("expected robot state publisher")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		task, ok := d.Queue.NextPending()
		if !ok {
			return nil
		}

		var idleRobot robots.Robot
		listRobot := d.Manager.List()
		minDist := math.MaxInt64

		for _, robot := range listRobot {
			if robot.TaskID == "" {
				manhDist := abs(robot.X-task.From.X) + abs(robot.Y-task.From.Y)
				if manhDist < minDist {
					idleRobot = robot
					minDist = manhDist
				} else if manhDist == minDist {
					if robot.ID < idleRobot.ID {
						idleRobot = robot
						minDist = manhDist
					}
				}
			}
		}

		if idleRobot.ID == "" {
			return nil
		}

		start := world.Point{X: idleRobot.X, Y: idleRobot.Y}

		// Phase 1: robot -> task.From
		phase1, err := pathfinding.FindPath(start, task.From, d.World)
		if err != nil {
			upErr := d.Queue.UpdateStatus(task.ID, "failed")
			if upErr != nil {
				return upErr
			}
			return nil
		}

		err = d.Reservations.ReservePath(phase1, task.ID)
		if err != nil {
			// обычный путь конфликтует, пробуем replanner
			newPhase1, replanErr := d.Replanner.Replan(start, task.From, task.ID)
			if replanErr != nil {
				return nil
			}

			err = d.Reservations.ReservePath(newPhase1, task.ID)
			if err != nil {
				return nil
			}

			phase1 = newPhase1
		}

		// Phase 2: task.From -> task.To
		phase2, err := pathfinding.FindPath(task.From, task.To, d.World)
		if err != nil {
			releaseErr := d.Reservations.ReleasePath(phase1, task.ID)
			if releaseErr != nil {
				return releaseErr
			}
			return nil
		}

		err = d.Reservations.ReservePath(phase2, task.ID)
		if err != nil {
			newPhase2, replanErr := d.Replanner.Replan(task.From, task.To, task.ID)
			if replanErr != nil {
				releaseErr := d.Reservations.ReleasePath(phase1, task.ID)
				if releaseErr != nil {
					return releaseErr
				}
				return nil
			}

			err = d.Reservations.ReservePath(newPhase2, task.ID)
			if err != nil {
				releaseErr := d.Reservations.ReleasePath(phase1, task.ID)
				if releaseErr != nil {
					return releaseErr
				}
				return nil
			}

			phase2 = newPhase2
		}

		// ===== ФИНАЛ: склейка маршрута =====
		var path []world.Point

		for _, cell := range phase1 {
			path = append(path, cell)
		}

		if len(phase2) > 0 {
			if len(path) > 0 && path[len(path)-1] == phase2[0] {
				// нормальный случай — пропускаем дубликат
				for _, cell := range phase2[1:] {
					path = append(path, cell)
				}
			} else {
				// защитный случай — добавляем всё
				for _, cell := range phase2 {
					path = append(path, cell)
				}
			}
		}

		task.Route = path

		err = d.Manager.SetBusy(idleRobot.ID, task.ID)
		if err != nil {
			// откатываем оба резерва (на всякий случай)
			_ = d.Reservations.ReleasePath(phase1, task.ID)
			_ = d.Reservations.ReleasePath(phase2, task.ID)
			return err
		}

		updatedRobots := d.Manager.List()
		for _, robot := range updatedRobots {
			if robot.ID == idleRobot.ID {
				slog.Info("robot assigned",
					"robot_id", robot.ID,
					"state", robot.State,
					"task_id", robot.TaskID,
				)
				break
			}
		}

		err = d.Queue.UpdateStatus(task.ID, "assigned")
		if err != nil {
			_ = d.Manager.SetFree(idleRobot.ID)
			_ = d.Reservations.ReleasePath(phase1, task.ID)
			_ = d.Reservations.ReleasePath(phase2, task.ID)
			return err
		}

		if err := d.Publisher.PublishRobotState(idleRobot.ID); err != nil {
			slog.Error("failed to publish robot state after assignment",
				"robot_id", idleRobot.ID,
				"task_id", task.ID,
				"error", err,
			)
		}

		routePoints := toRoutePoints(path)

		err = d.ROS.PublishRoute(idleRobot.ID, task.ID, routePoints)
		if err != nil {
			_ = d.Manager.SetFree(idleRobot.ID)
			_ = d.Queue.UpdateStatus(task.ID, "pending")
			_ = d.Reservations.ReleasePath(phase1, task.ID)
			_ = d.Reservations.ReleasePath(phase2, task.ID)
			return err
		}

		routeCopy := append([]world.Point(nil), task.Route...)
		go d.executeTask(idleRobot.ID, task.ID, routeCopy)

		marker := buildPathMarker(path)

		if err := d.ROS.PublishMarker("/robot/"+idleRobot.ID+"/path_marker", marker); err != nil {
			slog.Error("failed to publish path marker",
				"robot_id", idleRobot.ID,
				"task_id", task.ID,
				"error", err,
			)
		} else {
			slog.Info("path marker published",
				"robot_id", idleRobot.ID,
				"task_id", task.ID,
				"points_count", len(path),
			)
		}

		pathPoints := toPathPoints(path)

		if err := d.ROS.PublishPath(idleRobot.ID, pathPoints); err != nil {
			slog.Error("failed to publish visualization path",
				"robot_id", idleRobot.ID,
				"task_id", task.ID,
				"error", err,
			)
		} else {
			slog.Info("visualization path published",
				"robot_id", idleRobot.ID,
				"task_id", task.ID,
				"points_count", len(pathPoints),
			)
		}

		return nil
	}
}
