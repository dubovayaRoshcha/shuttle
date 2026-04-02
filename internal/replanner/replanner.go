package replanner

import (
	"errors"
	"shuttle/internal/pathfinding"
	"shuttle/internal/reservations"
	"shuttle/internal/world"
)

type Service struct {
	world        *world.World
	reservations *reservations.Manager
}

func NewService(w *world.World, r *reservations.Manager) *Service {
	return &Service{
		world:        w,
		reservations: r,
	}
}

func (s *Service) Replan(start, goal world.Point, owner string) ([]world.Point, error) {
	path, err := pathfinding.FindPathWithWalkable(start, goal, s.world, func(p world.Point) bool {
		if !s.world.Walkable(p.X, p.Y) {
			return false
		}

		if !s.reservations.IsReserved(p) {
			return true
		}

		ownerOfCell, _ := s.reservations.Owner(p)
		return ownerOfCell == owner
	})
	if err != nil {
		return nil, err
	}

	if len(path) == 0 {
		return nil, errors.New("no path found")
	}

	return path, nil
}
