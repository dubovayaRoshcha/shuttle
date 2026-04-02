package config

import "testing"

func TestLoadInit(t *testing.T) {
	path := "../../configs/config.yaml"

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	InitLogger(cfg.Logging.Level)

	Debug("debug message")
	Info("server started")
	Error("something went wrong")

	t.Log("логгер работает")
}
