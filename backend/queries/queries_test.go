package queries

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedQueryCatalogIsCompleteAndNonEmpty(t *testing.T) {
	names, err := Names()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 30 {
		t.Fatalf("expected the full query catalog, got %d files", len(names))
	}
	for _, name := range names {
		query, err := Read(name)
		if err != nil {
			t.Fatal(err)
		}
		query = strings.TrimSpace(query)
		if query == "" || !strings.HasSuffix(query, ";") {
			t.Errorf("query %s is empty or missing a terminating semicolon", name)
		}
	}
}

func TestReadRejectsPathTraversal(t *testing.T) {
	if _, err := Read("../migrations/000001_core_schema.sql"); err == nil {
		t.Fatal("expected query path traversal to be rejected")
	}
}

func TestEveryEmbeddedQueryIsReferencedByStorage(t *testing.T) {
	names, err := Names()
	if err != nil {
		t.Fatal(err)
	}
	storageDir := filepath.Join("..", "internal", "storage")
	files, err := os.ReadDir(storageDir)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`queries\.Must\("([^"]+)"\)`)
	used := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".go" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(storageDir, file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range pattern.FindAllSubmatch(contents, -1) {
			used[string(match[1])] = true
		}
	}
	for _, name := range names {
		if !used[name] {
			t.Errorf("embedded SQL file %q is not referenced by storage", name)
		}
	}
}
