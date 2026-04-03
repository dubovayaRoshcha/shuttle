package rosbridge

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type Handler func(topic string, msg json.RawMessage)

type Options struct {
	URL string
}
type Client struct {
	opts       Options
	subs       map[string][]Handler
	conn       *websocket.Conn
	mu         sync.RWMutex
	writeMu    sync.Mutex
	advertised map[string]string
}

type RoutePoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type RouteMessage struct {
	RobotID string       `json:"robot_id"`
	TaskID  string       `json:"task_id"`
	Points  []RoutePoint `json:"points"`
}

type PathPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type PathMessage struct {
	RobotID string      `json:"robot_id"`
	Points  []PathPoint `json:"points"`
}

type PoseMessage struct {
	RobotID string  `json:"robot_id"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Theta   float64 `json:"theta"`
}

type RosClient interface {
	Connect(ctx context.Context) error
	Subscribe(topic string, h Handler) error
	SubscribeTelemetry(robotID string, h Handler) error
	Close() error
	Publish(topic string, msg json.RawMessage) error
}

var _ RosClient = (*Client)(nil)

func New(opts Options) *Client {
	return &Client{opts: opts, subs: make(map[string][]Handler), advertised: make(map[string]string)}
}

func (c *Client) Connect(ctx context.Context) error {
	if c.opts.URL == "" {
		return errors.New("rosbridge: empty URL")
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, c.opts.URL, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop()

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) Subscribe(topic string, h Handler) error {
	if topic == "" || h == nil {
		return errors.New("rosbridge: empty topic or handler")
	}

	c.mu.Lock()
	c.subs[topic] = append(c.subs[topic], h)
	c.mu.Unlock()

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("rosbridge: not connected")
	}

	payload := map[string]interface{}{
		"op":    "subscribe",
		"topic": topic,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) Unsubscribe(topic string) {
	if topic == "" {
		return
	}

	c.mu.Lock()
	delete(c.subs, topic)
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	payload := map[string]interface{}{
		"op":    "unsubscribe",
		"topic": topic,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

// InjectPublish вручную вызывает подписчиков для указанного топика.
// Используется только для тестов и отладки без реального rosbridge.
func (c *Client) InjectPublish(topic string, msg json.RawMessage) {
	c.mu.RLock()
	arr := append([]Handler(nil), c.subs[topic]...)
	c.mu.RUnlock()
	for _, h := range arr {
		h(topic, msg)
	}

}

func (c *Client) Publish(topic string, msg json.RawMessage) error {
	if topic == "" {
		return errors.New("rosbridge: empty topic")
	}

	msgType := topicType(topic)
	err := c.ensureAdvertised(topic, msgType)
	if err != nil {
		return err
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("rosbridge: not connected")
	}

	payload := map[string]interface{}{
		"op":    "publish",
		"topic": topic,
		"msg":   json.RawMessage(msg),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) PublishRoute(robotID string, taskID string, points []RoutePoint) error {
	if robotID == "" {
		return errors.New("rosbridge: empty robot id")
	}
	if taskID == "" {
		return errors.New("rosbridge: empty task id")
	}

	msg := RouteMessage{
		RobotID: robotID,
		TaskID:  taskID,
		Points:  points,
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	wrapped, err := json.Marshal(map[string]string{
		"data": string(raw),
	})
	if err != nil {
		return err
	}

	topic := "/robot/" + robotID + "/route"
	return c.Publish(topic, wrapped)
}

func (c *Client) SubscribeTelemetry(robotID string, h Handler) error {
	if robotID == "" {
		return errors.New("rosbridge: empty robot id")
	}
	return c.Subscribe("/robot/"+robotID+"/odom", h)
}

func (c *Client) readLoop() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var incoming struct {
			Op    string          `json:"op"`
			Topic string          `json:"topic"`
			Msg   json.RawMessage `json:"msg"`
		}

		if err := json.Unmarshal(data, &incoming); err != nil {
			continue
		}

		if incoming.Op == "publish" {
			c.InjectPublish(incoming.Topic, incoming.Msg)
		}
	}
}

// 	ВИЗУАЛИЗАЦИЯ

// PublishPath публикует путь робота в отдельный topic визуализации (/robot/<id>/path).
// Используется для отображения маршрута в ROS/RViz, не связан напрямую с командным route.
func (c *Client) PublishPath(robotID string, points []PathPoint) error {
	if robotID == "" {
		return errors.New("rosbridge: empty robot id")
	}

	msg := PathMessage{
		RobotID: robotID,
		Points:  points,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	topic := "/robot/" + robotID + "/path"
	return c.Publish(topic, data)
}

// PublishPose публикует текущую позицию и ориентацию робота в отдельный topic (/robot/<id>/pose).
// Используется для визуализации текущего состояния робота в ROS/RViz.
func (c *Client) PublishPose(robotID string, x int, y int, theta float64) error {
	if robotID == "" {
		return errors.New("rosbridge: empty robot id")
	}

	msg := PoseMessage{
		RobotID: robotID,
		X:       x,
		Y:       y,
		Theta:   theta,
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	wrapped, err := json.Marshal(map[string]string{
		"data": string(raw),
	})
	if err != nil {
		return err
	}

	topic := "/robot/" + robotID + "/pose"
	return c.Publish(topic, wrapped)
}

func (c *Client) ensureAdvertised(topic string, msgType string) error {
	c.mu.Lock()
	if existingType, ok := c.advertised[topic]; ok {
		c.mu.Unlock()
		if existingType == msgType {
			return nil
		}
		return errors.New("rosbridge: topic already advertised with different type")
	}
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errors.New("rosbridge: not connected")
	}

	payload := map[string]interface{}{
		"op":    "advertise",
		"topic": topic,
		"type":  msgType,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.advertised[topic] = msgType
	c.mu.Unlock()

	return nil
}

func topicType(topic string) string {
	if strings.Contains(topic, "path_marker") {
		return "visualization_msgs/Marker"
	}
	if strings.Contains(topic, "pose_marker") {
		return "visualization_msgs/Marker"
	}
	if strings.Contains(topic, "/pose") {
		return "std_msgs/String"
	}
	if strings.Contains(topic, "/route") {
		return "std_msgs/String"
	}
	return "std_msgs/String"
}

func (c *Client) PublishMarker(topic string, marker interface{}) error {
	if topic == "" {
		return errors.New("empty topic")
	}

	if err := c.ensureAdvertised(topic, "visualization_msgs/Marker"); err != nil {
		return err
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("rosbridge: not connected")
	}

	payload := map[string]interface{}{
		"op":    "publish",
		"topic": topic,
		"msg":   marker,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return conn.WriteMessage(websocket.TextMessage, data)
}
