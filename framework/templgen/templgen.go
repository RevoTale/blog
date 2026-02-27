package templgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/a-h/templ/generator"
	"github.com/a-h/templ/parser/v2"
)

type Config struct {
	Files    []string
	Paths    []string
	BasePath string
}

func Run(cfg Config) error {
	basePath := strings.TrimSpace(cfg.BasePath)
	if basePath == "" {
		basePath = "."
	}

	resolvedFiles, err := collectFiles(cfg.Files, cfg.Paths)
	if err != nil {
		return err
	}
	if len(resolvedFiles) == 0 {
		return errors.New("no templ files found")
	}

	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("resolve base path %q: %w", basePath, err)
	}

	for _, fileName := range resolvedFiles {
		if err := generateFile(fileName, baseAbs); err != nil {
			return err
		}
	}

	return nil
}

func collectFiles(files []string, paths []string) ([]string, error) {
	seen := make(map[string]struct{})
	all := make([]string, 0, len(files)+8)

	for _, fileName := range files {
		absPath, err := filepath.Abs(fileName)
		if err != nil {
			return nil, fmt.Errorf("resolve file %q: %w", fileName, err)
		}
		if filepath.Ext(absPath) != ".templ" {
			return nil, fmt.Errorf("file %q must have .templ extension", fileName)
		}
		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}
		all = append(all, absPath)
	}

	for _, root := range paths {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve path %q: %w", root, err)
		}
		walkErr := filepath.WalkDir(rootAbs, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Ext(filePath) != ".templ" {
				return nil
			}
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return err
			}
			if _, ok := seen[absPath]; ok {
				return nil
			}
			seen[absPath] = struct{}{}
			all = append(all, absPath)
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("walk path %q: %w", root, walkErr)
		}
	}

	sort.Strings(all)
	return all, nil
}

func generateFile(fileName string, baseAbs string) error {
	t, err := parser.Parse(fileName)
	if err != nil {
		return fmt.Errorf("parse %q: %w", fileName, err)
	}

	relFileName, err := filepath.Rel(baseAbs, fileName)
	if err != nil {
		return fmt.Errorf("compute relative filename for %q: %w", fileName, err)
	}
	relFileName = filepath.ToSlash(relFileName)

	var output bytes.Buffer
	_, err = generator.Generate(t, &output, generator.WithFileName(relFileName))
	if err != nil {
		return fmt.Errorf("generate %q: %w", fileName, err)
	}

	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return fmt.Errorf("format generated output for %q: %w", fileName, err)
	}

	target := strings.TrimSuffix(fileName, ".templ") + "_templ.go"
	if err := os.WriteFile(target, formatted, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", target, err)
	}

	return nil
}
