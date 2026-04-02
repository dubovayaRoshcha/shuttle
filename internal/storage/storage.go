package storage

import (
	"context"

	"shuttle/internal/robots"
)

type Storage interface {
	UpsertRobot(ctx context.Context, robot robots.Robot) error // обновляет данные или добавляет
	GetRobot(ctx context.Context, id string) (robots.Robot, bool, error)
	ListRobots(ctx context.Context) ([]robots.Robot, error)
	Close() error
}
