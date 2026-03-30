package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/suessflorian/gqlfetch"
)

const (
	defaultEndpoint   = "http://localhost:3000/api/graphql"
	defaultOutputFile = "schema.graphql"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "fetchschema: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	endpoint := strings.TrimSpace(os.Getenv("BLOG_GRAPHQL_ENDPOINT"))
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	authToken := strings.TrimSpace(os.Getenv("BLOG_GRAPHQL_AUTH_TOKEN"))
	var outPath string
	var withoutBuiltins bool
	headers := make(headerFlags)

	flags := flag.NewFlagSet("fetchschema", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&endpoint, "endpoint", endpoint, "GraphQL server endpoint")
	flags.StringVar(&outPath, "out", "", "output file path (defaults to <project-root>/schema.graphql)")
	flags.BoolVar(&withoutBuiltins, "without-builtins", false, "Do not include builtin GraphQL types")
	flags.Var(&headers, "header", "Header to pass through to the endpoint as key=value (can be repeated)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	resolvedOutPath, err := resolveOutputPath(outPath)
	if err != nil {
		return err
	}

	schema, err := gqlfetch.BuildClientSchemaWithHeaders(
		ctx,
		endpoint,
		buildHeaders(authToken, headers),
		withoutBuiltins,
	)
	if err != nil {
		return fmt.Errorf("download schema from %s: %w", endpoint, err)
	}

	if err := writeFileAtomic(resolvedOutPath, []byte(schema)); err != nil {
		return fmt.Errorf("write %s: %w", resolvedOutPath, err)
	}

	if _, err := fmt.Fprintf(stdout, "wrote %s\n", resolvedOutPath); err != nil {
		return fmt.Errorf("write status output: %w", err)
	}

	return nil
}

func resolveOutputPath(outPath string) (string, error) {
	if strings.TrimSpace(outPath) != "" {
		return filepath.Abs(outPath)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	projectRoot, err := resolveProjectRoot(workingDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(projectRoot, defaultOutputFile), nil
}

func resolveProjectRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %s: %w", start, err)
	}

	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("unable to locate project root from working directory")
		}

		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func buildHeaders(authToken string, extra headerFlags) http.Header {
	headers := http.Header(extra)
	if authToken != "" && headers.Get("Authorization") == "" {
		headers.Set("Authorization", "Bearer "+authToken)
	}

	return headers
}

func writeFileAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".fetchschema-*.graphql")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

type headerFlags map[string][]string

func (h headerFlags) Set(input string) error {
	name, value, ok := strings.Cut(input, "=")
	if !ok || strings.TrimSpace(name) == "" {
		return errors.New("header must be provided as key=value")
	}

	h[name] = append(h[name], value)
	return nil
}

func (h *headerFlags) String() string {
	var items []string
	for name, values := range *h {
		items = append(items, fmt.Sprintf("%s=%s", name, strings.Join(values, ",")))
	}

	return strings.Join(items, ",")
}
