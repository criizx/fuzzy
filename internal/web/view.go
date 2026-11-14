package web

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"fuzzylab/internal/clustering"
	"fuzzylab/internal/fuzzy"
)

const palette = 10

type viewModel struct {
	Preview    bool
	Density    chart
	Membership chart
	Alpha      float64
	Seed       string
	Source     string
	Revision   uint64
	Count      int
	Message    string
	Error      string
	Clusters   []clusterRow
	Data       []dataRow
	Cuts       chart
	Plane      chart
}

type dataRow struct {
	Index, Cluster                   int
	Params                           string
	Lo, Hi                           float64
	Mode, Median, Expected, Variance float64
}

type clusterRow struct {
	Class  int
	Num    int
	Size   int
	Medoid string
}

type tick struct {
	Pos   float64
	Label string
}

type bar struct {
	ID         string
	X, Y, W, H float64
	Class      int
	Tip        string
}

type dot struct {
	ID     string
	X, Y   float64
	Class  int
	Medoid bool
	Tip    string
}

type poly struct {
	Class  int
	Points string
	Name   string
}

type segment struct {
	X1, Y1, X2, Y2 float64
}

// chart stores presentation-only SVG geometry.
type chart struct {
	Title                    string
	W, H                     int
	Left, Right, Top, Bottom float64
	XTicks, YTicks           []tick
	Bars                     []bar
	Dots                     []dot
	Polys                    []poly
	Diag                     *segment
	XLabel, YLabel           string
}

// Base returns the lower boundary of the plotting area.
func (c chart) Base() float64 { return float64(c.H) - c.Bottom }
func (c chart) End() float64  { return float64(c.W) - c.Right }

type scale struct{ lo, hi, p0, p1 float64 }

func (s scale) at(v float64) float64 { return s.p0 + (v-s.lo)/(s.hi-s.lo)*(s.p1-s.p0) }

func (s scale) ticks(n int) []tick {
	out := make([]tick, n+1)
	for i := range out {
		v := s.lo + (s.hi-s.lo)*float64(i)/float64(n)
		label := strconv.FormatFloat(v, 'g', 3, 64)
		if label == "-0" || label == "-0.0" {
			label = label[1:]
		}
		out[i] = tick{Pos: s.at(v), Label: label}
	}
	return out
}

func (s *Server) buildView(c ctl) viewModel {
	v := viewModel{Alpha: c.Alpha}
	run, err := s.experiments.Cluster(c)
	if err != nil {
		v.Error = err.Error()
		return v
	}
	items, res := run.Dataset.Items, run.Clusters
	v.Count, v.Source, v.Revision = len(items), run.Dataset.Source, run.Dataset.Revision
	v.Seed = "импорт CSV — seed неизвестен"
	if run.Dataset.Generation != nil {
		v.Seed = strconv.FormatInt(run.Dataset.Generation.Seed, 10)
	}
	cuts := make([]fuzzy.Interval, len(items))
	for i, it := range items {
		cuts[i] = fuzzy.CutClamped(it, c.Alpha)
	}
	dom := cuts[0]
	for _, cut := range cuts[1:] {
		dom = dom.Hull(cut)
	}
	dom = paddedDomain(dom)

	for i, it := range items {
		row := dataRow{Index: i + 1, Cluster: res.Labels[i] + 1, Params: fmt.Sprint(it), Lo: cuts[i].Lo, Hi: cuts[i].Hi}
		if n, ok := it.(fuzzy.LogNormal); ok {
			d := n.Distribution()
			row.Mode, row.Median, row.Expected, row.Variance = d.Mode(), d.Median(), d.Mean(), d.Variance()
		}
		v.Data = append(v.Data, row)
	}
	v.Clusters = clusterRows(items, res)
	v.Cuts = cutsChart(items, cuts, res, dom)
	v.Plane = planeChart(items, cuts, res)
	return v
}

func paddedDomain(dom fuzzy.Interval) fuzzy.Interval {
	pad := (dom.Hi - dom.Lo) * 0.03
	if pad == 0 {
		// Identical point cuts still need a nonzero axis range.
		pad = math.Max(math.Abs(dom.Lo), 1) * 0.03
	}
	return fuzzy.Interval{Lo: dom.Lo - pad, Hi: dom.Hi + pad}
}

func clusterRows(items []fuzzy.Number, res clustering.Result) []clusterRow {
	rows := make([]clusterRow, len(res.Medoids))
	for c, m := range res.Medoids {
		s := items[m].Support()
		rows[c] = clusterRow{Class: c % palette, Num: c + 1, Medoid: fmt.Sprintf("%v · [%.3g; %.3g]", items[m], s.Lo, s.Hi)}
	}
	for _, l := range res.Labels {
		rows[l].Size++
	}
	return rows
}

func tip(it fuzzy.Number, cut fuzzy.Interval, label int) string {
	s := it.Support()
	return fmt.Sprintf("[%.2f; %.2f] → срез [%.2f; %.2f], кластер %d", s.Lo, s.Hi, cut.Lo, cut.Hi, label+1)
}

