package catalog

import "testing"

// TestEdgeTableIsTotal proves every edge descriptor is complete: non-empty
// name and field, valid active parent kind, valid role, unique names, and
// per-kind ID order equal to visit order (ascending).
func TestEdgeTableIsTotal(t *testing.T) {
	names := map[string]Edge{}
	for _, e := range AllEdges() {
		if e.Name() == "" || e.Field() == "" {
			t.Errorf("edge %d has empty name/field", uint16(e))
			continue
		}
		if prior, dup := names[e.Name()]; dup {
			t.Errorf("edges %d and %d share name %q", uint16(prior), uint16(e), e.Name())
		}
		names[e.Name()] = e
		if !e.Parent().Valid() {
			t.Errorf("edge %s has invalid parent", e)
		}
		if e.Parent().Disposition() != DispositionActive {
			t.Errorf("edge %s hangs from non-active kind %s", e, e.Parent())
		}
		if !e.Role().Valid() {
			t.Errorf("edge %s has invalid role", e)
		}
	}
	for _, kind := range All() {
		edges := EdgesOf(kind)
		for i := 1; i < len(edges); i++ {
			if edges[i-1] >= edges[i] {
				t.Errorf("kind %s edge order not ascending: %s >= %s", kind, edges[i-1], edges[i])
			}
		}
	}
}

// pinnedEdgeIDs freezes every edge's permanent identity, independently
// restated so a renumber or reorder fails.
var pinnedEdgeIDs = map[string]uint16{
	"Ellipsis.Elt": 1, "FuncLit.Type": 2, "FuncLit.Body": 3, "CompositeLit.Type": 4,
	"CompositeLit.Elts": 5, "ParenExpr.X": 6, "SelectorExpr.X": 7, "SelectorExpr.Sel": 8,
	"IndexExpr.X": 9, "IndexExpr.Index": 10, "IndexListExpr.X": 11, "IndexListExpr.Indices": 12,
	"SliceExpr.X": 13, "SliceExpr.Low": 14, "SliceExpr.High": 15, "SliceExpr.Max": 16,
	"TypeAssertExpr.X": 17, "TypeAssertExpr.Type": 18, "CallExpr.Fun": 19, "CallExpr.Args": 20,
	"StarExpr.X": 21, "UnaryExpr.X": 22, "BinaryExpr.X": 23, "BinaryExpr.Y": 24,
	"KeyValueExpr.Key": 25, "KeyValueExpr.Value": 26, "ArrayType.Len": 27, "ArrayType.Elt": 28,
	"StructType.Fields": 29, "FuncType.TypeParams": 30, "FuncType.Params": 31, "FuncType.Results": 32,
	"InterfaceType.Methods": 33, "MapType.Key": 34, "MapType.Value": 35, "ChanType.Value": 36,
	"DeclStmt.Decl": 37, "LabeledStmt.Label": 38, "LabeledStmt.Stmt": 39, "ExprStmt.X": 40,
	"SendStmt.Chan": 41, "SendStmt.Value": 42, "IncDecStmt.X": 43, "AssignStmt.Lhs": 44,
	"AssignStmt.Rhs": 45, "GoStmt.Call": 46, "DeferStmt.Call": 47, "ReturnStmt.Results": 48,
	"BranchStmt.Label": 49, "BlockStmt.List": 50, "IfStmt.Init": 51, "IfStmt.Cond": 52,
	"IfStmt.Body": 53, "IfStmt.Else": 54, "CaseClause.List": 55, "CaseClause.Body": 56,
	"SwitchStmt.Init": 57, "SwitchStmt.Tag": 58, "SwitchStmt.Body": 59, "TypeSwitchStmt.Init": 60,
	"TypeSwitchStmt.Assign": 61, "TypeSwitchStmt.Body": 62, "CommClause.Comm": 63, "CommClause.Body": 64,
	"SelectStmt.Body": 65, "ForStmt.Init": 66, "ForStmt.Cond": 67, "ForStmt.Post": 68,
	"ForStmt.Body": 69, "RangeStmt.Key": 70, "RangeStmt.Value": 71, "RangeStmt.X": 72,
	"RangeStmt.Body": 73, "ImportSpec.Doc": 74, "ImportSpec.Name": 75, "ImportSpec.Path": 76,
	"ImportSpec.Comment": 77, "ValueSpec.Doc": 78, "ValueSpec.Names": 79, "ValueSpec.Type": 80,
	"ValueSpec.Values": 81, "ValueSpec.Comment": 82, "TypeSpec.Doc": 83, "TypeSpec.Name": 84,
	"TypeSpec.TypeParams": 85, "TypeSpec.Type": 86, "TypeSpec.Comment": 87, "GenDecl.Doc": 88,
	"GenDecl.Specs": 89, "FuncDecl.Doc": 90, "FuncDecl.Recv": 91, "FuncDecl.Name": 92,
	"FuncDecl.Type": 93, "FuncDecl.Body": 94, "File.Doc": 95, "File.Name": 96,
	"File.Decls": 97, "CommentGroup.List": 98, "Field.Doc": 99, "Field.Names": 100,
	"Field.Type": 101, "Field.Tag": 102, "Field.Comment": 103, "FieldList.List": 104,
}

// TestEdgeIDsArePinned proves edge identities are stable and explicit.
func TestEdgeIDsArePinned(t *testing.T) {
	if len(pinnedEdgeIDs) != edgeCount {
		t.Fatalf("pinned edge IDs cover %d, want %d", len(pinnedEdgeIDs), edgeCount)
	}
	for _, e := range AllEdges() {
		want, pinned := pinnedEdgeIDs[e.Name()]
		if !pinned {
			t.Errorf("edge %s (%d) has no pinned identity", e, uint16(e))
			continue
		}
		if uint16(e) != want {
			t.Errorf("edge %s has id %d, pinned to %d", e, uint16(e), want)
		}
	}
}

// TestRoleTableIsTotal proves every role is named and the sentinels are
// invalid; role identities are pinned by the explicit constants themselves and
// the count check.
func TestRoleTableIsTotal(t *testing.T) {
	if got := len(AllRoles()); got != roleCount {
		t.Fatalf("AllRoles() = %d roles, want %d", got, roleCount)
	}
	seen := map[string]Role{}
	for _, r := range AllRoles() {
		if !r.Valid() {
			t.Errorf("role %d invalid", uint16(r))
		}
		if roleNames[r] == "" {
			t.Errorf("role %d unnamed", uint16(r))
		}
		if prior, dup := seen[roleNames[r]]; dup {
			t.Errorf("roles %d and %d share name %q", uint16(prior), uint16(r), roleNames[r])
		}
		seen[roleNames[r]] = r
	}
	if RoleInvalid.Valid() || Role(roleCount+1).Valid() {
		t.Error("role sentinels must not be valid")
	}
	// Every role is assigned by at least one edge — no orphan vocabulary.
	used := map[Role]bool{}
	for _, e := range AllEdges() {
		used[e.Role()] = true
	}
	for _, r := range AllRoles() {
		if !used[r] {
			t.Errorf("role %s is not assigned by any edge", r)
		}
	}
}

// TestExcludedFieldsHaveReasons proves the exclusion record is complete and
// names only valid kinds.
func TestExcludedFieldsHaveReasons(t *testing.T) {
	for _, excluded := range ExcludedFields() {
		if !excluded.Kind.Valid() || excluded.Field == "" || excluded.Reason == "" {
			t.Errorf("incomplete exclusion record %+v", excluded)
		}
	}
}
