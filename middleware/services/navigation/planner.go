package navigation

import (
	"container/heap"
	"math"
)

// Point is a 2D world coordinate in metres.
type Point struct {
	X float64
	Y float64
}

// Distance returns the Euclidean distance between two points.
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Planner is the interface for path planning algorithms.
type Planner interface {
	Plan(from, to Point) []Point
}

// SimpleLinearPlanner plans a straight-line path from current to goal.
type SimpleLinearPlanner struct{}

// Plan returns a direct straight-line path with intermediate waypoints.
func (p *SimpleLinearPlanner) Plan(from, to Point) []Point {
	const steps = 10
	path := make([]Point, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		path[i] = Point{
			X: from.X + (to.X-from.X)*t,
			Y: from.Y + (to.Y-from.Y)*t,
		}
	}
	return path
}

// ---------------------------------------------------------------------------
// A* planner
// ---------------------------------------------------------------------------

// astarNode is an internal node used by the A* priority queue.
type astarNode struct {
	x, y   int
	g, f   float64
	parent *astarNode
	index  int
}

type astarHeap []*astarNode

func (h astarHeap) Len() int           { return len(h) }
func (h astarHeap) Less(i, j int) bool { return h[i].f < h[j].f }
func (h astarHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *astarHeap) Push(x any) {
	n := x.(*astarNode)
	n.index = len(*h)
	*h = append(*h, n)
}
func (h *astarHeap) Pop() any {
	old := *h
	n := old[len(old)-1]
	old[len(old)-1] = nil
	*h = old[:len(old)-1]
	n.index = -1
	return n
}

// AStarPlanner implements A* path planning on a 2D occupancy grid.
// Coordinates (0, 0) map to cell (0, 0); each cell is resolution × resolution metres.
type AStarPlanner struct {
	grid       [][]bool // grid[y][x] == true means obstacle
	resolution float64
	width      int
	height     int
}

// NewAStarPlanner creates an AStarPlanner with a clear width×height grid.
func NewAStarPlanner(width, height int, resolution float64) *AStarPlanner {
	grid := make([][]bool, height)
	for i := range grid {
		grid[i] = make([]bool, width)
	}
	return &AStarPlanner{grid: grid, resolution: resolution, width: width, height: height}
}

// SetObstacle marks the grid cell at world coordinate (x, y) as impassable.
func (p *AStarPlanner) SetObstacle(x, y float64) {
	cx, cy := p.toCell(x, y)
	if p.inBounds(cx, cy) {
		p.grid[cy][cx] = true
	}
}

// ClearObstacle removes the obstacle at the grid cell containing world coordinate (x, y).
func (p *AStarPlanner) ClearObstacle(x, y float64) {
	cx, cy := p.toCell(x, y)
	if p.inBounds(cx, cy) {
		p.grid[cy][cx] = false
	}
}

func (p *AStarPlanner) toCell(x, y float64) (int, int) {
	return int(x / p.resolution), int(y / p.resolution)
}

func (p *AStarPlanner) toWorld(cx, cy int) Point {
	return Point{
		X: (float64(cx) + 0.5) * p.resolution,
		Y: (float64(cy) + 0.5) * p.resolution,
	}
}

func (p *AStarPlanner) inBounds(cx, cy int) bool {
	return cx >= 0 && cx < p.width && cy >= 0 && cy < p.height
}

// Plan runs A* from 'from' to 'to' and returns the waypoint path.
// Returns nil if no path exists or if either endpoint is outside the grid.
func (p *AStarPlanner) Plan(from, to Point) []Point {
	sx, sy := p.toCell(from.X, from.Y)
	gx, gy := p.toCell(to.X, to.Y)

	if !p.inBounds(sx, sy) || !p.inBounds(gx, gy) {
		return nil
	}
	if p.grid[sy][sx] || p.grid[gy][gx] {
		return nil
	}
	if sx == gx && sy == gy {
		return []Point{to}
	}

	h := func(x, y int) float64 {
		dx := float64(x - gx)
		dy := float64(y - gy)
		return math.Sqrt(dx*dx + dy*dy)
	}

	type cellKey struct{ x, y int }

	gScore := map[cellKey]float64{{sx, sy}: 0}
	open := &astarHeap{}
	heap.Push(open, &astarNode{x: sx, y: sy, g: 0, f: h(sx, sy)})

	// 8-directional movement: cardinal + diagonal
	dirs := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	costs := [8]float64{1, 1, 1, 1, math.Sqrt2, math.Sqrt2, math.Sqrt2, math.Sqrt2}

	closed := map[cellKey]bool{}

	for open.Len() > 0 {
		cur := heap.Pop(open).(*astarNode)
		k := cellKey{cur.x, cur.y}
		if closed[k] {
			continue
		}
		closed[k] = true

		if cur.x == gx && cur.y == gy {
			return p.reconstructPath(cur)
		}

		for i, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]
			nk := cellKey{nx, ny}
			if !p.inBounds(nx, ny) || p.grid[ny][nx] || closed[nk] {
				continue
			}
			ng := cur.g + costs[i]
			if best, ok := gScore[nk]; ok && ng >= best {
				continue
			}
			gScore[nk] = ng
			heap.Push(open, &astarNode{
				x: nx, y: ny,
				g: ng, f: ng + h(nx, ny),
				parent: cur,
			})
		}
	}
	return nil
}

func (p *AStarPlanner) reconstructPath(n *astarNode) []Point {
	var rev []*astarNode
	for n != nil {
		rev = append(rev, n)
		n = n.parent
	}
	path := make([]Point, len(rev))
	for i, node := range rev {
		path[len(rev)-1-i] = p.toWorld(node.x, node.y)
	}
	return path
}
