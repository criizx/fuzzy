package fuzzy

import (
	"math"
)

// Distance measures dissimilarity between fuzzy values independently of clustering.
type Distance interface {
	Between(a, b Number) float64
	Name() string
}

// pairIntegral averages an interval functional over alpha in [0,min(height(a),height(b))].
func pairIntegral(a, b Number, cfg Config, f func(ca, cb Interval) float64) float64 {
	cfg = cfg.WithDefaults()
	h := math.Min(a.Height(), b.Height())
	if !(h > 0) {
		return math.NaN()
	}
	levels := AlphaLevels(h, cfg.AlphaLevels)
	return trapz(levels, func(alpha float64) float64 {
		return f(CutClamped(a, alpha), CutClamped(b, alpha))
	}) / h
}

// CutsL1Distance averages (abs(delta Lo)+abs(delta Hi))/2 over alpha.
type CutsL1Distance struct{ Cfg Config }

func (d CutsL1Distance) Name() string { return "cuts-L1" }
func (d CutsL1Distance) Between(a, b Number) float64 {
	return pairIntegral(a, b, d.Cfg, func(x, y Interval) float64 {
		return (math.Abs(x.Lo-y.Lo) + math.Abs(x.Hi-y.Hi)) / 2
	})
}

// CutsL2Distance returns the root-mean-square displacement of alpha-cut bounds.
type CutsL2Distance struct{ Cfg Config }

func (d CutsL2Distance) Name() string { return "cuts-L2" }
func (d CutsL2Distance) Between(a, b Number) float64 {
	v := pairIntegral(a, b, d.Cfg, func(x, y Interval) float64 {
		dl, dh := x.Lo-y.Lo, x.Hi-y.Hi
		return (dl*dl + dh*dh) / 2
	})
	return math.Sqrt(v)
}

// HausdorffDistance returns the maximum endpoint displacement over sampled alpha levels.
type HausdorffDistance struct{ Cfg Config }

func (d HausdorffDistance) Name() string { return "hausdorff" }
func (d HausdorffDistance) Between(a, b Number) float64 {
	cfg := d.Cfg.WithDefaults()
	h := math.Min(a.Height(), b.Height())
	best := 0.0
	for _, alpha := range AlphaLevels(h, cfg.AlphaLevels) {
		x, y := CutClamped(a, alpha), CutClamped(b, alpha)
		best = math.Max(best, math.Max(math.Abs(x.Lo-y.Lo), math.Abs(x.Hi-y.Hi)))
	}
	return best
}

// AlphaCutDistance compares cut endpoints at one alpha level using their mean absolute displacement.
type AlphaCutDistance struct{ Alpha float64 }

func (d AlphaCutDistance) Name() string { return "cut-L1" }
func (d AlphaCutDistance) Between(a, b Number) float64 {
	x, y := CutClamped(a, d.Alpha), CutClamped(b, d.Alpha)
	return (math.Abs(x.Lo-y.Lo) + math.Abs(x.Hi-y.Hi)) / 2
}

// CentroidDistance returns the absolute difference between fuzzy centroids.
type CentroidDistance struct{ Cfg Config }

func (d CentroidDistance) Name() string { return "centroid" }
func (d CentroidDistance) Between(a, b Number) float64 {
	return math.Abs(Centroid(a, d.Cfg) - Centroid(b, d.Cfg))
}

// HammingDistance integrates abs(membership(a)-membership(b)) on an x grid
// using the midpoint rule. Discontinuities and vertical edges can reduce accuracy.
type HammingDistance struct{ Cfg Config }

func (d HammingDistance) Name() string { return "hamming" }
func (d HammingDistance) Between(a, b Number) float64 {
	cfg := d.Cfg.WithDefaults()
	hull := a.Support().Hull(b.Support())
	w := hull.Width()
	if w <= 0 {
		return 0
	}
	step := w / float64(cfg.GridPoints)
	sum := 0.0
	for i := 0; i < cfg.GridPoints; i++ {
		x := hull.Lo + (float64(i)+0.5)*step
		sum += math.Abs(a.Membership(x) - b.Membership(x))
	}
	return sum * step
}

// DistanceMatrix constructs a symmetric matrix of pairwise distances with a zero diagonal.
func DistanceMatrix(items []Number, d Distance) [][]float64 {
	n := len(items)
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			v := d.Between(items[i], items[j])
			m[i][j], m[j][i] = v, v
		}
	}
	return m
}
