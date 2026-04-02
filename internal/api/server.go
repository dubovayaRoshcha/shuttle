package api

import (
	"net/http"
	"shuttle/internal/dispatcher"
	"shuttle/internal/storage"
)

type Server struct {
	storage    storage.Storage
	dispatcher *dispatcher.Dispatcher
}

// конструктор
func New(storage storage.Storage, dispatcher *dispatcher.Dispatcher) *Server {
	return &Server{storage: storage, dispatcher: dispatcher}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	h := NewHandlers(s.dispatcher)

	mux.HandleFunc("/tasks", h.Tasks)              // GET POST /tasks
	mux.HandleFunc("/tasks/", h.TaskByID)          // GET /tasks/ (префикс под /tasks/{id})
	mux.HandleFunc("/dispatch/run", h.DispatchRun) // POST /dispatch/run

	return mux
}
