package staticassets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	defaultURLPrefix = "/.revotale/"
	hashLength       = 16
)

type BuildConfig struct {
	SourceDir string
	URLPrefix string
}

type Manifest struct {
	Hash      string `json:"hash"`
	URLPrefix string `json:"url_prefix"`
}

type Bundle struct {
	hash      string
	urlPrefix string
	dir       string
}

func Build(cfg BuildConfig) (*Bundle, error) {
	sourceDir := strings.TrimSpace(cfg.SourceDir)
	if sourceDir == "" {
		return nil, fmt.Errorf("source dir is required")
	}

	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("stat source dir %q: %w", sourceDir, err)
	}
	if !sourceInfo.IsDir() {
		return nil, fmt.Errorf("source dir %q is not a directory", sourceDir)
	}

	entries, err := collectFiles(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("collect source files: %w", err)
	}

	outDir, err := os.MkdirTemp("", "blog-static-assets-*")
	if err != nil {
		return nil, fmt.Errorf("create temp output dir: %w", err)
	}

	hasher := sha256.New()
	for _, relativePath := range entries {
		sourcePath := filepath.Join(sourceDir, filepath.FromSlash(relativePath))
		content, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			_ = os.RemoveAll(outDir)
			return nil, fmt.Errorf("read source file %q: %w", sourcePath, readErr)
		}

		processed, processErr := processContent(relativePath, content)
		if processErr != nil {
			_ = os.RemoveAll(outDir)
			return nil, processErr
		}

		targetPath := filepath.Join(outDir, filepath.FromSlash(relativePath))
		if mkErr := os.MkdirAll(filepath.Dir(targetPath), 0o755); mkErr != nil {
			_ = os.RemoveAll(outDir)
			return nil, fmt.Errorf("create target dir for %q: %w", targetPath, mkErr)
		}
		if writeErr := os.WriteFile(targetPath, processed, 0o644); writeErr != nil {
			_ = os.RemoveAll(outDir)
			return nil, fmt.Errorf("write processed file %q: %w", targetPath, writeErr)
		}

		_, _ = hasher.Write([]byte(relativePath))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(processed)
		_, _ = hasher.Write([]byte{0})
	}

	fullHash := hex.EncodeToString(hasher.Sum(nil))
	shortHash := fullHash
	if len(shortHash) > hashLength {
		shortHash = shortHash[:hashLength]
	}

	normalizedPrefix := normalizeURLPrefix(cfg.URLPrefix)
	versionedPrefix := normalizedPrefix + shortHash + "/"

	return &Bundle{
		hash:      shortHash,
		urlPrefix: versionedPrefix,
		dir:       outDir,
	}, nil
}

func (bundle *Bundle) Hash() string {
	if bundle == nil {
		return ""
	}

	return bundle.hash
}

func (bundle *Bundle) URLPrefix() string {
	if bundle == nil {
		return ""
	}

	return bundle.urlPrefix
}

func (bundle *Bundle) Dir() string {
	if bundle == nil {
		return ""
	}

	return bundle.dir
}

func (bundle *Bundle) URL(path string) string {
	if bundle == nil {
		return ""
	}

	trimmed := strings.TrimSpace(path)
	trimmed = strings.ReplaceAll(trimmed, `\`, `/`)
	trimmed = strings.TrimPrefix(trimmed, "/")
	return bundle.urlPrefix + trimmed
}

func (bundle *Bundle) Cleanup() error {
	if bundle == nil || strings.TrimSpace(bundle.dir) == "" {
		return nil
	}

	return os.RemoveAll(bundle.dir)
}

func (bundle *Bundle) Manifest() Manifest {
	if bundle == nil {
		return Manifest{}
	}

	return Manifest{
		Hash:      bundle.hash,
		URLPrefix: bundle.urlPrefix,
	}
}

func WriteManifest(path string, manifest Manifest) error {
	manifest = normalizeManifest(manifest)
	if err := validateManifest(manifest); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	encoded = append(encoded, '\n')

	manifestPath := strings.TrimSpace(path)
	if manifestPath == "" {
		return fmt.Errorf("manifest path is required")
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func ReadManifest(path string) (Manifest, error) {
	manifestPath := strings.TrimSpace(path)
	if manifestPath == "" {
		return Manifest{}, fmt.Errorf("manifest path is required")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %q: %w", manifestPath, err)
	}

	manifest = normalizeManifest(manifest)
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func collectFiles(root string) ([]string, error) {
	files := make([]string, 0, 8)
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %q: %w", path, err)
		}
		normalized := filepath.ToSlash(relativePath)
		if strings.TrimSpace(normalized) == "" || normalized == "." {
			return nil
		}
		files = append(files, normalized)
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func processContent(relativePath string, content []byte) ([]byte, error) {
	extension := strings.ToLower(filepath.Ext(relativePath))
	switch extension {
	case ".js", ".mjs", ".cjs":
		return transformWithEsbuild(relativePath, content, api.LoaderJS)
	case ".css":
		return transformWithEsbuild(relativePath, content, api.LoaderCSS)
	default:
		copied := make([]byte, len(content))
		copy(copied, content)
		return copied, nil
	}
}

func transformWithEsbuild(relativePath string, content []byte, loader api.Loader) ([]byte, error) {
	result := api.Transform(string(content), api.TransformOptions{
		Loader:            loader,
		Sourcefile:        relativePath,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
	})

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("minify %q: %s", relativePath, result.Errors[0].Text)
	}

	return result.Code, nil
}

func normalizeURLPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		trimmed = defaultURLPrefix
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if !strings.HasSuffix(trimmed, "/") {
		trimmed += "/"
	}

	return trimmed
}

func normalizeManifest(manifest Manifest) Manifest {
	manifest.Hash = strings.TrimSpace(manifest.Hash)
	manifest.URLPrefix = normalizeURLPrefix(manifest.URLPrefix)
	return manifest
}

func validateManifest(manifest Manifest) error {
	if strings.TrimSpace(manifest.Hash) == "" {
		return fmt.Errorf("manifest hash is required")
	}
	if strings.TrimSpace(manifest.URLPrefix) == "" {
		return fmt.Errorf("manifest url prefix is required")
	}
	return nil
}
