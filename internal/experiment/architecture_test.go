package experiment_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestPackageLayoutAndDependencyDirection(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	allowed := map[string]map[string]bool{
		"distribution": {},
		"fuzzy":        {"distribution": true},
		"clustering":   {"fuzzy": true},
		"generator":    {"fuzzy": true},
		"storage":      {"fuzzy": true},
		"logger":       {"fuzzy": true, "clustering": true},
		"experiment":   {"generator": true, "fuzzy": true, "clustering": true, "storage": true},
		"web":          {"experiment": true, "fuzzy": true, "clustering": true},
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		if len(parts) == 2 && parts[0] == "cmd" {
			return nil
		}
		if len(parts) != 3 || parts[0] != "internal" {
			t.Errorf("Go package outside flat internal/cmd layout: %s", rel)
			return nil
		}
		rules, ok := allowed[parts[1]]
		if !ok {
			t.Errorf("unexpected package: %s", rel)
			return nil
		}
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(value, "fuzzylab/") {
				continue
			}
			dep := strings.TrimPrefix(value, "fuzzylab/internal/")
			if !rules[dep] {
				t.Errorf("forbidden production dependency %s -> %s", rel, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
