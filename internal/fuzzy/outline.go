package fuzzy

// Outline returns a membership outline: left cut boundaries from bottom to top,
// followed by right boundaries from top to bottom, omitting repeated points.
func Outline(n Number, steps int) []Point {
	levels := AlphaLevels(n.Height(), steps)
	left := make([]Point, 0, len(levels))
	right := make([]Point, 0, len(levels))
	for _, a := range levels {
		c := CutClamped(n, a)
		left = append(left, Point{X: c.Lo, Y: a})
		right = append(right, Point{X: c.Hi, Y: a})
	}
	out := make([]Point, 0, 2*len(levels))
	push := func(p Point) {
		if len(out) == 0 || out[len(out)-1] != p {
			out = append(out, p)
		}
	}
	for _, p := range left {
		push(p)
	}
	for i := len(right) - 1; i >= 0; i-- {
		push(right[i])
	}
	return out
}
