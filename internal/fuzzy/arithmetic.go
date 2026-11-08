package fuzzy

import (
	"fmt"
	"math"
)

// Operation applies a binary interval operation to two alpha cuts.
type Operation func(a, b Interval) (Interval, error)

// UnaryOperation transforms one alpha-cut interval.
type UnaryOperation func(a Interval) Interval

// Built-in interval operations can be passed to Apply.
func AddOp(a, b Interval) (Interval, error) { return a.Add(b), nil }
func SubOp(a, b Interval) (Interval, error) { return a.Sub(b), nil }
func MulOp(a, b Interval) (Interval, error) { return a.Mul(b), nil }
func DivOp(a, b Interval) (Interval, error) { return a.Div(b) }

// MinOp returns the componentwise minimum of the interval bounds.
func MinOp(a, b Interval) (Interval, error) {
	return Interval{Lo: math.Min(a.Lo, b.Lo), Hi: math.Min(a.Hi, b.Hi)}, nil
}
func MaxOp(a, b Interval) (Interval, error) {
	return Interval{Lo: math.Max(a.Lo, b.Lo), Hi: math.Max(a.Hi, b.Hi)}, nil
}

// Apply implements the extension principle on levels 0, h/N, ..., h, where
// h is the smaller operand height and N is cfg.AlphaLevels. Errors identify the alpha level.
func Apply(a, b Number, op Operation, cfg Config) (Cuts, error) {
	cfg = cfg.WithDefaults()
	h := math.Min(a.Height(), b.Height())
	if !(h > 0) {
		return Cuts{}, ErrEmpty
	}
	levels := AlphaLevels(h, cfg.AlphaLevels)
	cuts := make([]Interval, len(levels))
	for i, alpha := range levels {
		ca, okA := a.AlphaCut(alpha)
		cb, okB := b.AlphaCut(alpha)
		if !okA || !okB {
			return Cuts{}, ErrEmpty
		}
		r, err := op(ca, cb)
		if err != nil {
			return Cuts{}, fmt.Errorf("α=%.4g: %w", alpha, err)
		}
		cuts[i] = r
	}
	return NewCuts(levels, cuts)
}

// ApplyUnary applies a unary interval operation at each sampled alpha level.
func ApplyUnary(a Number, f UnaryOperation, cfg Config) (Cuts, error) {
	cfg = cfg.WithDefaults()
	h := a.Height()
	if !(h > 0) {
		return Cuts{}, ErrEmpty
	}
	levels := AlphaLevels(h, cfg.AlphaLevels)
	cuts := make([]Interval, len(levels))
	for i, alpha := range levels {
		c, ok := a.AlphaCut(alpha)
		if !ok {
			return Cuts{}, ErrEmpty
		}
		cuts[i] = f(c)
	}
	return NewCuts(levels, cuts)
}

func Add(a, b Number, cfg Config) (Cuts, error) { return Apply(a, b, AddOp, cfg) }
func Sub(a, b Number, cfg Config) (Cuts, error) { return Apply(a, b, SubOp, cfg) }
func Mul(a, b Number, cfg Config) (Cuts, error) { return Apply(a, b, MulOp, cfg) }
func Div(a, b Number, cfg Config) (Cuts, error) { return Apply(a, b, DivOp, cfg) }

// Neg returns the additive reflection of a fuzzy value.
func Neg(a Number, cfg Config) (Cuts, error) {
	return ApplyUnary(a, func(c Interval) Interval { return c.Neg() }, cfg)
}

// Scale multiplies by a crisp scalar, including negative values and zero.
func Scale(a Number, k float64, cfg Config) (Cuts, error) {
	return ApplyUnary(a, func(c Interval) Interval { return c.Scale(k) }, cfg)
}

// Calculator reuses one numerical configuration across chained operations.
type Calculator struct{ Cfg Config }

func (c Calculator) Add(a, b Number) (Cuts, error) { return Add(a, b, c.Cfg) }
func (c Calculator) Sub(a, b Number) (Cuts, error) { return Sub(a, b, c.Cfg) }
func (c Calculator) Mul(a, b Number) (Cuts, error) { return Mul(a, b, c.Cfg) }
func (c Calculator) Div(a, b Number) (Cuts, error) { return Div(a, b, c.Cfg) }
func (c Calculator) Neg(a Number) (Cuts, error)    { return Neg(a, c.Cfg) }
func (c Calculator) Scale(a Number, k float64) (Cuts, error) {
	return Scale(a, k, c.Cfg)
}
func (c Calculator) Apply(a, b Number, op Operation) (Cuts, error) {
	return Apply(a, b, op, c.Cfg)
}