// cutsChart prepares alpha-cut bars grouped by cluster.
func cutsChart(items []fuzzy.Number, cuts []fuzzy.Interval, res clustering.Result, dom fuzzy.Interval) chart {
	ch := chart{Title: "Интервалы α-срезов нечётких величин, сгруппированные по кластерам", W: 640, H: 380, Left: 16, Right: 16, Top: 12, Bottom: 34, XLabel: "x"}
	x := scale{dom.Lo, dom.Hi, ch.Left, float64(ch.W) - ch.Right}
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(p, q int) bool {
		i, j := order[p], order[q]
		if res.Labels[i] != res.Labels[j] {
			return res.Labels[i] < res.Labels[j]
		}
		return cuts[i].Lo < cuts[j].Lo
	})
	plotH := float64(ch.H) - ch.Top - ch.Bottom
	row := plotH / float64(len(items))
	for pos, i := range order {
		l := res.Labels[i]
		ch.Bars = append(ch.Bars, bar{
			ID: "b" + strconv.Itoa(i), X: x.at(cuts[i].Lo), W: math.Max(x.at(cuts[i].Hi)-x.at(cuts[i].Lo), 1.5),
			Y: ch.Top + float64(pos)*row, H: math.Max(row*0.8, 0.6), Class: l % palette, Tip: tip(items[i], cuts[i], l),
		})
	}
	ch.XTicks = x.ticks(6)
	return ch
}

// planeChart maps each alpha cut to (Lo,Hi), highlighting the medoids.
func planeChart(items []fuzzy.Number, cuts []fuzzy.Interval, res clustering.Result) chart {
	ch := chart{Title: "Карта нижних и верхних границ α-срезов; крупные точки — медоиды; масштабы осей независимы", W: 420, H: 380, Left: 44, Right: 16, Top: 12, Bottom: 34, XLabel: "нижняя граница среза", YLabel: "верхняя"}
	xdom := fuzzy.Interval{Lo: cuts[0].Lo, Hi: cuts[0].Lo}
	ydom := fuzzy.Interval{Lo: cuts[0].Hi, Hi: cuts[0].Hi}
	for _, cut := range cuts[1:] {
		xdom.Lo, xdom.Hi = math.Min(xdom.Lo, cut.Lo), math.Max(xdom.Hi, cut.Lo)
		ydom.Lo, ydom.Hi = math.Min(ydom.Lo, cut.Hi), math.Max(ydom.Hi, cut.Hi)
	}
	xdom, ydom = paddedDomain(xdom), paddedDomain(ydom)
	x := scale{xdom.Lo, xdom.Hi, ch.Left, float64(ch.W) - ch.Right}
	y := scale{ydom.Lo, ydom.Hi, float64(ch.H) - ch.Bottom, ch.Top}
	// Draw y=x only where its values lie within both visible axis ranges.
	lo, hi := math.Max(xdom.Lo, ydom.Lo), math.Min(xdom.Hi, ydom.Hi)
	if lo < hi {
		ch.Diag = &segment{X1: x.at(lo), Y1: y.at(lo), X2: x.at(hi), Y2: y.at(hi)}
	}
	isMedoid := map[int]bool{}
	for _, m := range res.Medoids {
		isMedoid[m] = true
	}
	for i, it := range items {
		l := res.Labels[i]
		ch.Dots = append(ch.Dots, dot{
			ID: "d" + strconv.Itoa(i), X: x.at(cuts[i].Lo), Y: y.at(cuts[i].Hi),
			Class: l % palette, Medoid: isMedoid[i], Tip: tip(it, cuts[i], l),
		})
	}
	ch.XTicks, ch.YTicks = x.ticks(5), y.ticks(5)
	return ch
}

// previewView prepares a dataset preview without running clustering.
func (s *Server) previewView() viewModel {
	d := s.experiments.Snapshot()
	v := viewModel{Preview: true, Count: len(d.Items), Source: d.Source, Revision: d.Revision, Seed: "импорт CSV — seed неизвестен"}
	if d.Generation != nil {
		v.Seed = strconv.FormatInt(d.Generation.Seed, 10)
	}
	if len(d.Items) == 0 {
		return v
	}
	items := d.Items[:min(3, len(d.Items))]
	names := make([]string, len(items))
	for i := range items {
		names[i] = fmt.Sprintf("№%d · %v", i+1, items[i])
	}
	v.Membership = opsChart(items, names)
	v.Membership.Title = "Функции принадлежности первых трёх величин"
	v.Density = densityChart(items, names)
	for i, it := range d.Items {
		row := dataRow{Index: i + 1, Params: fmt.Sprint(it), Lo: it.Support().Lo, Hi: it.Support().Hi}
		if n, ok := it.(fuzzy.LogNormal); ok {
			p := n.Distribution()
			row.Mode, row.Median, row.Expected, row.Variance = p.Mode(), p.Median(), p.Mean(), p.Variance()
		}
		v.Data = append(v.Data, row)
	}
	return v
}
