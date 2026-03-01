package i18n

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"blog/framework/i18n/keygen"
)

const MessagesDir = "messages"

func DiscoverMessageFiles(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, MessagesDir)
	if err != nil {
		return nil, fmt.Errorf("read messages directory %q: %w", MessagesDir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if entry.IsDir() {
			return nil, fmt.Errorf("messages directory %q must not contain subdirectories: %q", MessagesDir, name)
		}
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			return nil, fmt.Errorf("messages directory %q must contain only json files: %q", MessagesDir, name)
		}
		files = append(files, path.Join(MessagesDir, name))
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("messages directory %q has no json files", MessagesDir)
	}

	return files, nil
}

func ValidateMessageKeyParity(fsys fs.FS, files []string, expectedKeys []string) error {
	expectedKeySet := buildExpectedKeySet(expectedKeys)
	if len(expectedKeySet) == 0 {
		return fmt.Errorf("expected key set is empty")
	}

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		content, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("read locale file %q: %w", file, err)
		}

		messages, err := keygen.ParseCanonical(content)
		if err != nil {
			return fmt.Errorf("parse locale file %q: %w", file, err)
		}

		localeKeys := make(map[string]struct{}, len(messages))
		for _, message := range messages {
			localeKeys[message.ID] = struct{}{}
		}

		missing := make([]string, 0, 4)
		for key := range expectedKeySet {
			if _, ok := localeKeys[key]; !ok {
				missing = append(missing, key)
			}
		}
		sort.Strings(missing)

		extra := make([]string, 0, 4)
		for key := range localeKeys {
			if _, ok := expectedKeySet[key]; !ok {
				extra = append(extra, key)
			}
		}
		sort.Strings(extra)

		if len(missing) > 0 || len(extra) > 0 {
			return fmt.Errorf(
				"locale %q key parity mismatch: missing=%v extra=%v",
				localeFromPath(file),
				missing,
				extra,
			)
		}
	}

	return nil
}

func buildExpectedKeySet(keys []string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = struct{}{}
	}
	return out
}

func localeFromPath(pathValue string) string {
	trimmed := strings.TrimSpace(pathValue)
	fileName := trimmed
	if slash := strings.LastIndex(trimmed, "/"); slash >= 0 {
		fileName = trimmed[slash+1:]
	}

	const prefix = "active."
	const suffix = ".json"
	if strings.HasPrefix(fileName, prefix) && strings.HasSuffix(fileName, suffix) {
		return strings.TrimSuffix(strings.TrimPrefix(fileName, prefix), suffix)
	}
	return fileName
}
