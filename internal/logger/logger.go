// Package logger writes audit actions as JSON Lines.
package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fuzzylab/internal/clustering"
	"fuzzylab/internal/fuzzy"
)

// Logger appends one JSON Lines record per application action.
type Logger struct {
	mu sync.Mutex
	f  *os.File
}

type entry struct {
	Time   string `json:"время"`
	Action string `json:"действие"`
	Data   any    `json:"данные"`
}

type span [2]float64

type operation struct {
	A           span   `json:"A"`
	B           *span  `json:"B,omitempty"`
	Result      *span  `json:"результат,omitempty"`
	Error       string `json:"ошибка,omitempty"`
	ParametersA any    `json:"параметры_A,omitempty"`
	ParametersB any    `json:"параметры_B,omitempty"`
}

type clusteringEvent struct {
	Algo   string         `json:"алгоритм"`
	Params map[string]any `json:"параметры"`
	Sizes  []int          `json:"размеры_кластеров"`
}

// Open creates parent directories and opens an append-only journal.
func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f}, nil
}

func (l *Logger) Close() error { return l.f.Close() }

// Log appends an arbitrary audit action.
func (l *Logger) Log(action string, data any) error {
	line, err := json.Marshal(entry{Time: time.Now().Format(time.RFC3339), Action: action, Data: data})
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.f.Write(append(line, '\n'))
	return err
}

// Operation records operands, their parameters, and the result or error.
// The second operand and result may be nil for unary operations and failures.
func (l *Logger) Operation(op string, a, b, result fuzzy.Number, opErr error) error {
	d := operation{ParametersA: parameters(a), ParametersB: parameters(b), A: *toSpan(a), B: toSpan(b), Result: toSpan(result)}
	if opErr != nil {
		d.Error = opErr.Error()
	}
	return l.Log(op, d)
}

// Clustering records the algorithm, parameters, and cluster sizes.
func (l *Logger) Clustering(algo string, params map[string]any, r clustering.Result) error {
	sizes := make([]int, len(r.Medoids))
	for _, c := range r.Labels {
		sizes[c]++
	}
	return l.Log("кластеризация", clusteringEvent{Algo: algo, Params: params, Sizes: sizes})
}

func toSpan(n fuzzy.Number) *span {
	if n == nil {
		return nil
	}
	s := n.Support()
	return &span{s.Lo, s.Hi}
}

func parameters(n fuzzy.Number) any {
	if v, ok := n.(fuzzy.LogNormal); ok {
		return map[string]float64{"mu": v.Mu, "sigma": v.Sigma, "epsilon": fuzzy.TailEpsilon}
	}
	return nil
}
