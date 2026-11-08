package web

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"fuzzylab/internal/fuzzy"
)

// densityChart samples the original probability density over the working display window.
// The probability model itself is not truncated, and the vertical axis is not limited to one.
func densityChart(items []fuzzy.Number, names []string) chart {
	ch := chart{Title: "Плотности вероятности первых трёх величин", W: 640, H: 320, Left: 52, Right: 20, Top: 20, Bottom: 34, XLabel: "x", YLabel: "f(x)"}
	dom := items[0].Support()
	peak := 0.0
	for _, it := range items {
		dom = dom.Hull(it.Support())
		if n, ok := it.(fuzzy.LogNormal); ok {
			peak = math.Max(peak, n.Density(n.Mode()))
		}
	}
	x := scale{0, dom.Hi * 1.03, ch.Left, ch.End()}
	y := scale{0, peak * 1.1, ch.Base(), ch.Top}
	for i, it := range items {
		n, ok := it.(fuzzy.LogNormal)
		if !ok {
			continue
		}
		support := n.Support()
		values := []float64{n.Mode()}
		for j := 0; j <= 160; j++ {
			values = append(values, math.Exp(math.Log(support.Lo)+float64(j)/160*(math.Log(support.Hi)-math.Log(support.Lo))))
		}
		sort.Float64s(values)
		var points strings.Builder
		fmt.Fprintf(&points, "%.2f,%.2f ", x.at(values[0]), ch.Base())
		for _, v := range values {
			fmt.Fprintf(&points, "%.2f,%.2f ", x.at(v), y.at(n.Density(v)))
		}
		fmt.Fprintf(&points, "%.2f,%.2f", x.at(values[len(values)-1]), ch.Base())
		ch.Polys = append(ch.Polys, poly{Class: i, Points: points.String(), Name: names[i]})
	}
	ch.XTicks, ch.YTicks = x.ticks(5), y.ticks(5)
	return ch
}
