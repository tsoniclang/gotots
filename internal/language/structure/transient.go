package structure

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// TransientIndex is the short-lived syntax index shared by later Stage-1
// structural consumers. It never enters Graph or any finalized artifact.
type TransientIndex struct {
	definitions  map[identity.DefinitionID]ast.Node
	checked      map[identity.DefinitionID]ast.Node
	synthetic    map[identity.DefinitionID]ast.Node
	entries      map[identity.DefinitionID][]ast.Node
	files        map[identity.FileID]*source.LoadedFile
	structural   map[identity.OccurrenceID]ast.Node
	executable   map[identity.OccurrenceID]ast.Node
	structuralID map[ast.Node]identity.OccurrenceID
	executableID map[ast.Node]identity.OccurrenceID
	counterparts map[ast.Node]ast.Node
	originals    map[ast.Node]ast.Node
	unmapped     map[identity.OccurrenceID]bool
	checkedFset  *token.FileSet
	checkedFiles map[*token.File]identity.FileID
}

func newTransientIndex(
	universe *source.Universe,
) (*TransientIndex, error) {
	index := &TransientIndex{
		definitions:  map[identity.DefinitionID]ast.Node{},
		checked:      map[identity.DefinitionID]ast.Node{},
		synthetic:    map[identity.DefinitionID]ast.Node{},
		entries:      map[identity.DefinitionID][]ast.Node{},
		files:        map[identity.FileID]*source.LoadedFile{},
		structural:   map[identity.OccurrenceID]ast.Node{},
		executable:   map[identity.OccurrenceID]ast.Node{},
		structuralID: map[ast.Node]identity.OccurrenceID{},
		executableID: map[ast.Node]identity.OccurrenceID{},
		counterparts: map[ast.Node]ast.Node{},
		originals:    map[ast.Node]ast.Node{},
		unmapped:     map[identity.OccurrenceID]bool{},
		checkedFset:  universe.Fset(),
		checkedFiles: map[*token.File]identity.FileID{},
	}
	for _, pkg := range universe.Packages() {
		for tokenFile, fileID := range pkg.CheckerFileIdentities() {
			if existing := index.checkedFiles[tokenFile]; !existing.IsZero() &&
				existing != fileID {
				return nil, fmt.Errorf(
					"checker token file maps to source identities %s and %s",
					existing, fileID,
				)
			}
			index.checkedFiles[tokenFile] = fileID
		}
	}
	return index, nil
}

// MarkCheckedUnmapped records that a selected cgo executable occurrence has
// no same-construct checked counterpart and therefore requires an explicit
// checked-view transformation boundary in Stage 2.
func (i *TransientIndex) MarkCheckedUnmapped(
	id identity.OccurrenceID,
) {
	i.unmapped[id] = true
}

func (i *TransientIndex) CheckedUnmapped(
	id identity.OccurrenceID,
) bool {
	return i.unmapped[id]
}

func (i *TransientIndex) bindCheckedCounterparts(
	original ast.Node,
	checked ast.Node,
) error {
	originalKind, err := Classify(original)
	if err != nil {
		return err
	}
	checkedKind, err := Classify(checked)
	if err != nil {
		return err
	}
	if originalKind != checkedKind {
		return fmt.Errorf(
			"checked counterpart changes %s to %s",
			originalKind, checkedKind,
		)
	}
	if existing := i.counterparts[original]; existing != nil && existing != checked {
		return fmt.Errorf(
			"source node has conflicting checked counterparts",
		)
	}
	if existing := i.originals[checked]; existing != nil && existing != original {
		return fmt.Errorf(
			"checked node has conflicting source counterparts",
		)
	}
	i.counterparts[original] = checked
	i.originals[checked] = original
	originalChildren, err := Children(original, originalKind)
	if err != nil {
		return err
	}
	checkedChildren, err := Children(checked, checkedKind)
	if err != nil {
		return err
	}
	type childKey struct {
		edge    catalog.Edge
		ordinal int
	}
	byKey := map[childKey]Child{}
	for _, child := range checkedChildren {
		byKey[childKey{
			edge: child.Edge(), ordinal: child.Ordinal(),
		}] = child
	}
	for _, child := range originalChildren {
		counterpart, present := byKey[childKey{
			edge: child.Edge(), ordinal: child.Ordinal(),
		}]
		if !present {
			continue
		}
		originalChildKind, err := Classify(child.Node())
		if err != nil {
			return err
		}
		checkedChildKind, err := Classify(counterpart.Node())
		if err != nil {
			return err
		}
		if originalChildKind != checkedChildKind {
			continue
		}
		if err := i.bindCheckedCounterparts(
			child.Node(), counterpart.Node(),
		); err != nil {
			return err
		}
	}
	return nil
}

