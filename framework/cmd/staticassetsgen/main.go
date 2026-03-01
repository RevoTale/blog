package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"blog/framework/staticassets"
)

func main() {
	var sourceDir string
	var outDir string
	var manifestPath string
	var urlPrefix string

	flag.StringVar(&sourceDir, "source", "static", "source static directory")
	flag.StringVar(&outDir, "out", "static-build", "output static directory")
	flag.StringVar(&manifestPath, "manifest", "static-build/manifest.json", "manifest output path")
	flag.StringVar(&urlPrefix, "url-prefix", "/.revotale/", "base static URL prefix")
	flag.Parse()

	bundle, err := staticassets.Build(staticassets.BuildConfig{
		SourceDir: sourceDir,
		URLPrefix: urlPrefix,
	})
	if err != nil {
		exitf("build static bundle: %v", err)
	}
	defer func() {
		if cleanupErr := bundle.Cleanup(); cleanupErr != nil {
			exitf("cleanup temp bundle: %v", cleanupErr)
		}
	}()

	if err := os.RemoveAll(outDir); err != nil {
		exitf("clean output dir %q: %v", outDir, err)
	}
	if err := copyTree(bundle.Dir(), outDir); err != nil {
		exitf("copy processed assets to %q: %v", outDir, err)
	}

	if strings.TrimSpace(manifestPath) == "" {
		manifestPath = filepath.Join(outDir, "manifest.json")
	}
	if err := staticassets.WriteManifest(manifestPath, bundle.Manifest()); err != nil {
		exitf("write manifest %q: %v", manifestPath, err)
	}
}

func copyTree(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return os.MkdirAll(targetRoot, 0o755)
		}

		targetPath := filepath.Join(targetRoot, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			return err
		}

		return nil
	})
}

func exitf(formatText string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, formatText+"\n", args...)
	os.Exit(1)
}
