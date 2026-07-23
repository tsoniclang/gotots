package catalog

import "fmt"

// Variant is the closed semantic variant of one construct occurrence, resolved
// from typed context (go/types evidence plus the parent edge). Values are
// explicit and permanent. VariantNone is the valid resolution for a kind with
// no variant axis; VariantInvalid is never valid.
type Variant uint16

// Explicit, permanent variant identities. Do not renumber; append only.
const (
	VariantInvalid Variant = 0
	VariantNone    Variant = 1

	// CallExpr.
	VariantCallFunction Variant = 2
	VariantCallMethod   Variant = 3
	VariantCallBuiltin  Variant = 4
	VariantConversion   Variant = 5

	// IndexExpr / IndexListExpr.
	VariantIndexElement         Variant = 6
	VariantMapLookupValue       Variant = 7
	VariantMapLookupCommaOk     Variant = 8
	VariantGenericInstantiation Variant = 9

	// TypeAssertExpr.
	VariantAssertValue     Variant = 10
	VariantAssertCommaOk   Variant = 11
	VariantTypeSwitchGuard Variant = 12

	// SelectorExpr.
	VariantSelectField            Variant = 13
	VariantSelectPromotedField    Variant = 14
	VariantSelectMethodValue      Variant = 15
	VariantSelectMethodExpression Variant = 16
	VariantSelectPackageMember    Variant = 17

	// AssignStmt.
	VariantAssignBalanced Variant = 18
	VariantAssignCommaOk  Variant = 19
	VariantAssignFromCall Variant = 20
	VariantDefineBalanced Variant = 21
	VariantDefineCommaOk  Variant = 22
	VariantDefineFromCall Variant = 23
	VariantAssignCompound Variant = 24

	// CompositeLit.
	VariantLitStruct Variant = 25
	VariantLitArray  Variant = 26
	VariantLitSlice  Variant = 27
	VariantLitMap    Variant = 28

	// KeyValueExpr.
	VariantKeyFieldName  Variant = 29
	VariantKeyMapKey     Variant = 30
	VariantKeyArrayIndex Variant = 31

	// UnaryExpr receive.
	VariantReceiveValue   Variant = 32
	VariantReceiveCommaOk Variant = 33

	// StarExpr.
	VariantStarPointerType Variant = 34
	VariantStarDereference Variant = 35

	// ReturnStmt.
	VariantReturnVoid   Variant = 36
	VariantReturnValues Variant = 37
	VariantReturnBare   Variant = 38

	// RangeStmt.
	VariantRangeArray          Variant = 39
	VariantRangePointerToArray Variant = 40
	VariantRangeSlice          Variant = 41
	VariantRangeString         Variant = 42
	VariantRangeMap            Variant = 43
	VariantRangeChannel        Variant = 44
	VariantRangeInteger        Variant = 45
	VariantRangeFunc           Variant = 46

	// SwitchStmt.
	VariantSwitchExpression Variant = 47
	VariantSwitchTrue       Variant = 48

	// TypeSpec.
	VariantTypeDefinition Variant = 49
	VariantTypeAlias      Variant = 50

	// CommClause.
	VariantCommSend    Variant = 51
	VariantCommReceive Variant = 52
	VariantCommDefault Variant = 53

	// variantCount is the highest assigned identity; append-only.
	variantCount = 53
)

var variantNames = [variantCount + 1]string{
	VariantNone:         "none",
	VariantCallFunction: "call-function", VariantCallMethod: "call-method",
	VariantCallBuiltin: "call-builtin", VariantConversion: "conversion",
	VariantIndexElement: "index-element", VariantMapLookupValue: "map-lookup-value",
	VariantMapLookupCommaOk: "map-lookup-comma-ok", VariantGenericInstantiation: "generic-instantiation",
	VariantAssertValue: "assert-value", VariantAssertCommaOk: "assert-comma-ok",
	VariantTypeSwitchGuard: "type-switch-guard",
	VariantSelectField:     "select-field", VariantSelectPromotedField: "select-promoted-field",
	VariantSelectMethodValue: "select-method-value", VariantSelectMethodExpression: "select-method-expression",
	VariantSelectPackageMember: "select-package-member",
	VariantAssignBalanced:      "assign-balanced", VariantAssignCommaOk: "assign-comma-ok",
	VariantAssignFromCall: "assign-from-call", VariantDefineBalanced: "define-balanced",
	VariantDefineCommaOk: "define-comma-ok", VariantDefineFromCall: "define-from-call",
	VariantAssignCompound: "assign-compound",
	VariantLitStruct:      "lit-struct", VariantLitArray: "lit-array",
	VariantLitSlice: "lit-slice", VariantLitMap: "lit-map",
	VariantKeyFieldName: "key-field-name", VariantKeyMapKey: "key-map-key",
	VariantKeyArrayIndex: "key-array-index",
	VariantReceiveValue:  "receive-value", VariantReceiveCommaOk: "receive-comma-ok",
	VariantStarPointerType: "star-pointer-type", VariantStarDereference: "star-dereference",
	VariantReturnVoid: "return-void", VariantReturnValues: "return-values",
	VariantReturnBare: "return-bare",
	VariantRangeArray: "range-array", VariantRangePointerToArray: "range-pointer-to-array",
	VariantRangeSlice: "range-slice", VariantRangeString: "range-string",
	VariantRangeMap: "range-map", VariantRangeChannel: "range-channel",
	VariantRangeInteger: "range-integer", VariantRangeFunc: "range-func",
	VariantSwitchExpression: "switch-expression", VariantSwitchTrue: "switch-true",
	VariantTypeDefinition: "type-definition", VariantTypeAlias: "type-alias",
	VariantCommSend: "comm-send", VariantCommReceive: "comm-receive",
	VariantCommDefault: "comm-default",
}

// Valid reports whether v names a variant in the catalog.
func (v Variant) Valid() bool { return v >= 1 && v <= variantCount }

// String renders v for diagnostics and reports.
func (v Variant) String() string {
	if v.Valid() && variantNames[v] != "" {
		return variantNames[v]
	}
	return fmt.Sprintf("catalog.Variant(%d)", uint16(v))
}

// AllVariants returns every valid Variant in ascending identity order.
func AllVariants() []Variant {
	out := make([]Variant, 0, variantCount)
	for id := 1; id <= variantCount; id++ {
		out = append(out, Variant(id))
	}
	return out
}

// variantBearing is the closed set of kinds that carry a variant axis; every
// occurrence of these kinds must resolve to a non-None variant (UnaryExpr
// resolves to None unless it is a receive).
var variantBearing = map[Kind]bool{
	KindCallExpr: true, KindIndexExpr: true, KindIndexListExpr: true,
	KindTypeAssertExpr: true, KindSelectorExpr: true, KindAssignStmt: true,
	KindCompositeLit: true, KindKeyValueExpr: true, KindUnaryExpr: true,
	KindStarExpr: true, KindReturnStmt: true, KindRangeStmt: true,
	KindSwitchStmt: true, KindTypeSpec: true, KindCommClause: true,
}

// VariantBearing reports whether kind carries a semantic variant axis.
func VariantBearing(kind Kind) bool { return variantBearing[kind] }
