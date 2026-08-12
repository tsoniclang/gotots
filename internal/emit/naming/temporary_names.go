package naming

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type TemporarySnapshot struct {
	counters map[api.TemporaryKind]uint64
	names    map[string]struct{}
}

func (n *File) SnapshotTemporaries() TemporarySnapshot {
	snapshot := TemporarySnapshot{
		counters: make(map[api.TemporaryKind]uint64, len(n.temporaries)),
		names:    make(map[string]struct{}, len(n.generatedNames)),
	}
	for kind, value := range n.temporaries {
		snapshot.counters[kind] = value
	}
	for name := range n.generatedNames {
		snapshot.names[name] = struct{}{}
	}
	return snapshot
}

func (n *File) RestoreTemporaries(snapshot TemporarySnapshot) {
	n.temporaries = make(map[api.TemporaryKind]uint64, len(snapshot.counters))
	for kind, value := range snapshot.counters {
		n.temporaries[kind] = value
	}
	n.generatedNames = make(map[string]struct{}, len(snapshot.names))
	for name := range snapshot.names {
		n.generatedNames[name] = struct{}{}
	}
}

func (n *File) FinishTemporaryReplay(current TemporarySnapshot) {
	replayedNames := n.generatedNames
	n.RestoreTemporaries(current)
	for name := range replayedNames {
		n.generatedNames[name] = struct{}{}
	}
}

func (n *File) sourceNameExists(name string) bool {
	return (n.packageScope != nil && n.packageScope.Lookup(name) != nil) ||
		n.owner.hasSourceName(name)
}

func (n *File) hasImportName(name string) bool {
	_, exists := n.importNames[name]
	return exists
}

func (n *File) hasGeneratedName(name string) bool {
	_, exists := n.generatedNames[name]
	return exists
}

func (n *File) lexicalNameExists(name string) bool {
	return n.sourceNameExists(name) ||
		n.hasImportName(name) ||
		n.hasGeneratedName(name)
}

func (n *Owner) hasSourceName(name string) bool {
	_, exists := n.sourceNameBases[name]
	return exists
}

func (n *File) Temporary(kind api.TemporaryKind) (string, error) {
	prefix, err := api.TemporaryPrefix(kind)
	if err != nil {
		return "", err
	}
	for {
		index := n.temporaries[kind]
		n.temporaries[kind] = index + 1
		candidate := prefix + strconv.FormatUint(index, 10)
		if n.lexicalNameExists(candidate) {
			continue
		}
		if n.generatedNames == nil {
			n.generatedNames = make(map[string]struct{})
		}
		n.generatedNames[candidate] = struct{}{}
		return candidate, nil
	}
}
