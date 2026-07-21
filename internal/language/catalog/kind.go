// Package catalog is the closed, target-independent catalog of Go language
// constructs for the selected Go toolchain. It names every grammatical form
// with a stable enum identity and a validated descriptor.
//
// The catalog is authoritative Go code, not parallel hand-maintained JSON. It
// imports no go/ast, go/types, or target package: the binding from concrete
// syntax to a catalog Kind lives in internal/language/analyze, the only other
// package permitted to touch the toolchain AST. This separation keeps the
// vocabulary independent of both the frontend and the target.
package catalog

import "fmt"

// Kind is the stable identity of one Go grammatical construct form. The zero
// value KindInvalid is never a valid construct; numKinds is the terminal
// sentinel that sizes the descriptor table so an added enumerator without a
// descriptor, or a stray value past the end, fails construction and tests.
type Kind uint16

const (
	KindInvalid Kind = iota

	// Expressions.
	KindBadExpr
	KindIdent
	KindEllipsis
	KindBasicLit
	KindFuncLit
	KindCompositeLit
	KindParenExpr
	KindSelectorExpr
	KindIndexExpr
	KindIndexListExpr
	KindSliceExpr
	KindTypeAssertExpr
	KindCallExpr
	KindStarExpr
	KindUnaryExpr
	KindBinaryExpr
	KindKeyValueExpr

	// Type expressions.
	KindArrayType
	KindStructType
	KindFuncType
	KindInterfaceType
	KindMapType
	KindChanType

	// Statements.
	KindBadStmt
	KindDeclStmt
	KindEmptyStmt
	KindLabeledStmt
	KindExprStmt
	KindSendStmt
	KindIncDecStmt
	KindAssignStmt
	KindGoStmt
	KindDeferStmt
	KindReturnStmt
	KindBranchStmt
	KindBlockStmt
	KindIfStmt
	KindCaseClause
	KindSwitchStmt
	KindTypeSwitchStmt
	KindCommClause
	KindSelectStmt
	KindForStmt
	KindRangeStmt

	// Declarations.
	KindBadDecl
	KindGenDecl
	KindFuncDecl

	// Specifications.
	KindImportSpec
	KindValueSpec
	KindTypeSpec

	// Structural nodes.
	KindFile
	KindComment
	KindCommentGroup
	KindField
	KindFieldList

	// numKinds is the terminal sentinel. It must remain last.
	numKinds
)

// descriptor is the growing per-kind record. This slice of the catalog carries
// the stable name and grammatical category; later phases extend it with
// applicable roles, required typed evidence, produced semantic operation, and
// allowed support dispositions, each behind its own closed enum.
type descriptor struct {
	name     string
	category Category
}

// descriptors is an exact-size table indexed by Kind. A fixed array length of
// numKinds means a Kind added past the current end fails to compile, and a
// Kind added without a descriptor here leaves a zero-value (empty-name) entry
// that TestKindTableIsTotal rejects.
var descriptors = [numKinds]descriptor{
	KindBadExpr:        {"BadExpr", CategoryExpression},
	KindIdent:          {"Ident", CategoryExpression},
	KindEllipsis:       {"Ellipsis", CategoryExpression},
	KindBasicLit:       {"BasicLit", CategoryExpression},
	KindFuncLit:        {"FuncLit", CategoryExpression},
	KindCompositeLit:   {"CompositeLit", CategoryExpression},
	KindParenExpr:      {"ParenExpr", CategoryExpression},
	KindSelectorExpr:   {"SelectorExpr", CategoryExpression},
	KindIndexExpr:      {"IndexExpr", CategoryExpression},
	KindIndexListExpr:  {"IndexListExpr", CategoryExpression},
	KindSliceExpr:      {"SliceExpr", CategoryExpression},
	KindTypeAssertExpr: {"TypeAssertExpr", CategoryExpression},
	KindCallExpr:       {"CallExpr", CategoryExpression},
	KindStarExpr:       {"StarExpr", CategoryExpression},
	KindUnaryExpr:      {"UnaryExpr", CategoryExpression},
	KindBinaryExpr:     {"BinaryExpr", CategoryExpression},
	KindKeyValueExpr:   {"KeyValueExpr", CategoryExpression},

	KindArrayType:     {"ArrayType", CategoryType},
	KindStructType:    {"StructType", CategoryType},
	KindFuncType:      {"FuncType", CategoryType},
	KindInterfaceType: {"InterfaceType", CategoryType},
	KindMapType:       {"MapType", CategoryType},
	KindChanType:      {"ChanType", CategoryType},

	KindBadStmt:        {"BadStmt", CategoryStatement},
	KindDeclStmt:       {"DeclStmt", CategoryStatement},
	KindEmptyStmt:      {"EmptyStmt", CategoryStatement},
	KindLabeledStmt:    {"LabeledStmt", CategoryStatement},
	KindExprStmt:       {"ExprStmt", CategoryStatement},
	KindSendStmt:       {"SendStmt", CategoryStatement},
	KindIncDecStmt:     {"IncDecStmt", CategoryStatement},
	KindAssignStmt:     {"AssignStmt", CategoryStatement},
	KindGoStmt:         {"GoStmt", CategoryStatement},
	KindDeferStmt:      {"DeferStmt", CategoryStatement},
	KindReturnStmt:     {"ReturnStmt", CategoryStatement},
	KindBranchStmt:     {"BranchStmt", CategoryStatement},
	KindBlockStmt:      {"BlockStmt", CategoryStatement},
	KindIfStmt:         {"IfStmt", CategoryStatement},
	KindCaseClause:     {"CaseClause", CategoryStatement},
	KindSwitchStmt:     {"SwitchStmt", CategoryStatement},
	KindTypeSwitchStmt: {"TypeSwitchStmt", CategoryStatement},
	KindCommClause:     {"CommClause", CategoryStatement},
	KindSelectStmt:     {"SelectStmt", CategoryStatement},
	KindForStmt:        {"ForStmt", CategoryStatement},
	KindRangeStmt:      {"RangeStmt", CategoryStatement},

	KindBadDecl:  {"BadDecl", CategoryDeclaration},
	KindGenDecl:  {"GenDecl", CategoryDeclaration},
	KindFuncDecl: {"FuncDecl", CategoryDeclaration},

	KindImportSpec: {"ImportSpec", CategorySpec},
	KindValueSpec:  {"ValueSpec", CategorySpec},
	KindTypeSpec:   {"TypeSpec", CategorySpec},

	KindFile:         {"File", CategoryStructural},
	KindComment:      {"Comment", CategoryStructural},
	KindCommentGroup: {"CommentGroup", CategoryStructural},
	KindField:        {"Field", CategoryStructural},
	KindFieldList:    {"FieldList", CategoryStructural},
}

// Valid reports whether k names a construct in the catalog. KindInvalid and
// the terminal sentinel are not valid.
func (k Kind) Valid() bool { return k > KindInvalid && k < numKinds }

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

// String renders k for diagnostics.
func (k Kind) String() string {
	if name := k.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.Kind(%d)", uint16(k))
}

// All returns every valid Kind in enumeration order. Iterating All is the one
// way consumers walk the catalog; it never includes KindInvalid or the
// terminal sentinel.
func All() []Kind {
	kinds := make([]Kind, 0, int(numKinds)-1)
	for k := KindInvalid + 1; k < numKinds; k++ {
		kinds = append(kinds, k)
	}
	return kinds
}
