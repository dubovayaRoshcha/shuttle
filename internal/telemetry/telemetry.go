package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"shuttle/internal/config"
	"shuttle/internal/robots"
	"shuttle/internal/rosbridge"
	"shuttle/internal/storage"
	"sync"
	"time"
)

type Telemetry struct {
	ros       rosbridge.RosClient
	store     storage.Storage
	robots    *robots.Manager
	defaultID string

	mu      sync.RWMutex
	lastX   int
	lastY   int
	hasPose bool
}

func New(ros rosbridge.RosClient, st storage.Storage, robotsManager *robots.Manager, defaultID string) *Telemetry {
	return &Telemetry{
		ros:       ros,
		store:     st,
		robots:    robotsManager,
		defaultID: defaultID,
	}
}

func (t *Telemetry) Start(ctx context.Context) error {
	err := t.ros.SubscribeTelemetry(t.defaultID, t.handle)
	if err != nil {
		config.Error("не удалось подписаться на telemetry для " + t.defaultID)
		return err
	}

	config.Info("telemetry subscribed for " + t.defaultID)

	return nil
}

func (t *Telemetry) handle(topic string, msg json.RawMessage) {
	config.Info("telemetry message received on topic: " + topic)

	var odom struct {
		Pose struct {
			Pose struct {
				Position struct {
					X *float64 `json:"x"`
					Y *float64 `json:"y"`
				} `json:"position"`
			} `json:"pose"`
		} `json:"pose"`
	}

	if err := json.Unmarshal(msg, &odom); err != nil {
		config.Error("failed to parse /odom message: " + err.Error())
		return
	}
	if odom.Pose.Pose.Position.X == nil || odom.Pose.Pose.Position.Y == nil {
		config.Error("not full odom message")
		return
	}

	x := *odom.Pose.Pose.Position.X
	y := *odom.Pose.Pose.Position.Y

	t.mu.Lock()
	t.lastX = int(x)
	t.lastY = int(y)
	t.hasPose = true
	t.mu.Unlock()

	if err := t.robots.UpdatePosition(t.defaultID, int(x), int(y)); err != nil {
		config.Error("failed to update robot position: " + err.Error())
	}

	config.Info(fmt.Sprintf("telemetry updated robot=%s x=%d y=%d", t.defaultID, int(x), int(y)))

	if err := t.publishRobotStateFromManager(t.defaultID); err != nil {
		config.Error("failed to publish robot state from manager: " + err.Error())
	}

	r := robots.Robot{
		ID:        t.defaultID,
		X:         int(x),
		Y:         int(y),
		Theta:     0,
		Battery:   90,
		State:     "moving",
		UpdatedAt: time.Now(),
	}

	_ = t.store.UpsertRobot(context.Background(), r)
}

func (t *Telemetry) publishPoseMarker(robotID string, x, y int, source string) error {
	now := time.Now()

	marker := map[string]interface{}{
		"header": map[string]interface{}{
			"frame_id": "map",
			"stamp": map[string]interface{}{
				"sec":     now.Unix(),
				"nanosec": now.Nanosecond(),
			},
		},
		"ns":     "robot",
		"id":     1,
		"type":   2,
		"action": 0,
		"scale": map[string]interface{}{
			"x": 0.3,
			"y": 0.3,
			"z": 0.3,
		},
		"color": map[string]interface{}{
			"r": 1.0,
			"g": 0.0,
			"b": 0.0,
			"a": 1.0,
		},
		"pose": map[string]interface{}{
			"position": map[string]interface{}{
				"x": float64(x),
				"y": float64(y),
				"z": 0.0,
			},
			"orientation": map[string]interface{}{
				"x": 0.0,
				"y": 0.0,
				"z": 0.0,
				"w": 1.0,
			},
		},
	}

	poseMarkerTopic := "/robot/" + robotID + "/pose_marker"

	markerJSON, err := json.Marshal(marker)
	if err != nil {
		config.Error("failed to marshal pose marker: " + err.Error())
	} else {
		config.Info("publishing pose marker from " + source + " to " + poseMarkerTopic + ": " + string(markerJSON))
	}

	client, ok := t.ros.(*rosbridge.Client)
	if !ok {
		return fmt.Errorf("ros client does not support marker publishing")
	}

	if err := client.PublishMarker(poseMarkerTopic, marker); err != nil {
		return err
	}

	return nil
}

func (t *Telemetry) PublishCurrentPoseMarker() error {
	t.mu.RLock()
	x := t.lastX
	y := t.lastY
	hasPose := t.hasPose
	t.mu.RUnlock()

	if !hasPose {
		return fmt.Errorf("current pose is not available yet")
	}

	if err := t.publishPoseMarker(t.defaultID, x, y, "republish"); err != nil {
		return err
	}

	config.Info(fmt.Sprintf("current pose marker re-published robot=%s x=%d y=%d", t.defaultID, x, y))
	return nil
}

func (t *Telemetry) publishRobotStateFromManager(robotID string) error {
	if robotID == "" {
		return fmt.Errorf("robot id is empty")
	}

	r, err := t.robots.GetState(robotID)
	if err != nil {
		return err
	}

	if err := t.publishPoseMarker(robotID, r.X, r.Y, "manager"); err != nil {
		return err
	}

	if err := t.ros.PublishPose(robotID, r.X, r.Y, r.Theta); err != nil {
		return err
	}

	config.Info(fmt.Sprintf(
		"robot state published from manager robot=%s x=%d y=%d theta=%.2f",
		r.ID, r.X, r.Y, r.Theta,
	))

	return nil
}

func (t *Telemetry) PublishRobotState(robotID string) error {
	if robotID == "" {
		return fmt.Errorf("robot id is empty")
	}

	if err := t.publishRobotStateFromManager(robotID); err != nil {
		return err
	}

	if robotID == t.defaultID {
		r, err := t.robots.GetState(robotID)
		if err != nil {
			return err
		}

		t.mu.Lock()
		t.lastX = r.X
		t.lastY = r.Y
		t.hasPose = true
		t.mu.Unlock()
	}

	config.Info("robot state published on demand for " + robotID)
	return nil
}
