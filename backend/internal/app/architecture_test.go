package app

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Keep infrastructure and transport dependencies from leaking back into business rules.
func TestLayerBoundaries(t *testing.T) {
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		layer := strings.Split(filepath.ToSlash(rel), "/")[0]
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			name, _ := strconv.Unquote(imp.Path.Value)
			forbidden := false
			switch layer {
			case "model":
				forbidden = strings.HasPrefix(name, "english-app/internal/") && !strings.Contains(name, "/platform/") || strings.HasPrefix(name, "github.com/") || name == "net/http"
			case "repository":
				forbidden = strings.Contains(name, "/service") || strings.Contains(name, "/handler") || strings.Contains(name, "/provider/") || name == "net/http"
			case "service":
				forbidden = strings.Contains(name, "/handler") || strings.Contains(name, "/provider/") || strings.Contains(name, "/repository/postgres") || strings.Contains(name, "/database") || strings.Contains(name, "jackc/") || name == "net/http"
			case "handler":
				forbidden = strings.Contains(name, "/repository") || strings.Contains(name, "/database") || strings.Contains(name, "/provider/") || strings.Contains(name, "jackc/")
			case "provider":
				forbidden = strings.Contains(name, "/service") || strings.Contains(name, "/repository") || strings.Contains(name, "/handler") || strings.Contains(name, "/database")
			}
			if forbidden {
				t.Errorf("%s: dependency %s violates the %s layer", rel, name, layer)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
