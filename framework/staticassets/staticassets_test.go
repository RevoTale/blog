package staticassets

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestBuild_MinifiesAndCopiesAssets(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	writeTestFile(
		t,
		filepath.Join(sourceDir, "app.js"),
		"function sum (a, b) {\n  return a + b\n}\nconsole.log(sum(1, 2))\n",
	)
	writeTestFile(t, filepath.Join(sourceDir, "styles.css"), "body {\n  color: red;\n  margin: 0;\n}\n")
	writeTestFile(t, filepath.Join(sourceDir, "logo.svg"), "<svg>\n  <rect width=\"10\" height=\"10\"/>\n</svg>\n")

	bundle, err := Build(BuildConfig{
		SourceDir: sourceDir,
		URLPrefix: "/.revotale/",
	})
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := bundle.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup bundle: %v", cleanupErr)
		}
	})

	prefixPattern := regexp.MustCompile(`^/\.revotale/[0-9a-f]{16}/$`)
	if !prefixPattern.MatchString(bundle.URLPrefix()) {
		t.Fatalf("unexpected url prefix %q", bundle.URLPrefix())
	}
	if got := bundle.URL("styles.css"); got != bundle.URLPrefix()+"styles.css" {
		t.Fatalf("unexpected asset url: %q", got)
	}

	minifiedJS := mustReadFile(t, filepath.Join(bundle.Dir(), "app.js"))
	originalJS := mustReadFile(t, filepath.Join(sourceDir, "app.js"))
	if string(minifiedJS) == string(originalJS) {
		t.Fatalf("expected js to be minified")
	}
	if len(minifiedJS) >= len(originalJS) {
		t.Fatalf("expected minified js to be smaller: original=%d minified=%d", len(originalJS), len(minifiedJS))
	}

	minifiedCSS := mustReadFile(t, filepath.Join(bundle.Dir(), "styles.css"))
	originalCSS := mustReadFile(t, filepath.Join(sourceDir, "styles.css"))
	if string(minifiedCSS) == string(originalCSS) {
		t.Fatalf("expected css to be minified")
	}
	if len(minifiedCSS) >= len(originalCSS) {
		t.Fatalf("expected minified css to be smaller: original=%d minified=%d", len(originalCSS), len(minifiedCSS))
	}

	copiedSVG := mustReadFile(t, filepath.Join(bundle.Dir(), "logo.svg"))
	originalSVG := mustReadFile(t, filepath.Join(sourceDir, "logo.svg"))
	if string(copiedSVG) != string(originalSVG) {
		t.Fatalf("expected svg to be copied unchanged")
	}
}

func TestBuild_HashDeterministicForSameSource(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	writeTestFile(t, filepath.Join(sourceDir, "app.js"), "const value = 1 + 2;\nconsole.log(value)\n")
	writeTestFile(t, filepath.Join(sourceDir, "styles.css"), "body { color: blue; }\n")
	writeTestFile(t, filepath.Join(sourceDir, "logo.svg"), "<svg><circle cx=\"5\" cy=\"5\" r=\"2\" /></svg>\n")

	first, err := Build(BuildConfig{SourceDir: sourceDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build first bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := first.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup first bundle: %v", cleanupErr)
		}
	})

	second, err := Build(BuildConfig{SourceDir: sourceDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build second bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := second.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup second bundle: %v", cleanupErr)
		}
	})

	if first.Hash() != second.Hash() {
		t.Fatalf("expected deterministic hash, got %q and %q", first.Hash(), second.Hash())
	}
}

func TestBuild_HashChangesWhenContentChanges(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	writeTestFile(t, filepath.Join(sourceDir, "app.js"), "console.log('a')\n")

	first, err := Build(BuildConfig{SourceDir: sourceDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build first bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := first.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup first bundle: %v", cleanupErr)
		}
	})

	writeTestFile(t, filepath.Join(sourceDir, "app.js"), "console.log('b')\n")
	second, err := Build(BuildConfig{SourceDir: sourceDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build second bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := second.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup second bundle: %v", cleanupErr)
		}
	})

	if first.Hash() == second.Hash() {
		t.Fatalf("expected hash to change after content update")
	}
}

func TestBuild_HashChangesWhenRelativePathChanges(t *testing.T) {
	t.Parallel()

	firstDir := t.TempDir()
	secondDir := t.TempDir()

	writeTestFile(t, filepath.Join(firstDir, "a.js"), "console.log('same')\n")
	writeTestFile(t, filepath.Join(secondDir, "nested", "a.js"), "console.log('same')\n")

	first, err := Build(BuildConfig{SourceDir: firstDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build first bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := first.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup first bundle: %v", cleanupErr)
		}
	})

	second, err := Build(BuildConfig{SourceDir: secondDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build second bundle: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := second.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup second bundle: %v", cleanupErr)
		}
	})

	if first.Hash() == second.Hash() {
		t.Fatalf("expected hash to change when relative path changes")
	}
}

func TestBundleCleanupRemovesOutputDir(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	writeTestFile(t, filepath.Join(sourceDir, "app.js"), "console.log('x')\n")

	bundle, err := Build(BuildConfig{SourceDir: sourceDir, URLPrefix: "/.revotale/"})
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}

	outDir := bundle.Dir()
	if outDir == "" {
		t.Fatalf("expected non-empty output dir")
	}
	if _, statErr := os.Stat(outDir); statErr != nil {
		t.Fatalf("expected output dir to exist: %v", statErr)
	}

	if cleanupErr := bundle.Cleanup(); cleanupErr != nil {
		t.Fatalf("cleanup bundle: %v", cleanupErr)
	}
	if _, statErr := os.Stat(outDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected output dir to be removed, stat err=%v", statErr)
	}
}

func TestManifestWriteAndRead(t *testing.T) {
	t.Parallel()

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	expected := Manifest{
		Hash:      "abc123",
		URLPrefix: "/.revotale/abc123/",
	}

	if err := WriteManifest(manifestPath, expected); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	actual, err := ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	if actual.Hash != expected.Hash {
		t.Fatalf("hash mismatch: expected %q, got %q", expected.Hash, actual.Hash)
	}
	if actual.URLPrefix != expected.URLPrefix {
		t.Fatalf("prefix mismatch: expected %q, got %q", expected.URLPrefix, actual.URLPrefix)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	return content
}
