package pathfinding

import (
	"errors"
	"shuttle/internal/world"
)

var (
	ErrPathNotFound = errors.New("pathfinding: path not found")
)

type WalkableFunc func(p world.Point) bool

func abs(num int) int {
	if num < 0 {
		return -num
	}

	return num
}

type node struct {
	point  world.Point
	g      int
	h      int
	f      int
	parent *node
}

func popLowestF(open []*node) (*node, []*node) {
	if len(open) == 0 {
		return nil, open
	}

	fSlice := make([]*node, 0)
	minF := open[0].f
	for _, node := range open[1:] {
		if node.f < minF {
			minF = node.f
		}
	}

	for _, node := range open {
		if node.f == minF {
			fSlice = append(fSlice, node)
		}
	}

	bestNode := fSlice[0]
	bestInd := 0

	if len(fSlice) > 1 {
		for _, node := range fSlice[1:] {
			if node.h < bestNode.h {
				bestNode = node
			}
		}
	}

	for ind := range open {
		if open[ind] == bestNode {
			bestInd = ind
			break
		}
	}

	return bestNode, append(open[:bestInd], open[bestInd+1:]...)
}

func manhattan(a, b world.Point) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func reconstructPath(end *node) []world.Point {
	node := end
	path := make([]world.Point, 0)
	for node != nil {
		path = append(path, node.point)
		node = node.parent
	}

	right := len(path) - 1
	for left := 0; left < right; left++ {
		path[left], path[right] = path[right], path[left]
		right--
	}

	return path
}

func neighborsWithWalkable(current world.Point, w *world.World, walkable WalkableFunc) []world.Point {
	candidates := []world.Point{
		{X: current.X, Y: current.Y - 1},
		{X: current.X + 1, Y: current.Y},
		{X: current.X, Y: current.Y + 1},
		{X: current.X - 1, Y: current.Y},
	}

	result := make([]world.Point, 0, 4)
	for _, p := range candidates {
		if walkable != nil {
			if walkable(p) {
				result = append(result, p)
			}
			continue
		}

		if w.Walkable(p.X, p.Y) {
			result = append(result, p)
		}
	}

	return result
}

func FindPath(start, goal world.Point, w *world.World) ([]world.Point, error) {
	return FindPathWithWalkable(start, goal, w, nil)
}

func FindPathWithWalkable(start, goal world.Point, w *world.World, walkable WalkableFunc) ([]world.Point, error) {
	open := make([]*node, 0)
	closed := make(map[world.Point]bool)
	gScore := make(map[world.Point]int)

	if w == nil {
		return nil, errors.New("expected world")
	}

	canWalk := func(p world.Point) bool {
		if walkable != nil {
			return walkable(p)
		}
		return w.Walkable(p.X, p.Y)
	}

	if start == goal {
		if !canWalk(start) {
			return nil, ErrPathNotFound
		}
		return []world.Point{start}, nil
	}

	if !canWalk(start) || !canWalk(goal) {
		return nil, ErrPathNotFound
	}

	startNode := &node{
		point:  start,
		g:      0,
		h:      manhattan(start, goal),
		parent: nil,
	}
	startNode.f = startNode.g + startNode.h

	open = append(open, startNode)
	gScore[start] = 0

	for len(open) != 0 {
		var current *node
		current, open = popLowestF(open)
		if current == nil {
			break
		}

		if bestG, ok := gScore[current.point]; ok && current.g != bestG {
			continue
		}

		if closed[current.point] {
			continue
		}

		if current.point == goal {
			return reconstructPath(current), nil
		}

		closed[current.point] = true

		neighbors := neighborsWithWalkable(current.point, w, walkable)
		for _, neighbor := range neighbors {
			if closed[neighbor] {
				continue
			}

			tentativeG := current.g + 1

			prevG, ok := gScore[neighbor]
			if !ok || tentativeG < prevG {
				gScore[neighbor] = tentativeG

				n := &node{
					point:  neighbor,
					g:      tentativeG,
					h:      manhattan(neighbor, goal),
					parent: current,
				}
				n.f = n.g + n.h

				open = append(open, n)
			}
		}
	}

	return nil, ErrPathNotFound
}
