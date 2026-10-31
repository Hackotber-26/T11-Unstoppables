package queries

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed *.sql
var files embed.FS

func Read(name string) (string, error) {
	if !fs.ValidPath(name) || path.Base(name) != name || !strings.HasSuffix(name, ".sql") {
		return "", fmt.Errorf("invalid query name %q", name)
	}
	contents, err := files.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read embedded query %q: %w", name, err)
	}
	return string(contents), nil
}

func Must(name string) string {
	query, err := Read(name)
	if err != nil {
		panic(err)
	}
	return query
}

func Names() ([]string, error) {
	entries, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}
