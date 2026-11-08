package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fuzzylab/internal/experiment"
	"fuzzylab/internal/logger"
)

func newHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	l, err := logger.Open(filepath.Join(dir, "log.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	service, err := experiment.New(dir, l)
	if err != nil {
		t.Fatal(err)
	}
	return New(service).Handler(), dir
}

func do(h http.Handler, method, target string, form url.Values) string {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Body.String()
}

func TestIndexAndView(t *testing.T) {
	h, _ := newHandler(t)
	if page := do(h, "GET", "/", nil); !strings.Contains(page, "<svg") || !strings.Contains(page, `name="alpha"`) {
		t.Error("index must contain charts and α control")
	}
	for _, q := range []string{"/view?alpha=0.5&algo=pam&k=3", "/view?alpha=1&algo=agglo&threshold=5", "/view?alpha=zzz&k=-4"} {
		out := do(h, "GET", q, nil)
		if !strings.Contains(out, "<circle") || strings.Contains(out, `class="error"`) {
			t.Errorf("%s: bad view: %.200s", q, out)
		}
	}
}

func TestGenerateSaveLoad(t *testing.T) {
	h, dir := newHandler(t)
	gen := url.Values{"mode": {"random"}, "n": {"120"}, "groups": {"3"}, "mulo": {"0"}, "muhi": {"2"},
		"sigmalo": {"0.2"}, "sigmahi": {"0.5"}, "noise": {"0.05"}, "seed": {"7"}}
	if out := do(h, "POST", "/generate", gen); !strings.Contains(out, "Объектов</span><strong>120</strong>") {
		t.Errorf("generate: %.200s", out)
	}
	gen.Set("n", "abc")
	if out := do(h, "POST", "/generate", gen); !strings.Contains(out, `class="error"`) {
		t.Error("invalid n must give an error")
	}
	if out := do(h, "POST", "/save", url.Values{"file": {"../x"}}); !strings.Contains(out, "Сохранено") {
		t.Errorf("save: %.200s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.csv")); err != nil {
		t.Error("file must be saved inside data dir:", err)
	}
	if out := do(h, "POST", "/load", url.Values{"file": {"x"}}); !strings.Contains(out, `class="error"`) {
		t.Error("загрузка должна читать только из data/input")
	}
	if err := os.MkdirAll(filepath.Join(dir, "input"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "x.csv"), filepath.Join(dir, "input", "x.csv")); err != nil {
		t.Fatal(err)
	}
	if out := do(h, "POST", "/load", url.Values{"file": {"x"}}); !strings.Contains(out, "Объектов</span><strong>120</strong>") {
		t.Errorf("load: %.200s", out)
	}
	if out := do(h, "POST", "/load", url.Values{"file": {"missing"}}); !strings.Contains(out, `class="error"`) {
		t.Error("missing file must give an error")
	}
}

func TestOps(t *testing.T) {
	h, _ := newHandler(t)
	if page := do(h, "GET", "/ops", nil); !strings.Contains(page, "<polygon") || !strings.Contains(page, `name="op"`) {
		t.Error("ops page must contain chart and operation selector")
	}
	q := func(op, mu2, sigma2 string) string {
		return do(h, "GET", "/ops/calc?mu1=0&sigma1=0.3&mu2="+mu2+"&sigma2="+sigma2+"&op="+op, nil)
	}
	if out := q("add", "0.7", "0.4"); strings.Count(out, "<polygon") != 3 || strings.Contains(out, `class="error"`) {
		t.Errorf("add: %.300s", out)
	}
	if out := q("div", "-1", "0"); !strings.Contains(out, `class="error"`) || strings.Count(out, "<polygon") != 0 {
		t.Errorf("div by interval with zero must report error and show only A and B: %.300s", out)
	}
	if out := q("add", "5", "-2"); !strings.Contains(out, `class="error"`) {
		t.Error("a >= b must give an error")
	}
}

func TestStatic(t *testing.T) {
	h, _ := newHandler(t)
	if out := do(h, "GET", "/static/htmx.min.js", nil); !strings.Contains(out, "htmx") {
		t.Error("htmx must be served")
	}
}

func TestGenerationPreviewsBeforeExplicitCalculation(t *testing.T) {
	h, dir := newHandler(t)
	journal := func() string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(dir, "log.json"))
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	page := do(h, "GET", "/", nil)
	if !strings.Contains(page, "Предварительный просмотр") || !strings.Contains(page, "Вероятностная модель") || strings.Contains(journal(), "размеры_кластеров") {
		t.Fatal("opening preview ran clustering or hid the probability model")
	}
	out := do(h, "GET", "/view?alpha=0.5&algo=pam&k=3", nil)
	if !strings.Contains(out, "Результат эксперимента") {
		t.Fatal("explicit calculation did not return results")
	}
	first := journal()
	do(h, "GET", "/view?alpha=0.5&algo=pam&k=3", nil)
	if journal() != first {
		t.Fatal("unchanged view repeated calculation audit")
	}
}

func TestRandomGenerationIgnoresCloudOnlyFields(t *testing.T) {
	h, _ := newHandler(t)
	form := url.Values{"mode": {"random"}, "n": {"120"}, "mulo": {"0"}, "muhi": {"2"}, "sigmalo": {"0.2"}, "sigmahi": {"0.5"}, "seed": {"7"}, "groups": {"invalid"}, "noise": {"invalid"}}
	if out := do(h, "POST", "/generate", form); strings.Contains(out, `class="error"`) || !strings.Contains(out, "Предварительный просмотр") {
		t.Fatal("random mode must ignore cloud-only fields and return a preview")
	}
	form.Set("mode", "clouds")
	if out := do(h, "POST", "/generate", form); !strings.Contains(out, `class="error"`) {
		t.Fatal("cloud mode must validate groups and noise")
	}
	form.Set("mode", "random")
	form.Set("mulo", "invalid")
	if out := do(h, "POST", "/generate", form); !strings.Contains(out, `class="error"`) {
		t.Fatal("random mode must validate common parameters")
	}
}

func TestOversizedFormIsRejected(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest("POST", "/generate", strings.NewReader("mode="+strings.Repeat("x", (1<<20)+1)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}
