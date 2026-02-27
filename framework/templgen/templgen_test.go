package templgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunGeneratesOutputFromPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourcePath := filepath.Join(root, "hello.templ")
	writeTestFile(t, sourcePath, "package sample\n\ntempl Hello() { <div>hello</div> }\n")

	err := Run(Config{
		Paths:    []string{root},
		BasePath: root,
	})
	if err != nil {
		t.Fatalf("run templgen: %v", err)
	}

	generatedPath := filepath.Join(root, "hello_templ.go")
	if _, statErr := os.Stat(generatedPath); statErr != nil {
		t.Fatalf("expected generated file %q: %v", generatedPath, statErr)
	}
}

func TestRunReturnsErrorForEmptySelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	err := Run(Config{
		BasePath: root,
	})
	if err == nil {
		t.Fatal("expected no templ files error")
	}
}

func writeTestFile(t *testing.T, filePath string, content string) {
	t.Helper()
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", filePath, err)
	}
}
