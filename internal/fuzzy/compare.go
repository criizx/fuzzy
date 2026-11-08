package fuzzy

import (
	"math"
	"sort"
)

const bisectIter = 60

// PossibilityGE returns sup_{x>=y} min(membership(a,x),membership(b,y)).
// Bisection finds the largest alpha for which the upper cut of a reaches the lower cut of b.
func PossibilityGE(a, b Number) float64 {
	h := math.Min(a.Height(), b.Height())
	holds := func(alpha float64) bool {
		ca, okA := a.AlphaCut(alpha)
		cb, okB := b.AlphaCut(alpha)
		return okA && okB && ca.Hi >= cb.Lo
	}
	if !(h > 0) || !holds(0) {
		return 0
	}
	if holds(h) {
		return h
	}
	lo, hi := 0.0, h // holds(lo) = true, holds(hi) = false; the predicate is monotone in alpha
	for i := 0; i < bisectIter; i++ {
		mid := (lo + hi) / 2
		if holds(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

// PossibilityGT — sup_{x>y} min(μa(x), μb(y)). A strict inequality
// can hold below the height even when the modal points coincide. Return
// its supremum (not a maximum, which need not exist).
func PossibilityGT(a, b Number) float64 {
	h := math.Min(a.Height(), b.Height())
	holds := func(alpha float64) bool {
		ca, okA := a.AlphaCut(alpha)
		cb, okB := b.AlphaCut(alpha)
		return okA && okB && ca.Hi > cb.Lo
	}
	if !(h > 0) || !holds(0) {
		return 0
	}
	if holds(h) {
		return h
	}
	lo, hi := 0.0, h
	for i := 0; i < bisectIter; i++ {
		mid := (lo + hi) / 2
		if holds(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

// NecessityGE — Nec(a ≥ b) = 1 - Pos(b > a), for normal numbers.
// The strict complement matters for crisp numbers and touching supports.
func NecessityGE(a, b Number) float64 {
	return 1 - PossibilityGT(b, a)
}

// Result describes a three-way fuzzy comparison.
type Result int

const (
	Less    Result = -1
	Equal   Result = 0
	Greater Result = 1
)

func (r Result) String() string {
	switch r {
	case Less:
		return "<"
	case Greater:
		return ">"
	default:
		return "="
	}
}

// Strategy defines an interchangeable ordering rule for fuzzy values.
type Strategy interface {
	Compare(a, b Number) Result
	Name() string
}

func sign(d float64) Result {
	switch {
	case d > 0:
		return Greater
	case d < 0:
		return Less
	default:
		return Equal
	}
}

// CentroidStrategy compares membership-weighted centroids.
type CentroidStrategy struct {
	Cfg Config
}

func (s CentroidStrategy) Name() string { return "centroid" }

func (s CentroidStrategy) Compare(a, b Number) Result {
	ca := Centroid(a, s.Cfg)
	cb := Centroid(b, s.Cfg)
	return sign(ca - cb)
}

// PossibilityStrategy compares Pos(a>=b)-Pos(b>=a).
// A zero difference is treated as equality.
type PossibilityStrategy struct{}

func (s PossibilityStrategy) Name() string { return "possibility" }

func (s PossibilityStrategy) Compare(a, b Number) Result {
	return sign(PossibilityGE(a, b) - PossibilityGE(b, a))
}

// IntervalStrategy compares alpha-cut midpoints at the selected level.
type IntervalStrategy struct {
	Alpha float64
}

func (s IntervalStrategy) Name() string { return "alpha-cut" }

func (s IntervalStrategy) Compare(a, b Number) Result {
	ca := CutClamped(a, s.Alpha)
	cb := CutClamped(b, s.Alpha)
	return sign(ca.Mid() - cb.Mid())
}

// Sort orders values in ascending order, preserving ties.
func Sort(items []Number, s Strategy) {
	sort.SliceStable(items, func(i, j int) bool { return s.Compare(items[i], items[j]) == Less })
}

// Max returns the greatest value, choosing the first in a tie.
func Max(items []Number, s Strategy) (Number, error) {
	if len(items) == 0 {
		return nil, ErrEmpty
	}
	best := items[0]
	for _, it := range items[1:] {
		if s.Compare(it, best) == Greater {
			best = it
		}
	}
	return best, nil
}

// Min returns the smallest value, choosing the first in a tie.
func Min(items []Number, s Strategy) (Number, error) {
	if len(items) == 0 {
		return nil, ErrEmpty
	}
	best := items[0]
	for _, it := range items[1:] {
		if s.Compare(it, best) == Less {
			best = it
		}
	}
	return best, nil
}
