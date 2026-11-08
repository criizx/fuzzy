// Package storage — сохранение и загрузка наборов логнормальных величин в CSV.
package storage

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
)

var header = []string{"kind", "mu", "sigma"}

// Save записывает набор в CSV (kind,mu,sigma), создавая каталоги пути.
func Save(path string, items []core.Number) error {
	rows := [][]string{header}
	for i, it := range items {
		u, ok := it.(number.LogNormal)
		if !ok {
			return fmt.Errorf("%w: элемент %d имеет тип %q, поддерживается только lognormal",
				core.ErrInvalidParams, i, it.Kind())
		}
		rows = append(rows, []string{u.Kind(), format(u.Mu), format(u.Sigma)})
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fileErr("создать каталог для", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fileErr("создать", path, err)
	}
	if err := csv.NewWriter(f).WriteAll(rows); err != nil {
		f.Close()
		return fileErr("записать", path, err)
	}
	return f.Close()
}

func fileErr(action, path string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("файл не найден: %s", path)
	}
	return fmt.Errorf("не удалось %s файл %s: %w", action, path, err)
}

// Load читает набор, записанный Save.
func Load(path string) ([]core.Number, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fileErr("открыть", path, err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: файл не в формате CSV (kind,mu,sigma)", core.ErrInvalidParams)
	}
	if len(rows) < 2 {
		return nil, core.ErrEmpty
	}
	if len(rows) > 301 {
		return nil, fmt.Errorf("%w: максимум 300 величин", core.ErrInvalidParams)
	}
	if len(rows[0]) != 3 || rows[0][0] != "kind" || rows[0][1] != "mu" || rows[0][2] != "sigma" {
		return nil, fmt.Errorf("%w: ожидается заголовок kind,mu,sigma", core.ErrInvalidParams)
	}
	items := make([]core.Number, 0, len(rows)-1)
	for i, row := range rows[1:] {
		if len(row) != len(header) || row[0] != "lognormal" {
			return nil, fmt.Errorf("%w: строка %d (ожидалось lognormal,a,b)", core.ErrInvalidParams, i+2)
		}
		a, errA := strconv.ParseFloat(row[1], 64)
		b, errB := strconv.ParseFloat(row[2], 64)
		if errA != nil || errB != nil {
			return nil, fmt.Errorf("%w: строка %d (μ и σ должны быть числами)", core.ErrInvalidParams, i+2)
		}
		u, err := number.NewLogNormal(a, b)
		if err != nil {
			return nil, fmt.Errorf("строка %d: %w", i+2, err)
		}
		items = append(items, u)
	}
	return items, nil
}

func format(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
