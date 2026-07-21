package catalog

import "testing"

// TestKindTableIsTotal proves the descriptor table is exact-size and complete:
// every valid Kind has a non-empty name, a valid category, and a valid
// disposition, with unique names. A Kind added to the enum without a descriptor
// leaves a zero-value entry that fails here.
func TestKindTableIsTotal(t *testing.T) {
	all := All()
	if got, want := len(all), kindCount; got != want {
		t.Fatalf("All() returned %d kinds, want %d", got, want)
	}
	seenNames := map[string]Kind{}
	for _, k := range all {
		if !k.Valid() {
			t.Errorf("All() yielded invalid kind %d", uint16(k))
			continue
		}
		if k.Name() == "" {
			t.Errorf("kind %d has empty descriptor name", uint16(k))
		}
		if prior, dup := seenNames[k.Name()]; dup {
			t.Errorf("kinds %d and %d share name %q", uint16(prior), uint16(k), k.Name())
		}
		seenNames[k.Name()] = k
		if !k.Category().Valid() {
			t.Errorf("kind %s has invalid category", k)
		}
		if !k.Disposition().Valid() {
			t.Errorf("kind %s has invalid disposition", k)
		}
	}
}

// TestKindSentinelsInvalid proves the boundary values are not usable kinds.
func TestKindSentinelsInvalid(t *testing.T) {
	if KindInvalid.Valid() {
		t.Error("KindInvalid must not be valid")
	}
	if (Kind(kindCount + 1)).Valid() {
		t.Error("a value past kindCount must not be valid")
	}
	if got := KindInvalid.Name(); got != "" {
		t.Errorf("KindInvalid.Name() = %q, want empty", got)
	}
}

// pinnedIDs freezes every construct's permanent integer identity. It is an
// independent restatement of the enum: reordering, inserting, or renumbering a
// Kind changes a value here and fails the test, enforcing stable identities.
var pinnedIDs = map[Kind]uint16{
	KindBadExpr: 1, KindIdent: 2, KindEllipsis: 3, KindBasicLit: 4, KindFuncLit: 5,
	KindCompositeLit: 6, KindParenExpr: 7, KindSelectorExpr: 8, KindIndexExpr: 9,
	KindIndexListExpr: 10, KindSliceExpr: 11, KindTypeAssertExpr: 12, KindCallExpr: 13,
	KindStarExpr: 14, KindUnaryExpr: 15, KindBinaryExpr: 16, KindKeyValueExpr: 17,
	KindArrayType: 18, KindStructType: 19, KindFuncType: 20, KindInterfaceType: 21,
	KindMapType: 22, KindChanType: 23, KindBadStmt: 24, KindDeclStmt: 25,
	KindEmptyStmt: 26, KindLabeledStmt: 27, KindExprStmt: 28, KindSendStmt: 29,
	KindIncDecStmt: 30, KindAssignStmt: 31, KindGoStmt: 32, KindDeferStmt: 33,
	KindReturnStmt: 34, KindBranchStmt: 35, KindBlockStmt: 36, KindIfStmt: 37,
	KindCaseClause: 38, KindSwitchStmt: 39, KindTypeSwitchStmt: 40, KindCommClause: 41,
	KindSelectStmt: 42, KindForStmt: 43, KindRangeStmt: 44, KindBadDecl: 45,
	KindGenDecl: 46, KindFuncDecl: 47, KindImportSpec: 48, KindValueSpec: 49,
	KindTypeSpec: 50, KindFile: 51, KindComment: 52, KindCommentGroup: 53,
	KindField: 54, KindFieldList: 55, KindDirective: 56, KindPackage: 57,
}

// TestKindIDsArePinned proves construct identities are stable and explicit.
func TestKindIDsArePinned(t *testing.T) {
	if len(pinnedIDs) != kindCount {
		t.Fatalf("pinned IDs cover %d kinds, want %d", len(pinnedIDs), kindCount)
	}
	seen := map[uint16]Kind{}
	for k, want := range pinnedIDs {
		if uint16(k) != want {
			t.Errorf("kind %s has id %d, pinned to %d", k, uint16(k), want)
		}
		if prior, dup := seen[want]; dup {
			t.Errorf("id %d assigned to both %s and %s", want, prior, k)
		}
		seen[want] = k
	}
}

// TestCategoryClosed proves the category enum is closed with a terminal
// sentinel and named members.
func TestCategoryClosed(t *testing.T) {
	for c := CategoryInvalid + 1; c < numCategories; c++ {
		if !c.Valid() || categoryNames[c] == "" {
			t.Errorf("category %d is not a valid named member", uint8(c))
		}
	}
	if CategoryInvalid.Valid() || numCategories.Valid() {
		t.Error("category sentinels must not be valid")
	}
}

// TestDispositionClosed proves the disposition enum is closed and named, and
// that ast.Package is the deprecated form.
func TestDispositionClosed(t *testing.T) {
	for d := DispositionInvalid + 1; d < numDispositions; d++ {
		if !d.Valid() || dispositionNames[d] == "" {
			t.Errorf("disposition %d is not a valid named member", uint8(d))
		}
	}
	if DispositionInvalid.Valid() || numDispositions.Valid() {
		t.Error("disposition sentinels must not be valid")
	}
	if KindPackage.Disposition() != DispositionDeprecated {
		t.Errorf("KindPackage disposition = %s, want deprecated", KindPackage.Disposition())
	}
	if KindDirective.Disposition() != DispositionActive {
		t.Errorf("KindDirective disposition = %s, want active", KindDirective.Disposition())
	}
}
