package config

import "testing"

func TestLoggerBasic(t *testing.T) {
	InitLogger("info")

	Debug("debug message")
	Info("info message")
	Error("error message")

	t.Log("логгер работает")
}

func TestLoggerWithLoad(t *testing.T) {
	InitLogger("info")

	Debug("debug message")
	Info("info message")
	Error("error message")

	t.Log("логгер работает")
}
