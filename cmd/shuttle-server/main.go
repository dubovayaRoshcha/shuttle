// cmd/shuttle-server/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"shuttle/internal/api"
	"shuttle/internal/config"
	"shuttle/internal/dispatcher"
	"shuttle/internal/replanner"
	"shuttle/internal/reservations"
	"shuttle/internal/robots"
	"shuttle/internal/rosbridge"
	"shuttle/internal/storage"
	"shuttle/internal/tasks"
	"shuttle/internal/telemetry"
	"shuttle/internal/world"
)

const debugROS = true

func main() {
	// 1) Конфиг + логгер
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}
	config.InitLogger(cfg.Logging.Level)
	config.Info("starting shuttle-server")

	// 2) Хранилище
	st := storage.NewMemory()
	defer st.Close()

	// 3) Контекст с отменой по сигналам
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	// 4) Пока тут задаются параметры для отладки
	queue := tasks.NewQueue()
	manager := robots.NewManager()
	manager.Upsert(robots.Robot{
		ID: cfg.App.DefaultRobotID,
		X:  0,
		Y:  0,
	})
	w := world.New(10, 10, []world.Point{})
	res := reservations.NewManager()
	rep := replanner.NewService(w, res)

	// 5) ROS bridge + Telemetry
	rb := rosbridge.New(rosbridge.Options{URL: cfg.Rosbridge.URL}) // см. поля в Config

	if err := rb.Connect(ctx); err != nil {
		config.Error("rosbridge connect failed: " + err.Error())
		stop()
		return
	}

	tm := telemetry.New(rb, st, manager, cfg.App.DefaultRobotID) // DefaultRobotID из config.yaml

	if debugROS {
		err = rb.Subscribe("/robot/"+cfg.App.DefaultRobotID+"/route", func(topic string, msg json.RawMessage) {
			config.Info("route published: " + string(msg))
		}) // это временно для проверки
		if err != nil {
			config.Error("route subscribe failed: " + err.Error())
		}
	}

	if err := tm.Start(ctx); err != nil {
		config.Error("telemetry start failed: " + err.Error())
		stop()
		return
	}

	/*go func() {
		time.Sleep(20 * time.Second)
		if err := tm.PublishCurrentPoseMarker(); err != nil {
			config.Error("republish pose marker failed: " + err.Error())
		}
	}()*/

	/*if debugROS {
		go func() {
			time.Sleep(time.Second) // ждём секунду, чтобы telemetry успела подписаться
			rb.InjectPublish("/robot/"+cfg.App.DefaultRobotID+"/odom", []byte(`{"pose":{"pose":{"position":{"x":2.0,"y":3.0}}}}`))
		}()
	}*/

	disp := dispatcher.New(dispatcher.Options{
		Queue:        queue,
		Manager:      manager,
		World:        w,
		Reservations: res,
		Replanner:    rep,
		ROS:          rb,
		Publisher:    tm,
	})

	// 5) HTTP API

	srv := api.New(st, disp)
	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)

	rootMux := http.NewServeMux()

	rootMux.HandleFunc("/debug/republish-pose-marker", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := tm.PublishCurrentPoseMarker(); err != nil {
			config.Error("republish pose marker failed: " + err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		config.Info("republish pose marker requested via debug endpoint")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	rootMux.Handle("/", srv.Router())

	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      rootMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		config.Info("http listen on " + addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Error("http server error: " + err.Error())
			stop()
		}
	}()

	// 6) Ожидание сигнала и аккуратное завершение
	<-ctx.Done()
	config.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		config.Error("http shutdown error: " + err.Error())
	}

	if err := rb.Close(); err != nil {
		config.Error("rosbridge close error: " + err.Error())
	}
	config.Info("bye")
}
