package structure

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// TransientIndex is the short-lived syntax index shared by later Stage-1
// structural consumers. It never enters Graph or any finalized artifact.
type TransientIndex struct {
	definitions map[identity.DefinitionID]ast.Node
	checked     map[identity.DefinitionID]ast.Node
	entries     map[identity.DefinitionID][]ast.Node
	files       map[identity.FileID]*source.LoadedFile
	checkedFset *token.FileSet
}

func newTransientIndex(universe *source.Universe) *TransientIndex {
	return &TransientIndex{
		definitions: map[identity.DefinitionID]ast.Node{},
		checked:     map[identity.DefinitionID]ast.Node{},
		entries:     map[identity.DefinitionID][]ast.Node{},
		files:       map[identity.FileID]*source.LoadedFile{},
		checkedFset: universe.Fset(),
	}
}

func (i *TransientIndex) DefinitionNode(
	id identity.DefinitionID,
) (ast.Node, bool) {
	node, ok := i.definitions[id]
	return node, ok
}

func (i *TransientIndex) CheckedDefinitionNode(
	id identity.DefinitionID,
) (ast.Node, bool) {
	node, ok := i.checked[id]
	return node, ok
}

func (i *TransientIndex) CheckedFileSet() *token.FileSet {
	return i.checkedFset
}

func (i *TransientIndex) ExecutionEntryNodes(
	id identity.DefinitionID,
) []ast.Node {
	return append([]ast.Node(nil), i.entries[id]...)
}

func (i *TransientIndex) File(
	id identity.FileID,
) (*source.LoadedFile, bool) {
	file, ok := i.files[id]
	return file, ok
}
