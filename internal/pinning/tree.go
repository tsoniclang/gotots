package pinning

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strconv"
	"strings"
)

// TreeEntry is one blob in the pinned commit tree.
type TreeEntry struct {
	OID  string // object ID in the repository's object format
	Path string // slash-separated, relative to the checkout root
}

// Tree is the complete tracked-file manifest of the pinned commit.
type Tree struct {
	Entries map[string]TreeEntry // path -> entry (blobs only)
	// Submodules maps gitlink paths to their pinned commit IDs.
	Submodules map[string]string
	// objectFormat is "sha1" or "sha256".
	objectFormat string
}

func loadTree(dir string) (*Tree, error) {
	objectFormat, err := git(dir, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return nil, fmt.Errorf("unsupported git object format %q", objectFormat)
	}
	output, err := git(dir, "ls-tree", "-r", "-z", "HEAD")
	if err != nil {
		return nil, err
	}
	tree := &Tree{
		Entries:      map[string]TreeEntry{},
		Submodules:   map[string]string{},
		objectFormat: objectFormat,
	}
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		// Format: <mode> <type> <oid>\t<path>
		meta, path, ok := strings.Cut(record, "\t")
		if !ok {
			return nil, fmt.Errorf("unexpected ls-tree record %q", record)
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree record %q", record)
		}
		mode, objectType, oid := fields[0], fields[1], fields[2]
		switch objectType {
		case "blob":
			tree.Entries[path] = TreeEntry{OID: oid, Path: path}
		case "commit":
			tree.Submodules[path] = oid
		default:
			return nil, fmt.Errorf("unexpected ls-tree object type %q for %s (mode %s)", objectType, path, mode)
		}
	}
	return tree, nil
}

// VerifyFile proves that the given content is byte-identical to the blob
// recorded for path in the pinned commit tree. This catches files that git
// status does not report, such as ignored files injected into package
// directories.
func (t *Tree) VerifyFile(path string, content []byte) error {
	entry, ok := t.Entries[path]
	if !ok {
		return fmt.Errorf("file %s is not tracked by the pinned commit; refusing untracked source", path)
	}
	var h hash.Hash
	switch t.objectFormat {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	}
	h.Write([]byte("blob " + strconv.Itoa(len(content))))
	h.Write([]byte{0})
	h.Write(content)
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != entry.OID {
		return fmt.Errorf("file %s content (blob %s) does not match the pinned commit blob %s", path, actual, entry.OID)
	}
	return nil
}

// Has reports whether path is a tracked blob in the pinned commit.
func (t *Tree) Has(path string) bool {
	_, ok := t.Entries[path]
	return ok
}

// Files returns every tracked path, sorted.
func (t *Tree) Files() []string {
	files := make([]string, 0, len(t.Entries))
	for path := range t.Entries {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}
