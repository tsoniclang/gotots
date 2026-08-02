package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"
)

type information struct {
	name      string
	directory bool
}

func (i information) Name() string       { return i.name }
func (i information) Size() int64        { return 0 }
func (i information) Mode() fs.FileMode  { return 0 }
func (i information) ModTime() time.Time { return time.Time{} }
func (i information) IsDir() bool        { return i.directory }
func (i information) Sys() any           { return nil }

type entry struct {
	name      string
	directory bool
}

func (e entry) Name() string               { return e.name }
func (e entry) IsDir() bool                { return e.directory }
func (e entry) Type() fs.FileMode          { return 0 }
func (e entry) Info() (fs.FileInfo, error) { return information(e), nil }

type directFS struct{}

func (directFS) Open(string) (fs.File, error) { return nil, errors.New("unused") }

func (directFS) Stat(path string) (fs.FileInfo, error) {
	return information{name: path, directory: true}, nil
}

func (directFS) ReadDir(path string) ([]fs.DirEntry, error) {
	if path != "." {
		return nil, nil
	}
	return []fs.DirEntry{
		entry{name: "a"},
		entry{name: "skip", directory: true},
		entry{name: "z"},
	}, nil
}

type fallbackFS struct{}

func (fallbackFS) Open(path string) (fs.File, error) {
	return &fallbackFile{path: path}, nil
}

type fallbackFile struct {
	path string
}

func (f *fallbackFile) Stat() (fs.FileInfo, error) {
	return information{name: f.path, directory: true}, nil
}

func (*fallbackFile) Read([]byte) (int, error) { return 0, io.EOF }
func (*fallbackFile) Close() error             { return nil }

func (f *fallbackFile) ReadDir(int) ([]fs.DirEntry, error) {
	if f.path != "." {
		return nil, nil
	}
	return []fs.DirEntry{entry{name: "b"}}, nil
}

func main() {
	var syncPaths []string
	syncFailure := fs.WalkDir(directFS{}, ".", func(
		path string,
		_ fs.DirEntry,
		_ error,
	) error {
		syncPaths = append(syncPaths, path)
		if path == "skip" {
			return fs.SkipDir
		}
		return nil
	})
	var asyncPaths []string
	asyncFailure := fs.WalkDir(fallbackFS{}, ".", func(
		path string,
		_ fs.DirEntry,
		_ error,
	) error {
		asyncPaths = append(asyncPaths, path)
		if path == "b" {
			return errors.New("blocked")
		}
		return nil
	})
	fmt.Printf("sync:%s:%s\n", join(syncPaths), failureText(syncFailure))
	fmt.Printf("async:%s:%s\n", join(asyncPaths), failureText(asyncFailure))
}

func failureText(failure error) string {
	if failure == nil {
		return "ok"
	}
	return failure.Error()
}

func join(values []string) string {
	result := ""
	for index, value := range values {
		if index != 0 {
			result += ","
		}
		result += value
	}
	return result
}
