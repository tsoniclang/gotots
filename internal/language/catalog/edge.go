package catalog

import "fmt"

// Edge is the closed identity of one parent→child edge in the Go syntax
// grammar: a node-bearing field of one construct kind. Values are explicit and
// permanent; TestEdgeIDsArePinned freezes the mapping. The per-kind edge order
// is the source visit order, and the traversal is driven by this catalog — the
// visitor holds no private edge knowledge.
type Edge uint16

// edgeDescriptor is one edge's record: owning parent kind, the toolchain
// struct field it binds, the grammatical role the parent assigns across it,
// and whether the field is a node sequence.
type edgeDescriptor struct {
	name   string
	parent Kind
	field  string
	role   Role
	list   bool
}

// Valid reports whether e names an edge in the catalog.
func (e Edge) Valid() bool { return e >= 1 && e <= edgeCount }

// Name is the stable descriptive name, e.g. "FuncDecl.Body".
func (e Edge) Name() string {
	if !e.Valid() {
		return ""
	}
	return edges[e].name
}

// Parent is the owning construct kind.
func (e Edge) Parent() Kind {
	if !e.Valid() {
		return KindInvalid
	}
	return edges[e].parent
}

// Field is the toolchain struct field name the edge binds.
func (e Edge) Field() string {
	if !e.Valid() {
		return ""
	}
	return edges[e].field
}

// Role is the grammatical role the parent assigns across this edge.
func (e Edge) Role() Role {
	if !e.Valid() {
		return RoleInvalid
	}
	return edges[e].role
}

// IsList reports whether the edge binds a node sequence.
func (e Edge) IsList() bool { return e.Valid() && edges[e].list }

// String renders e for diagnostics and reports.
func (e Edge) String() string {
	if name := e.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.Edge(%d)", uint16(e))
}

// EdgeByName reconstructs an edge from its canonical name (e.g. "FuncDecl.Body"),
// for decoding a serialized reference back to the typed catalog edge.
func EdgeByName(name string) (Edge, error) {
	for id := Edge(1); id <= edgeCount; id++ {
		if id.Name() == name {
			return id, nil
		}
	}
	return Edge(0), fmt.Errorf("no catalog edge named %q", name)
}

// AllEdges returns every valid Edge in ascending identity order.
func AllEdges() []Edge {
	out := make([]Edge, 0, edgeCount)
	for id := 1; id <= edgeCount; id++ {
		out = append(out, Edge(id))
	}
	return out
}

// edgesByKind indexes the per-kind ordered edge lists once. Edge IDs are
// pinned in visit order within each kind, so ascending order is visit order.
var edgesByKind = func() [kindCount + 1][]Edge {
	var byKind [kindCount + 1][]Edge
	for id := Edge(1); id <= edgeCount; id++ {
		parent := edges[id].parent
		byKind[parent] = append(byKind[parent], id)
	}
	return byKind
}()

// EdgesOf returns the ordered child edges of one construct kind. A leaf kind
// returns nil.
func EdgesOf(kind Kind) []Edge {
	if !kind.Valid() {
		return nil
	}
	return edgesByKind[kind]
}

// ExcludedField is one toolchain node-bearing field deliberately outside the
// edge grammar, with its recorded reason. The reconciliation gate accepts a
// node-bearing field only when it is a cataloged edge or appears here.
type ExcludedField struct {
	Kind   Kind
	Field  string
	Reason string
}

// ExcludedFields is the closed exclusion record for toolchain fields that hold
// nodes but are not grammar edges.
func ExcludedFields() []ExcludedField {
	return []ExcludedField{
		{KindFile, "Imports", "derived index of the import specs owned by GenDecl edges"},
		{KindFile, "Unresolved", "deprecated toolchain resolution artifact"},
		{KindFile, "Comments", "comment index; groups attach through Doc/Comment edges, directives through the directive inventory"},
	}
}
