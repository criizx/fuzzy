package web

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"fuzzylab/internal/experiment"
	"fuzzylab/internal/fuzzy"
)

// opsForm describes two lognormal operands and their requested operation.
type opsForm = experiment.OperationRequest

type opChoice struct {
	Key, Sym, Label string
}

var opChoices = []opChoice{
	{"add", "+", "Сложение"},
	{"sub", "−", "Вычитание"},
	{"mul", "×", "Умножение"},
	{"div", "÷", "Деление"},
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
		if err := r.ParseForm(); err != nil {
			http.Error(w, "некорректный запрос", http.StatusBadRequest)
			return
		}
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
		var body bytes.Buffer
		if err := tmpl.ExecuteTemplate(&body, page, data); err != nil {
			http.Error(w, "ошибка отображения страницы: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = body.WriteTo(w)
	}
}

// calcOps adapts a domain calculation result into chart and table data.
func (s *Server) calcOps(f opsForm) (opsView, error) {
	r, err := s.experiments.Calculate(f)
	var v opsView
	if len(r.Items) == 0 {
		return v, err
	}
	v.Chart = opsChart(r.Items, r.Names)
	for i, it := range r.Characteristics {
		v.Legend = append(v.Legend, legendRow{Class: i, Name: it.Name, Centroid: it.Centroid, Mean: it.Mean, Variance: it.Variance, Span: fmt.Sprintf("[%.2f; %.2f]", it.Support.Lo, it.Support.Hi)})
	}
	v.Pos, v.Nec, v.Ready = r.Possibility, r.Necessity, true
	return v, err
}

// opsChart prepares membership outlines on a shared coordinate system.
func opsChart(items []fuzzy.Number, names []string) chart {
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
		for _, p := range fuzzy.Outline(it, 40) {
			fmt.Fprintf(&pts, "%.2f,%.2f ", x.at(p.X), y.at(p.Y))
		}
		ch.Polys = append(ch.Polys, poly{Class: i, Points: pts.String(), Name: names[i]})
	}
	ch.XTicks, ch.YTicks = x.ticks(6), y.ticks(5)
	return ch
}
