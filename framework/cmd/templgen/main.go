package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"blog/framework/templgen"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("value cannot be empty")
	}
	*m = append(*m, trimmed)
	return nil
}

func main() {
	var files multiFlag
	var paths multiFlag
	var basePath string

	flag.Var(&files, "file", "templ file to compile (repeatable)")
	flag.Var(&paths, "path", "directory to scan for .templ files (repeatable)")
	flag.StringVar(&basePath, "base", ".", "base path for relative filenames embedded in generated output")
	flag.Parse()

	if err := templgen.Run(templgen.Config{
		Files:    files,
		Paths:    paths,
		BasePath: basePath,
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "templgen: %v\n", err)
		os.Exit(1)
	}
}
