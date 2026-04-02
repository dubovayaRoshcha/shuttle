package world

type Point struct {
	X int
	Y int
}

type World struct {
	Width     int
	Height    int
	obstacles map[Point]bool
}

func New(width, height int, obstacles []Point) *World {
	mapObstacles := make(map[Point]bool)
	for _, point := range obstacles {
		mapObstacles[point] = true
	}

	return &World{Width: width, Height: height, obstacles: mapObstacles}
}

func (world *World) Walkable(x, y int) bool {
	if x < 0 || y < 0 || x >= world.Width || y >= world.Height {
		return false
	}

	point := Point{X: x, Y: y}
	if world.obstacles[point] {
		return false
	}

	return true
}

func (world *World) Neighbors(x, y int) []Point {
	neighbors := make([]Point, 0)
	up := Point{X: x, Y: y - 1}
	right := Point{X: x + 1, Y: y}
	down := Point{X: x, Y: y + 1}
	left := Point{X: x - 1, Y: y}
	order := []Point{up, right, down, left}
	for _, point := range order {
		if world.Walkable(point.X, point.Y) {
			neighbors = append(neighbors, point)
		}
	}

	return neighbors
}