// CheckedCounterpart returns the exact checker-facing node paired to one
// original cgo source node by definition origin and catalog edge/ordinal.
func (i *TransientIndex) CheckedCounterpart(
	original ast.Node,
) (ast.Node, bool) {
	node, present := i.counterparts[original]
	return node, present
}

func (i *TransientIndex) bindCheckedFile(
	file *source.LoadedFile,
) error {
	syntax := file.CheckedSyntax()
	if syntax == nil {
		return nil
	}
	tokenFile := i.checkedFset.File(syntax.Pos())
	if tokenFile == nil {
		return fmt.Errorf(
			"checked source %s is absent from the checker file set",
			file.ID(),
		)
	}
	if existing := i.checkedFiles[tokenFile]; !existing.IsZero() &&
		existing != file.ID() {
		return fmt.Errorf(
			"checker file maps to source identities %s and %s",
			existing, file.ID(),
		)
	}
	i.checkedFiles[tokenFile] = file.ID()
	return nil
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

func (i *TransientIndex) SyntheticDefinitionNode(
	id identity.DefinitionID,
) (ast.Node, bool) {
	node, ok := i.synthetic[id]
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

func (i *TransientIndex) bindStructuralOccurrence(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	return i.bindOccurrence(
		i.structural, i.structuralID, id, node,
	)
}

// BindExecutableOccurrence records the exact node already visited by the
// Stage-1 executable traversal. It does not traverse or classify a new path.
func (i *TransientIndex) BindExecutableOccurrence(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	return i.bindOccurrence(
		i.executable, i.executableID, id, node,
	)
}

func (i *TransientIndex) bindOccurrence(
	target map[identity.OccurrenceID]ast.Node,
	reverse map[ast.Node]identity.OccurrenceID,
	id identity.OccurrenceID,
	node ast.Node,
) error {
	if id.IsZero() || node == nil {
		return fmt.Errorf(
			"transient occurrence binding requires identity and node",
		)
	}
	kind, err := Classify(node)
	if err != nil {
		return err
	}
	if uint16(kind) != id.KindID() {
		return fmt.Errorf(
			"transient occurrence %s has node kind %s",
			id, kind,
		)
	}
	if existing, present := target[id]; present &&
		existing != node {
		return fmt.Errorf(
			"transient occurrence %s has conflicting nodes", id,
		)
	}
	if existing, present := reverse[node]; present &&
		existing != id {
		return fmt.Errorf(
			"transient node has conflicting occurrences %s and %s",
			existing, id,
		)
	}
	target[id] = node
	reverse[node] = id
	return nil
}

// OccurrenceNode returns the checker-facing node for one locally retained
// occurrence. Executable checked-view evidence takes precedence over the
// physical structural node for the same canonical identity.
func (i *TransientIndex) OccurrenceNode(
	id identity.OccurrenceID,
) (ast.Node, bool) {
	if node, present := i.executable[id]; present {
		return node, true
	}
	node, present := i.structural[id]
	if present {
		if checked := i.counterparts[node]; checked != nil {
			return checked, true
		}
	}
	return node, present
}

// OccurrenceID returns the canonical occurrence already assigned to one
// transient checker/source node. It never classifies or traverses the node.
func (i *TransientIndex) OccurrenceID(
	node ast.Node,
) (identity.OccurrenceID, bool) {
	if id, present := i.executableID[node]; present {
		return id, true
	}
	if id, present := i.structuralID[node]; present {
		return id, true
	}
	original := i.originals[node]
	id, present := i.structuralID[original]
	return id, present
}

// IdentifierOccurrence resolves a checker object's source position to the
// canonical identifier occurrence identity without retaining or traversing
// its surrounding syntax.
func (i *TransientIndex) IdentifierOccurrence(
	position token.Pos,
	name string,
) (identity.OccurrenceID, error) {
	if !position.IsValid() || name == "" {
		return identity.OccurrenceID{}, fmt.Errorf(
			"checker identifier requires position and name",
		)
	}
	tokenFile := i.checkedFset.File(position)
	fileID := i.checkedFiles[tokenFile]
	if tokenFile == nil || fileID.IsZero() {
		return identity.OccurrenceID{}, fmt.Errorf(
			"checker identifier %q has no canonical source file",
			name,
		)
	}
	start := tokenFile.Offset(position)
	span, err := identity.NewSpanID(
		fileID, start, start+len(name),
	)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return identity.NewOccurrenceID(
		span, uint16(catalog.KindIdent),
	)
}

func (i *TransientIndex) StructuralOccurrenceCount() int {
	return len(i.structural)
}

func (i *TransientIndex) ExecutableOccurrenceCount() int {
	return len(i.executable)
}
