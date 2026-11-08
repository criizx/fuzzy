package fuzzy

// Number describes a convex fuzzy value with membership in [0,1].
// Algorithms depend on this interface rather than a particular membership model.
type Number interface {
	// Membership returns the membership grade at x.
	Membership(x float64) float64
	// Support returns the closure of the set with positive membership.
	Support() Interval
	// AlphaCut returns the interval with membership at least alpha.
	// AlphaCut(0) denotes the support; invalid levels return ok=false.
	AlphaCut(alpha float64) (cut Interval, ok bool)
	// Height returns the supremum of the membership function.
	Height() float64
	// Kind returns a short model identifier for persistence and presentation.
	Kind() string
}

// Point stores a coordinate pair on a membership outline.
type Point struct{ X, Y float64 }

// AlphaLevels returns n+1 levels from zero to h. The final level is exactly h.
func AlphaLevels(h float64, n int) []float64 {
	if n < 1 {
		n = 1
	}
	levels := make([]float64, n+1)
	for k := 0; k < n; k++ {
		levels[k] = h * float64(k) / float64(n)
	}
	levels[n] = h
	return levels
}

// CutClamped returns an alpha cut, clamping levels above the height to the height.
// If the requested cut is invalid, it returns the support.
func CutClamped(n Number, alpha float64) Interval {
	if h := n.Height(); alpha > h {
		alpha = h
	}
	if c, ok := n.AlphaCut(alpha); ok {
		return c
	}
	return n.Support()
}
