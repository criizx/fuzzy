// Package web serves the experiment interface using embedded HTML, CSS, SVG, and htmx.
package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"

	"fuzzylab/internal/experiment"
)

//go:embed templates static
var assets embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{"f": func(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}}).ParseFS(assets, "templates/*.html"))

// Server adapts HTTP requests to the experiment service and renders HTML.
type Server struct {
	experiments *experiment.Service
}

// genForm is the presentation alias for dataset generation settings.
type genForm = experiment.Generation

// ctl is the presentation alias for clustering controls.
type ctl = experiment.Controls

var (
	defaultGen = experiment.DefaultGeneration()
	defaultCtl = experiment.DefaultControls()
)

// New binds HTTP handlers to an already initialized experiment service.
func New(service *experiment.Service) *Server {
	return &Server{experiments: service}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(assets))
	mux.HandleFunc("GET /{$}", s.handle(noop, "index"))
	mux.HandleFunc("GET /view", s.handle(noop, "view"))
	mux.HandleFunc("GET /ops", s.opsHandler("ops"))
	mux.HandleFunc("GET /ops/calc", s.opsHandler("opsview"))
	mux.HandleFunc("POST /generate", s.handle(s.onGenerate, "view"))
	mux.HandleFunc("POST /save", s.handle(s.onSave, "view"))
	mux.HandleFunc("POST /load", s.handle(s.onLoad, "view"))
	return mux
}

// handle parses a request, invokes an action, and renders an HTML page or fragment.
func (s *Server) handle(act func(*http.Request) (string, error), page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "некорректный запрос", http.StatusBadRequest)
			return
		}
		msg, err := act(r)
		var v viewModel
		if page == "index" || r.URL.Path == "/generate" || r.URL.Path == "/load" {
			v = s.previewView()
		} else {
			v = s.buildView(parseCtl(r))
		}
		v.Message = msg
		if err != nil {
			v.Error = err.Error()
		}
		data := any(v)
		if page == "index" {
			data = struct {
				View viewModel
				Gen  genForm
				Ctl  ctl
				File string
			}{v, s.generation(), defaultCtl, "set.csv"}
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

func (s *Server) onGenerate(r *http.Request) (string, error) {
	f := defaultGen
	f.Mode = r.Form.Get("mode")
	var err error
	for _, p := range []struct {
		name, label string
		dst         any
	}{
		{"n", "число величин", &f.N},
		{"mulo", "μ от", &f.MuLo}, {"muhi", "μ до", &f.MuHi},
		{"sigmalo", "σ от", &f.SigmaLo}, {"sigmahi", "σ до", &f.SigmaHi},
		{"seed", "seed", &f.Seed},
	} {
		if e := scan(r.Form.Get(p.name), p.dst); e != nil && err == nil {
			err = fmt.Errorf("поле «%s»: %w", p.label, e)
		}
	}
	if f.Mode == "clouds" {
		for _, p := range []struct {
			name, label string
			dst         any
		}{{"groups", "облаков", &f.Groups}, {"noise", "шум", &f.Noise}} {
			if e := scan(r.Form.Get(p.name), p.dst); e != nil && err == nil {
				err = fmt.Errorf("поле «%s»: %w", p.label, e)
			}
		}
	}
	if err != nil {
		return "", err
	}
	if f.N < 100 || f.N > 300 {
		return "", fmt.Errorf("для лабораторной нужно от 100 до 300 величин")
	}
	if err := s.experiments.Generate(f); err != nil {
		return "", err
	}
	return fmt.Sprintf("Сгенерировано величин: %d", len(s.experiments.Snapshot().Items)), nil
}

func (s *Server) generation() genForm {
	if f := s.experiments.Snapshot().Generation; f != nil {
		return *f
	}
	return defaultGen
}

func (s *Server) onSave(r *http.Request) (string, error) {
	path, err := s.experiments.Save(r.Form.Get("file"))
	if err != nil {
		return "", err
	}
	return "Сохранено: " + path, nil
}

func (s *Server) onLoad(r *http.Request) (string, error) {
	path, err := s.experiments.Load(r.Form.Get("file"))
	if err != nil {
		return "", err
	}
	return "Загружено: " + path, nil
}

func parseCtl(r *http.Request) ctl {
	c := defaultCtl
	_ = scan(r.Form.Get("alpha"), &c.Alpha)
	_ = scan(r.Form.Get("k"), &c.K)
	_ = scan(r.Form.Get("threshold"), &c.Threshold)
	if r.Form.Get("algo") == "agglo" {
		c.Algo = "agglo"
	}
	c.Alpha = min(max(c.Alpha, 0), 1)
	c.K = max(c.K, 1)
	return c
}

// scan parses into *int, *int64, or *float64 without changing the destination on error.
func scan(s string, dst any) error {
	var err error
	switch d := dst.(type) {
	case *int:
		var v int
		if v, err = strconv.Atoi(s); err == nil {
			*d = v
		}
	case *int64:
		var v int64
		if v, err = strconv.ParseInt(s, 10, 64); err == nil {
			*d = v
		}
	case *float64:
		var v float64
		if v, err = strconv.ParseFloat(s, 64); err == nil {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("нужно конечное число")
			}
			*d = v
		}
	default:
		return fmt.Errorf("неподдерживаемый тип %T", dst)
	}
	if err != nil {
		return fmt.Errorf("ожидалось число, получено %q", s)
	}
	return nil
}

func noop(*http.Request) (string, error) { return "", nil }
