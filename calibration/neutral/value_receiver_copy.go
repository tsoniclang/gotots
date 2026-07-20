// Neutral semantic control: mutation of a value receiver affects only
// its copy — the caller's value is unchanged.
package neutral

type Point struct {
	X int
	Y int
}

func (p Point) Move(dx int, dy int) int {
	p.X += dx
	p.Y += dy
	return p.X + p.Y
}
