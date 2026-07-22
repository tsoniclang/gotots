// Package catalog is the closed, target-independent catalog of Go language
// constructs for the selected Go toolchain. It names every grammatical form
// with a stable enum identity, a grammatical category, and a support
// disposition.
//
// The catalog is authoritative Go code, not parallel hand-maintained JSON. It
// imports no go/ast, go/types, or target package: the binding from concrete
// syntax to a catalog Kind lives in internal/language/analyze, the only other
// package permitted to touch the toolchain AST. This separation keeps the
// vocabulary independent of both the frontend and the target.
package catalog

import "fmt"

// Kind is the stable identity of one Go grammatical construct form. Values are
// explicit and permanent: a Kind's integer identity never changes, new kinds
// take the next free value, and TestKindIDsArePinned freezes the whole mapping
// so an insertion or reordering that shifts an identity fails.
type Kind uint16

// Explicit, permanent construct identities. Do not renumber; append only.
const (
	KindInvalid Kind = 0

	// Expressions.
	KindBadExpr        Kind = 1
	KindIdent          Kind = 2
	KindEllipsis       Kind = 3
	KindBasicLit       Kind = 4
	KindFuncLit        Kind = 5
	KindCompositeLit   Kind = 6
	KindParenExpr      Kind = 7
	KindSelectorExpr   Kind = 8
	KindIndexExpr      Kind = 9
	KindIndexListExpr  Kind = 10
	KindSliceExpr      Kind = 11
	KindTypeAssertExpr Kind = 12
	KindCallExpr       Kind = 13
	KindStarExpr       Kind = 14
	KindUnaryExpr      Kind = 15
	KindBinaryExpr     Kind = 16
	KindKeyValueExpr   Kind = 17

	// Type expressions.
	KindArrayType     Kind = 18
	KindStructType    Kind = 19
	KindFuncType      Kind = 20
	KindInterfaceType Kind = 21
	KindMapType       Kind = 22
	KindChanType      Kind = 23

	// Statements.
	KindBadStmt        Kind = 24
	KindDeclStmt       Kind = 25
	KindEmptyStmt      Kind = 26
	KindLabeledStmt    Kind = 27
	KindExprStmt       Kind = 28
	KindSendStmt       Kind = 29
	KindIncDecStmt     Kind = 30
	KindAssignStmt     Kind = 31
	KindGoStmt         Kind = 32
	KindDeferStmt      Kind = 33
	KindReturnStmt     Kind = 34
	KindBranchStmt     Kind = 35
	KindBlockStmt      Kind = 36
	KindIfStmt         Kind = 37
	KindCaseClause     Kind = 38
	KindSwitchStmt     Kind = 39
	KindTypeSwitchStmt Kind = 40
	KindCommClause     Kind = 41
	KindSelectStmt     Kind = 42
	KindForStmt        Kind = 43
	KindRangeStmt      Kind = 44

	// Declarations.
	KindBadDecl  Kind = 45
	KindGenDecl  Kind = 46
	KindFuncDecl Kind = 47

	// Specifications.
	KindImportSpec Kind = 48
	KindValueSpec  Kind = 49
	KindTypeSpec   Kind = 50

	// Structural nodes.
	KindFile         Kind = 51
	KindComment      Kind = 52
	KindCommentGroup Kind = 53
	KindField        Kind = 54
	KindFieldList    Kind = 55
	KindDirective    Kind = 56
	KindPackage      Kind = 57

	// kindCount is the highest assigned identity. Appended kinds increment it;
	// it sizes the descriptor table so a Kind added past the end fails to
	// compile until the table grows.
	kindCount = 57
)

// descriptor is the growing per-kind record. This slice of the catalog carries
// the stable name, grammatical category, and support disposition; later phases
// extend it with applicable roles, required typed evidence, and produced
// semantic operation, each behind its own closed enum.
type descriptor struct {
	name        string
	category    Category
	disposition Disposition
}

