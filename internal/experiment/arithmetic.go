package experiment

import (
	"fmt"

	"fuzzylab/internal/fuzzy"
)

// OperationRequest specifies two lognormal operands and a binary operation.
type OperationRequest struct {
	Mu1, Sigma1, Mu2, Sigma2 float64
	Op                       string
}

// Characteristic describes fuzzy geometry, not probabilistic moments.
type Characteristic struct {
	Name                     string
	Support                  fuzzy.Interval
	Centroid, Mean, Variance float64
}

// ArithmeticResult contains domain values ready for presentation without chart coordinates.
type ArithmeticResult struct {
	Items                  []fuzzy.Number
	Names                  []string
	Characteristics        []Characteristic
	Possibility, Necessity float64
}

func cloneArithmetic(r ArithmeticResult) ArithmeticResult {
	r.Items = append([]fuzzy.Number(nil), r.Items...)
	r.Names = append([]string(nil), r.Names...)
	r.Characteristics = append([]Characteristic(nil), r.Characteristics...)
	return r
}

// Calculate applies alpha-cut arithmetic and journals the outcome.
// Repeated requests reuse the last successfully journaled result.
func (s *Service) Calculate(f OperationRequest) (ArithmeticResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opCached != nil && s.lastOp == f {
		return cloneArithmetic(*s.opCached), s.opErr
	}
	var empty ArithmeticResult
	a, err := fuzzy.NewLogNormal(f.Mu1, f.Sigma1)
	if err != nil {
		return empty, fmt.Errorf("A: %w", err)
	}
	b, err := fuzzy.NewLogNormal(f.Mu2, f.Sigma2)
	if err != nil {
		return empty, fmt.Errorf("B: %w", err)
	}
	var fn func(fuzzy.Number, fuzzy.Number, fuzzy.Config) (fuzzy.Cuts, error)
	var symbol, label string
	switch f.Op {
	case "add":
		fn, symbol, label = fuzzy.Add, "+", "сложение"
	case "sub":
		fn, symbol, label = fuzzy.Sub, "−", "вычитание"
	case "mul":
		fn, symbol, label = fuzzy.Mul, "×", "умножение"
	case "div":
		fn, symbol, label = fuzzy.Div, "÷", "деление"
	default:
		return empty, fmt.Errorf("неизвестная операция %q", f.Op)
	}
	r := ArithmeticResult{Items: []fuzzy.Number{a, b}, Names: []string{"A", "B"}}
	value, opErr := fn(a, b, fuzzy.Config{})
	var result fuzzy.Number
	if opErr == nil {
		result = value
		r.Items = append(r.Items, value)
		r.Names = append(r.Names, "A "+symbol+" B")
	}
	if err = s.log.Operation(label, a, b, result, opErr); err != nil {
		return empty, err
	}
	for i, it := range r.Items {
		r.Characteristics = append(r.Characteristics, Characteristic{Name: r.Names[i], Support: it.Support(), Centroid: fuzzy.Centroid(it, fuzzy.Config{}), Mean: fuzzy.Mean(it, fuzzy.Config{}), Variance: fuzzy.Variance(it, fuzzy.Config{})})
	}
	r.Possibility, r.Necessity = fuzzy.PossibilityGE(a, b), fuzzy.NecessityGE(a, b)
	s.lastOp = f
	s.opCached = &r
	s.opErr = opErr
	return cloneArithmetic(r), opErr
}
