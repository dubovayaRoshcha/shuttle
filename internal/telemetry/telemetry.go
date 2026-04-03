package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"shuttle/internal/config"
	"shuttle/internal/robots"
	"shuttle/internal/rosbridge"
	"shuttle/internal/storage"
	"time"
)

type Telemetry struct {
	ros       rosbridge.RosClient
	store     storage.Storage
	robots    *robots.Manager
	defaultID string
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

	marker := map[string]interface{}{
		"header": map[string]interface{}{
			"frame_id": "map",
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
				"x": x,
				"y": y,
				"z": 0.0,
			},
		},
	}

	if client, ok := t.ros.(*rosbridge.Client); ok {
		err := client.PublishMarker(
			"/robot/"+t.defaultID+"/pose_marker",
			marker,
		)
		if err != nil {
			config.Error("failed to publish marker: " + err.Error())
		}
	}

	if err := t.robots.UpdatePosition(t.defaultID, int(x), int(y)); err != nil {
		config.Error("failed to update robot position: " + err.Error())
	}

	config.Info(fmt.Sprintf("telemetry updated robot=%s x=%d y=%d", t.defaultID, int(x), int(y)))

	poseMsg := struct {
		RobotID string `json:"robot_id"`
		X       int    `json:"x"`
		Y       int    `json:"y"`
	}{
		RobotID: t.defaultID,
		X:       int(x),
		Y:       int(y),
	}

	data, err := json.Marshal(poseMsg)
	if err != nil {
		config.Error("failed to marshal pose: " + err.Error())
		return
	}

	poseTopic := "/robot/" + t.defaultID + "/pose"

	if err := t.ros.Publish(poseTopic, data); err != nil {
		config.Error("failed to publish pose: " + err.Error())
	} else {
		config.Info("pose published to " + poseTopic)
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
