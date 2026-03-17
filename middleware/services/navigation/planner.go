package navigation

import "math"

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
// Phase 1: SimpleLinearPlanner
// Phase 2: AStarPlanner
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
