package toolchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

const rootManifestSchema = 1

type rootContract struct {
	root           string
	toolDirectory  string
	container      string
	rootDigest     string
	manifestDigest string
	rootInfo       fs.FileInfo
	sealInfo       fs.FileInfo
	census         *rootInspectionCensus
}

type rootInspectionCensus struct {
	fullWalks atomic.Uint64
}

type rootSeal struct {
	Schema         int    `json:"schema"`
	RootDigest     string `json:"rootDigest"`
	ManifestDigest string `json:"manifestDigest"`
}

func canonicalDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	return filepath.Clean(canonical), nil
}

func withinRoot(root string, selected string) bool {
	relative, err := filepath.Rel(root, selected)
	return err == nil && relative != ".." &&
		!filepath.IsAbs(relative) &&
		(relative == "." || !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func (c rootContract) Valid() bool {
	return c.root != "" && c.toolDirectory != "" && c.container != "" &&
		len(c.rootDigest) == sha256.Size*2 &&
		len(c.manifestDigest) == sha256.Size*2 &&
		c.rootInfo != nil && c.sealInfo != nil && c.census != nil
}

func (c rootContract) VerifyHandle() error {
	if !c.Valid() {
		return fmt.Errorf("sealed Go root handle is invalid")
	}
	rootInfo, err := os.Stat(c.root)
	if err != nil {
		return fmt.Errorf("inspect sealed Go root: %w", err)
	}
	if !rootInfo.IsDir() || !os.SameFile(c.rootInfo, rootInfo) {
		return fmt.Errorf("sealed Go root handle changed")
	}
	sealPath := filepath.Join(c.container, "seal.json")
	sealInfo, err := os.Stat(sealPath)
	if err != nil {
		return fmt.Errorf("inspect sealed Go root seal: %w", err)
	}
	if !sealInfo.Mode().IsRegular() || !os.SameFile(c.sealInfo, sealInfo) {
		return fmt.Errorf("sealed Go root seal handle changed")
	}
	seal, err := readRootSeal(sealPath)
	if err != nil {
		return err
	}
	if seal.RootDigest != c.rootDigest || seal.ManifestDigest != c.manifestDigest {
		return fmt.Errorf("sealed Go root seal identity changed")
	}
	return nil
}

func (c rootContract) VerifyComplete() error {
	if err := c.VerifyHandle(); err != nil {
		return err
	}
	_, _, err := verifyRootSnapshot(c.container, nil, c.rootDigest, c.census)
	return err
}

func (c rootContract) FullWalkCount() uint64 {
	if c.census == nil {
		return 0
	}
	return c.census.fullWalks.Load()
}

func readRootSeal(path string) (rootSeal, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return rootSeal{}, fmt.Errorf("read sealed Go root seal: %w", err)
	}
	var seal rootSeal
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&seal); err != nil {
		return rootSeal{}, fmt.Errorf("decode sealed Go root seal: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return rootSeal{}, fmt.Errorf("decode sealed Go root seal: %w", err)
	}
	if seal.Schema != rootManifestSchema || len(seal.RootDigest) != sha256.Size*2 ||
		len(seal.ManifestDigest) != sha256.Size*2 {
		return rootSeal{}, fmt.Errorf("sealed Go root seal is invalid")
	}
	return seal, nil
}

func digestBytes(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func executableFileName(name string) string {
	if filepath.Ext(name) == "" && os.PathSeparator == '\\' {
		return name + ".exe"
	}
	return name
}
