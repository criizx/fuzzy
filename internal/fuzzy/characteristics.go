package fuzzy

// Numerical characteristics use trapezoidal integration over alpha levels.

// Area integrates membership using alpha-cut widths: integral (Hi(alpha)-Lo(alpha)) d alpha.
func Area(n Number, cfg Config) float64 {
	cfg = cfg.WithDefaults()
	levels := AlphaLevels(n.Height(), cfg.AlphaLevels)
	return trapz(levels, func(a float64) float64 { return CutClamped(n, a).Width() })
}

// Centroid returns the membership-weighted center of gravity.
// A zero-width value falls back to the midpoint of its support.
func Centroid(n Number, cfg Config) float64 {
	cfg = cfg.WithDefaults()
	levels := AlphaLevels(n.Height(), cfg.AlphaLevels)
	area := trapz(levels, func(a float64) float64 { return CutClamped(n, a).Width() })
	if area <= 0 {
		return n.Support().Mid()
	}
	moment := trapz(levels, func(a float64) float64 {
		c := CutClamped(n, a)
		return (c.Hi*c.Hi - c.Lo*c.Lo) / 2
	})
	return moment / area
}

// Mean returns the Carlsson-Fuller possibilistic mean,
// integral alpha*(Lo(alpha)+Hi(alpha)) d alpha / height², not E[X].
func Mean(n Number, cfg Config) float64 {
	cfg = cfg.WithDefaults()
	h := n.Height()
	levels := AlphaLevels(h, cfg.AlphaLevels)
	return trapz(levels, func(a float64) float64 {
		c := CutClamped(n, a)
		return a * (c.Lo + c.Hi)
	}) / (h * h)
}

// Variance returns the Carlsson-Fuller possibilistic variance,
// integral alpha*(Hi(alpha)-Lo(alpha))² d alpha / (2*height²), not Var[X].
func Variance(n Number, cfg Config) float64 {
	cfg = cfg.WithDefaults()
	h := n.Height()
	levels := AlphaLevels(h, cfg.AlphaLevels)
	return trapz(levels, func(a float64) float64 {
		w := CutClamped(n, a).Width()
		return a * w * w
	}) / (2 * h * h)
}

// Spread returns the width of the working support.
func Spread(n Number) float64 { return n.Support().Width() }
