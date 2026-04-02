// internal/api/handlers.go
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"shuttle/internal/config"
	"shuttle/internal/dispatcher"
	"shuttle/internal/tasks"
	"shuttle/internal/world"
)

type pointDTO struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Отдельная струткра для запроса про контракту
type createTaskRequest struct {
	Type string   `json:"type"`
	From pointDTO `json:"from"`
	To   pointDTO `json:"to"`
}

// Handlers инкапсулирует зависимости HTTP-слоя.
type Handlers struct {
	dispatcher *dispatcher.Dispatcher
}

// NewHandlers создаёт набор хендлеров.
func NewHandlers(dispatcher *dispatcher.Dispatcher) *Handlers {
	return &Handlers{dispatcher: dispatcher}
}

func (h *Handlers) Tasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		listTasks := h.dispatcher.Queue.ListTasks()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) //200
		json.NewEncoder(w).Encode(listTasks)
		return
	}

	if r.Method == http.MethodPost {
		var ctr createTaskRequest
		err := json.NewDecoder(r.Body).Decode(&ctr)
		if err != nil {
			config.Error("Error with JSON " + err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) //400
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		task := tasks.Task{
			Type: ctr.Type,
			From: world.Point{X: ctr.From.X, Y: ctr.From.Y},
			To:   world.Point{X: ctr.To.X, Y: ctr.To.Y},
		}

		err = h.dispatcher.Queue.AddTask(&task)
		if err != nil {
			config.Error("Adding task failed " + err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError) //500
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) //201
		json.NewEncoder(w).Encode(task)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed) //405
}

func (h *Handlers) TaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id, _ := strings.CutPrefix(r.URL.Path, "/tasks/")
		if id == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound) //404
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		task, ok := h.dispatcher.Queue.GetTask(id)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound) //404
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) //200
		json.NewEncoder(w).Encode(task)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed) //405
}

func (h *Handlers) DispatchRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := h.dispatcher.RunOnce(r.Context())
	if err != nil {
		config.Error("Error with using task or robot " + err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

/*
// Health — GET /healthz -> 200 {"status":"ok"}.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed) //405
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// TelemetryRobots — GET /telemetry/robots -> JSON-массив роботов или 500.
func (h *Handlers) TelemetryRobots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	robots, err := h.store.ListRobots(r.Context())
	if err != nil {
		// Логируем ошибку через наш пакет config.
		config.Error("list robots failed: " + err.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) //500
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(robots)
}

*/
