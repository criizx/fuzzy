package fuzzy

import (
	"fmt"
	"math"
	"sort"
)

// Cuts stores nested alpha-cut intervals with linearly interpolated bounds.
// Piecewise-linear boundaries are represented exactly; nonlinear ones are approximated
// with accuracy determined by the number of alpha levels.
type Cuts struct {
	levels []float64  // strictly increasing; the final level is the height
	cuts   []Interval // nested: cuts[i+1] is contained in cuts[i]
}

var _ Number = Cuts{}

// NewCuts validates and copies strictly increasing levels in [0,1] and nested intervals.
// The final level must be positive. If the first level is not zero, its cut is the support.
func NewCuts(levels []float64, cuts []Interval) (Cuts, error) {
	n := len(levels)
	if n == 0 || n != len(cuts) {
		return Cuts{}, fmt.Errorf("%w: уровни и срезы должны быть непустыми и одной длины", ErrInvalidParams)
	}
	ls := append([]float64(nil), levels...)
	cs := append([]Interval(nil), cuts...)
	for i := range ls {
		if math.IsNaN(ls[i]) || ls[i] < 0 || ls[i] > 1+1e-12 {
			return Cuts{}, fmt.Errorf("%w: уровень %g вне [0,1]", ErrInvalidParams, ls[i])
		}
		if i > 0 && ls[i] <= ls[i-1] {
			return Cuts{}, fmt.Errorf("%w: уровни должны строго возрастать", ErrInvalidParams)
		}
		if !cs[i].Valid() {
			return Cuts{}, fmt.Errorf("%w: некорректный срез %v на уровне %g", ErrInvalidParams, cs[i], ls[i])
		}
		if i > 0 {
			p := cs[i-1]
			tol := 1e-9 * (1 + math.Max(math.Abs(p.Lo), math.Abs(p.Hi)))
			if cs[i].Lo < p.Lo-tol || cs[i].Hi > p.Hi+tol {
				return Cuts{}, fmt.Errorf("%w: срезы не вложены на уровне %g", ErrInvalidParams, ls[i])
			}
			// Clamp roundoff within the nesting tolerance.
			cs[i].Lo = math.Max(cs[i].Lo, p.Lo)
			cs[i].Hi = math.Min(cs[i].Hi, p.Hi)
			if cs[i].Lo > cs[i].Hi {
				m := (cs[i].Lo + cs[i].Hi) / 2
				cs[i] = Interval{Lo: m, Hi: m}
			}
		}
	}
	if ls[n-1] <= 0 {
		return Cuts{}, fmt.Errorf("%w: высота должна быть положительной", ErrInvalidParams)
	}
	return Cuts{levels: ls, cuts: cs}, nil
}

// FromNumber samples n+1 equally spaced alpha cuts between zero and the height.
func FromNumber(num Number, n int) (Cuts, error) {
	h := num.Height()
	if !(h > 0) {
		return Cuts{}, ErrEmpty
	}
	levels := AlphaLevels(h, n)
	cuts := make([]Interval, len(levels))
	for i, a := range levels {
		c, ok := num.AlphaCut(a)
		if !ok {
			return Cuts{}, ErrEmpty
		}
		cuts[i] = c
	}
	return NewCuts(levels, cuts)
}

func (c Cuts) Len() int { return len(c.levels) }

// Levels returns a copy of the sampled membership levels.
func (c Cuts) Levels() []float64 { return append([]float64(nil), c.levels...) }

// Intervals returns a copy of the sampled intervals.
func (c Cuts) Intervals() []Interval { return append([]Interval(nil), c.cuts...) }

func (c Cuts) Height() float64   { return c.levels[len(c.levels)-1] }
func (c Cuts) Support() Interval { return c.cuts[0] }
func (c Cuts) Kind() string      { return "cuts" }

// AlphaCut linearly interpolates cut bounds between neighboring sampled levels.
func (c Cuts) AlphaCut(alpha float64) (Interval, bool) {
	if math.IsNaN(alpha) || alpha < 0 || alpha > c.Height()+1e-12 {
		return Interval{}, false
	}
	if alpha <= c.levels[0] {
		return c.cuts[0], true
	}
	n := len(c.levels)
	i := sort.SearchFloat64s(c.levels, alpha) // first levels[i] >= alpha
	if i >= n {
		return c.cuts[n-1], true
	}
	if c.levels[i] == alpha {
		return c.cuts[i], true
	}
	t := (alpha - c.levels[i-1]) / (c.levels[i] - c.levels[i-1])
	lo := c.cuts[i-1].Lo + t*(c.cuts[i].Lo-c.cuts[i-1].Lo)
	hi := c.cuts[i-1].Hi + t*(c.cuts[i].Hi-c.cuts[i-1].Hi)
	return Interval{Lo: lo, Hi: hi}, true
}

// Membership reconstructs the grade by inverting both cut-boundary branches
// and taking their minimum.
func (c Cuts) Membership(x float64) float64 {
	if math.IsNaN(x) || !c.cuts[0].Contains(x) {
		return 0
	}
	n := len(c.levels)

	// The left boundary is nondecreasing in alpha; find the last Lo <= x.
	i := sort.Search(n, func(k int) bool { return c.cuts[k].Lo > x }) - 1
	muL := c.levels[i]
	if i < n-1 {
		d := c.cuts[i+1].Lo - c.cuts[i].Lo
		muL += (x - c.cuts[i].Lo) / d * (c.levels[i+1] - c.levels[i])
	}

	// The right boundary is nonincreasing in alpha; find the last Hi >= x.
	j := sort.Search(n, func(k int) bool { return c.cuts[k].Hi < x }) - 1
	muR := c.levels[j]
	if j < n-1 {
		d := c.cuts[j].Hi - c.cuts[j+1].Hi
		muR += (c.cuts[j].Hi - x) / d * (c.levels[j+1] - c.levels[j])
	}
	return math.Min(muL, muR)
}

// Normalized returns a copy with levels divided by the original height.
func (c Cuts) Normalized() Cuts {
	h := c.Height()
	ls := make([]float64, len(c.levels))
	for i, l := range c.levels {
		ls[i] = l / h
	}
	ls[len(ls)-1] = 1
	return Cuts{levels: ls, cuts: append([]Interval(nil), c.cuts...)}
}

func (c Cuts) String() string {
	s := c.Support()
	return fmt.Sprintf("Cuts{support=[%.4g; %.4g], height=%.3g, levels=%d}", s.Lo, s.Hi, c.Height(), len(c.levels))
}
