package tsoniccore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func installPackage(root, namespace, name string, files map[string]string) error {
	directory := root
	for _, component := range []string{"node_modules", namespace, name} {
		directory = filepath.Join(directory, component)
		info, err := os.Lstat(directory)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect resolution fixture directory %s: %w", directory, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("resolution fixture requires an owned directory: %s", directory)
		}
	}
	names := make([]string, 0, len(files))
	for filename := range files {
		if filename != filepath.Base(filename) || filename == "." || filename == ".." {
			return fmt.Errorf("invalid resolution fixture filename: %s", filename)
		}
		names = append(names, filename)
	}
	sort.Strings(names)
	for _, filename := range names {
		path := filepath.Join(directory, filename)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect resolution fixture file %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("resolution fixture refuses a non-regular file: %s", path)
		}
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create resolution fixture directory: %w", err)
	}
	for _, filename := range names {
		if err := os.WriteFile(filepath.Join(directory, filename), []byte(files[filename]), 0o600); err != nil {
			return fmt.Errorf("install resolution fixture %s: %w", filename, err)
		}
	}
	return nil
}
