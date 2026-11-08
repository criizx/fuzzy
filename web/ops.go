package web

import (
	"fmt"
	"net/http"
	"strings"

	"fuzzylab/fuzzy/compare"
	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/metrics"
	"fuzzylab/fuzzy/number"
	"fuzzylab/fuzzy/ops"
)

// opsForm — две логнормальные величины и операция над ними.
type opsForm struct {
	Mu1, Sigma1, Mu2, Sigma2 float64
	Op                       string
}

type opChoice struct {
	Key, Sym, Label string
	Fn              func(a, b core.Number, cfg core.Config) (number.Cuts, error)
}

var opChoices = []opChoice{
	{"add", "+", "Сложение", ops.Add},
	{"sub", "−", "Вычитание", ops.Sub},
	{"mul", "×", "Умножение", ops.Mul},
	{"div", "÷", "Деление", ops.Div},
}

var defaultOps = opsForm{Mu1: 0, Sigma1: 0.3, Mu2: 0.7, Sigma2: 0.4, Op: "add"}

type legendRow struct {
	Class                    int
	Name                     string
	Span                     string
	Centroid, Mean, Variance float64
}

type opsView struct {
	Error    string
	Chart    chart
	Legend   []legendRow
	Pos, Nec float64
	Ready    bool
}

func parseOps(r *http.Request) (opsForm, error) {
	f := defaultOps
	if !r.Form.Has("mu1") {
		return f, nil
	}
	f.Op = r.Form.Get("op")
	for _, p := range []struct {
		name, label string
		dst         *float64
	}{{"mu1", "A: μ", &f.Mu1}, {"sigma1", "A: σ", &f.Sigma1}, {"mu2", "B: μ", &f.Mu2}, {"sigma2", "B: σ", &f.Sigma2}} {
		if err := scan(r.Form.Get(p.name), p.dst); err != nil {
			return f, fmt.Errorf("поле «%s»: %w", p.label, err)
		}
	}
	return f, nil
}

func (s *Server) opsHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		_ = r.ParseForm()
		f, err := parseOps(r)
		var v opsView
		if err == nil {
			v, err = s.calcOps(f)
		}
		if err != nil {
			v.Error = err.Error()
		}
		data := any(v)
		if page == "ops" {
			data = struct {
				Form opsForm
				View opsView
				Ops  []opChoice
			}{f, v, opChoices}
		}
		if err := tmpl.ExecuteTemplate(w, page, data); err != nil {
			http.Error(w, "ошибка отображения страницы: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// calcOps выполняет операцию, пишет её в журнал и готовит данные для графика.
func (s *Server) calcOps(f opsForm) (opsView, error) {
	var v opsView
	a, err := number.NewLogNormal(f.Mu1, f.Sigma1)
	if err != nil {
		return v, fmt.Errorf("A: %w", err)
	}
	b, err := number.NewLogNormal(f.Mu2, f.Sigma2)
	if err != nil {
		return v, fmt.Errorf("B: %w", err)
	}
	op := opChoices[0]
	found := false
	for _, c := range opChoices {
		if c.Key == f.Op {
			op = c
			found = true
		}
	}
	if !found {
		return v, fmt.Errorf("неизвестная операция %q", f.Op)
	}
	items, names := []core.Number{a, b}, []string{"A", "B"}
	res, opErr := op.Fn(a, b, core.Config{})
	var result core.Number
	if opErr == nil {
		result = res
		items, names = append(items, res), append(names, "A "+op.Sym+" B")
	}
	if key := fmt.Sprint(f, opErr); key != s.lastOp {
		if err := s.log.Operation(strings.ToLower(op.Label), a, b, result, opErr); err != nil {
			return v, err
		}
		s.lastOp = key
	}
	v.Chart = opsChart(items, names)
	for i, it := range items {
		sp := it.Support()
		v.Legend = append(v.Legend, legendRow{Class: i, Name: names[i], Centroid: metrics.Centroid(it, core.Config{}), Mean: metrics.Mean(it, core.Config{}), Variance: metrics.Variance(it, core.Config{}), Span: fmt.Sprintf("[%.2f; %.2f]", sp.Lo, sp.Hi)})
	}
	v.Pos, v.Nec = compare.PossibilityGE(a, b), compare.NecessityGE(a, b)
	v.Ready = true
	return v, opErr
}

// opsChart рисует функции принадлежности величин на одной оси.
func opsChart(items []core.Number, names []string) chart {
	dom := items[0].Support()
	for _, it := range items[1:] {
		dom = dom.Hull(it.Support())
	}
	pad := (dom.Hi - dom.Lo) * 0.05
	ch := chart{Title: "Функции принадлежности A, B и результата операции", W: 640, H: 320, Left: 44, Right: 16, Top: 12, Bottom: 34, XLabel: "x", YLabel: "μ"}
	x := scale{dom.Lo - pad, dom.Hi + pad, ch.Left, ch.End()}
	y := scale{0, 1, ch.Base(), ch.Top}
	for i, it := range items {
		var pts strings.Builder
		for _, p := range number.Outline(it, 40) {
			fmt.Fprintf(&pts, "%.2f,%.2f ", x.at(p.X), y.at(p.Y))
		}
		ch.Polys = append(ch.Polys, poly{Class: i, Points: pts.String(), Name: names[i]})
	}
	ch.XTicks, ch.YTicks = x.ticks(6), y.ticks(5)
	return ch
}