// descriptors is an exact-size table indexed by Kind. Its fixed length means a
// Kind added past the current end fails to compile, and a Kind added without a
// descriptor here leaves a zero-value (empty-name) entry that
// TestKindTableIsTotal rejects.
var descriptors = [kindCount + 1]descriptor{
	// Bad* are parser error-recovery forms: cataloged for total toolchain
	// reconciliation, never admissible in an admitted tree.
	KindBadExpr:        {"BadExpr", CategoryExpression, DispositionRecovery},
	KindIdent:          {"Ident", CategoryExpression, DispositionActive},
	KindEllipsis:       {"Ellipsis", CategoryExpression, DispositionActive},
	KindBasicLit:       {"BasicLit", CategoryExpression, DispositionActive},
	KindFuncLit:        {"FuncLit", CategoryExpression, DispositionActive},
	KindCompositeLit:   {"CompositeLit", CategoryExpression, DispositionActive},
	KindParenExpr:      {"ParenExpr", CategoryExpression, DispositionActive},
	KindSelectorExpr:   {"SelectorExpr", CategoryExpression, DispositionActive},
	KindIndexExpr:      {"IndexExpr", CategoryExpression, DispositionActive},
	KindIndexListExpr:  {"IndexListExpr", CategoryExpression, DispositionActive},
	KindSliceExpr:      {"SliceExpr", CategoryExpression, DispositionActive},
	KindTypeAssertExpr: {"TypeAssertExpr", CategoryExpression, DispositionActive},
	KindCallExpr:       {"CallExpr", CategoryExpression, DispositionActive},
	KindStarExpr:       {"StarExpr", CategoryExpression, DispositionActive},
	KindUnaryExpr:      {"UnaryExpr", CategoryExpression, DispositionActive},
	KindBinaryExpr:     {"BinaryExpr", CategoryExpression, DispositionActive},
	KindKeyValueExpr:   {"KeyValueExpr", CategoryExpression, DispositionActive},

	KindArrayType:     {"ArrayType", CategoryType, DispositionActive},
	KindStructType:    {"StructType", CategoryType, DispositionActive},
	KindFuncType:      {"FuncType", CategoryType, DispositionActive},
	KindInterfaceType: {"InterfaceType", CategoryType, DispositionActive},
	KindMapType:       {"MapType", CategoryType, DispositionActive},
	KindChanType:      {"ChanType", CategoryType, DispositionActive},

	KindBadStmt:        {"BadStmt", CategoryStatement, DispositionRecovery},
	KindDeclStmt:       {"DeclStmt", CategoryStatement, DispositionActive},
	KindEmptyStmt:      {"EmptyStmt", CategoryStatement, DispositionActive},
	KindLabeledStmt:    {"LabeledStmt", CategoryStatement, DispositionActive},
	KindExprStmt:       {"ExprStmt", CategoryStatement, DispositionActive},
	KindSendStmt:       {"SendStmt", CategoryStatement, DispositionActive},
	KindIncDecStmt:     {"IncDecStmt", CategoryStatement, DispositionActive},
	KindAssignStmt:     {"AssignStmt", CategoryStatement, DispositionActive},
	KindGoStmt:         {"GoStmt", CategoryStatement, DispositionActive},
	KindDeferStmt:      {"DeferStmt", CategoryStatement, DispositionActive},
	KindReturnStmt:     {"ReturnStmt", CategoryStatement, DispositionActive},
	KindBranchStmt:     {"BranchStmt", CategoryStatement, DispositionActive},
	KindBlockStmt:      {"BlockStmt", CategoryStatement, DispositionActive},
	KindIfStmt:         {"IfStmt", CategoryStatement, DispositionActive},
	KindCaseClause:     {"CaseClause", CategoryStatement, DispositionActive},
	KindSwitchStmt:     {"SwitchStmt", CategoryStatement, DispositionActive},
	KindTypeSwitchStmt: {"TypeSwitchStmt", CategoryStatement, DispositionActive},
	KindCommClause:     {"CommClause", CategoryStatement, DispositionActive},
	KindSelectStmt:     {"SelectStmt", CategoryStatement, DispositionActive},
	KindForStmt:        {"ForStmt", CategoryStatement, DispositionActive},
	KindRangeStmt:      {"RangeStmt", CategoryStatement, DispositionActive},

	KindBadDecl:  {"BadDecl", CategoryDeclaration, DispositionRecovery},
	KindGenDecl:  {"GenDecl", CategoryDeclaration, DispositionActive},
	KindFuncDecl: {"FuncDecl", CategoryDeclaration, DispositionActive},

	KindImportSpec: {"ImportSpec", CategorySpec, DispositionActive},
	KindValueSpec:  {"ValueSpec", CategorySpec, DispositionActive},
	KindTypeSpec:   {"TypeSpec", CategorySpec, DispositionActive},

	KindFile:         {"File", CategoryStructural, DispositionActive},
	KindComment:      {"Comment", CategoryStructural, DispositionActive},
	KindCommentGroup: {"CommentGroup", CategoryStructural, DispositionActive},
	KindField:        {"Field", CategoryStructural, DispositionActive},
	KindFieldList:    {"FieldList", CategoryStructural, DispositionActive},
	KindDirective:    {"Directive", CategoryStructural, DispositionActive},
	// ast.Package is deprecated in the toolchain and never produced by a file
	// parse; it is cataloged for total reconciliation but carries a deprecated
	// disposition so a later phase can reject rather than translate it.
	KindPackage: {"Package", CategoryStructural, DispositionDeprecated},
}

// Valid reports whether k names a construct in the catalog. KindInvalid and any
// value past kindCount are not valid.
func (k Kind) Valid() bool { return k >= 1 && k <= kindCount }

// Name returns the stable descriptive name of the construct, or "" if k is not
// a valid Kind.
func (k Kind) Name() string {
	if !k.Valid() {
		return ""
	}
	return descriptors[k].name
}

// Category returns the grammatical category of the construct, or
// CategoryInvalid if k is not a valid Kind.
func (k Kind) Category() Category {
	if !k.Valid() {
		return CategoryInvalid
	}
	return descriptors[k].category
}

// Disposition returns the support disposition of the construct, or
// DispositionInvalid if k is not a valid Kind.
func (k Kind) Disposition() Disposition {
	if !k.Valid() {
		return DispositionInvalid
	}
	return descriptors[k].disposition
}

// String renders k for diagnostics.
func (k Kind) String() string {
	if name := k.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.Kind(%d)", uint16(k))
}

// All returns every valid Kind in ascending identity order. Iterating All is
// the one way consumers walk the catalog; it never includes KindInvalid.
func All() []Kind {
	kinds := make([]Kind, 0, kindCount)
	for id := 1; id <= kindCount; id++ {
		kinds = append(kinds, Kind(id))
	}
	return kinds
}
