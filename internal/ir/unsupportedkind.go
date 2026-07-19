package ir

import "fmt"

// Unsupported is the stable fail-closed diagnostic for a construct outside
// the reviewed subset.
type Unsupported struct {
	// Kind is the producer-owned closed classification of this exact
	// rejection site; every literal names it directly, so the inventory
	// classifies by Kind, never by string-prefix matching on Construct.
	Kind      UnsupportedKind
	Code      string // GOTOTS_UNSUPPORTED_{STATEMENT,EXPRESSION,TYPE,DECLARATION,OPERATION}
	Construct string
	Span      Span
}

func (u *Unsupported) Error() string {
	return fmt.Sprintf("%s:\n%s at %s:%d:%d", u.Code, u.Construct, u.Span.File, u.Span.Line, u.Span.Col)
}

// UnsupportedKind is the CLOSED, producer-owned classification of one
// rejection site. Each &Unsupported literal names its Kind directly, so
// classification is an exhaustive operation over this enum rather than
// ordered string-prefix matching — an ordered-prefix shadow (e.g.
// "type switch on ..." collapsing to the "type" family) is impossible.
type UnsupportedKind int

const (
	// KindUnsupportedInvalid is the zero value: a site that never set a
	// Kind. It has no disposition, so the inventory surfaces it as
	// unclassified rather than mislabeling it.
	KindUnsupportedInvalid UnsupportedKind = iota
	KindAddressOf
	KindAddressOfALoopClauseVariable
	KindAddressOfANamedResult
	KindAddressOfANonAddressableExpression
	KindAddressOfARangeVariable
	KindAddressOfATupleBoundVariable
	KindAddressOfATypeSwitchVariable
	KindAliasDeclaration
	KindAppendOfFixedArrayElements
	KindAppendTo
	KindAssignmentArityMismatch
	KindAssignmentThrough
	KindAssignmentTo
	KindAssignmentToNonFieldSelector
	KindAssignmentToNonVariable
	KindAssignmentToken
	KindBasicType
	KindBlankImportInitSideEffects
	KindBlankNamedResult
	KindBlankSlotArity
	KindBlankSlotInUnrecognizedTupleForm
	KindBlankStructField
	KindBlankVariableDeclaration
	KindBodylessFunction
	KindBranch
	KindBreakInsideATypeSwitchClause
	KindBuiltin
	KindBuiltinStatement
	KindCallOf
	KindCallOutsideTheTranslatedUnit
	KindCallOutsideTheTranslatedUnitUnqualified
	KindCallToAGenericExternalMethod
	KindCallWithoutSignatureEvidence
	KindCapOf
	KindChannelSendStatement
	KindChannelType
	KindClearOf
	KindCompositeLiteralOf
	KindCompoundAssignmentArity
	KindCompoundAssignmentOn
	KindCompoundAssignmentToTheBlankIdentifier
	KindConstantOfType
	KindConversionAsStatement
	KindConversionFrom
	KindConversionFromUntypedNilTo
	KindCopyBetween
	KindCopyOfFixedArrayElements
	KindDeferBelowTheFunctionSTopLevelBlockRunsAtFunctionExitNeedsTheDeferStackLowering
	KindDeferInAFunctionWithNamedResultsDeferredResultMutation
	KindDeferredNonCallExpression
	KindDereferenceOf
	KindEqualityBetweenAnInterfaceAnd
	KindEqualityOn
	KindEqualityOnArrayOf
	KindEqualityPlanFor
	KindEqualityPlanForExternal
	KindExpressionStatement
	KindExpressionWithoutTypeEvidence
	KindFieldAccessOn
	KindFieldAssignmentOn
	KindFloat32Arithmetic
	KindFullSliceExpressionOn
	KindFunctionWithoutTypedDefinition
	KindGenericCall
	KindGenericCallInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation
	KindGenericCallWithoutInstantiationEvidence
	KindGenericExternalMethodCall
	KindGenericFunctionInstantiatedWithAStructValueCopySemanticsVaryPerInstantiation
	KindGenericFunctionInstantiatedWithAnUnreviewedTypeArgument
	KindGenericFunctionType
	KindGenericMethodCall
	KindGenericMethodExpression
	KindGenericMethodValue
	KindGenericTypeInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation
	KindGenericTypeInstantiatedWithAnUnreviewedTypeArgument
	KindGoroutineStatement
	KindIdentifier
	KindIncDecOf
	KindIndexOn
	KindIndexedAssignmentOn
	KindInterfaceMethodExpression
	KindInterfaceValueOfAnInstantiatedGenericType
	KindInterfaceValueOfType
	KindKeyedArrayLiteral
	KindKeyedSliceLiteral
	KindLabelOnANonLoopStatement
	KindLabelOnARangeOverFuncLoop
	KindLabeledBranchInsideARangeOverFuncBody
	KindLenOf
	KindMakeOf
	KindMapKeyType
	KindMapLiteralWithoutKeys
	KindMethodCallOn
	KindMethodCallOutsideTheTranslatedUnitUnqualified
	KindMethodCallWithoutSignatureEvidence
	KindMethodExpressionOnAnUnnamedReceiverType
	KindMethodExpressionOutsideTheTranslatedUnit
	KindMethodOnUnnamedReceiverType
	KindMethodValueBindTimeReceiverCapture
	KindMethodValueOn
	KindMethodValueOnAnUnnamedReceiverType
	KindMethodValueOutsideTheTranslatedUnit
	KindMethodWithoutCanonicalIdentity
	KindMethodWithoutCanonicalSlot
	KindMixedKeyedAndPositionalLiteral
	KindMultiResultCallInExpressionPosition
	KindMultiResultForwardingIntoAVariadicCall
	KindMultiValueVarInitializer
	KindNestedError
	KindNewOf
	KindNilComparisonOn
	KindNilOfType
	KindNonFieldSelector
	KindNonIntegralIntegerConstant
	KindNonStructNamedType
	KindNonValueVarSpec
	KindNonVarDeclarationStatement
	KindOperator
	KindOrderingOn
	KindPackageLevel
	KindPackageLevelMultiValueVarInitializer
	KindPackageLevelMultiVariableInitializer
	KindPanicWith
	KindPointerReceiverMethodCallOn
	KindPointerReceiverMethodValueOn
	KindPointerToNonNamedType
	KindPointerToNonStructType
	KindPointerToTypeOutsideTheTranslatedUnit
	KindPromotedGenericMethod
	KindPromotedMethodFromATypeOutsideTheTranslatedUnit
	KindPromotedMethodWithoutCanonicalIdentity
	KindPromotedSelectionThrough
	KindPromotionThroughANonStructEmbedding
	KindPromotionThroughAnEmbeddedPointer
	KindPromotionThroughAnUnnamedEmbedding
	KindRangeOver
	KindRangeOverAnIntegerWithASecondVariable
	KindRangeVariableIsNotAnIdentifier
	KindRangeWithAssignmentForm
	KindReferenceToAFunctionOutsideTheTranslatedUnit
	KindResliceOf
	KindReturnArityMismatch
	KindRuntimeTypeIdentityOf
	KindSelectStatement
	KindShortDeclarationArityMismatch
	KindShortDeclarationOfNonIdentifier
	KindShortDeclarationReusingANonVariable
	KindShortDeclarationReusingAnExistingVariableWithoutATupleSource
	KindStoreIntoASliceOfExternalValues
	KindStoreIntoAnArrayOfExternalValues
	KindStructType
	KindSwitchCaseOf
	KindSwitchTagOf
	KindTwoRangeVariablesOverAOneValueSequence
	KindType
	KindTypeAssertionOn
	KindTypeInCallPosition
	KindTypeSwitchClauseWithAnInterfaceTypeMethodSetTest
	KindTypeSwitchGuardForm
	KindTypeSwitchOn
	KindTypeWithoutTypedDefinition
	KindUnaryOperator
	KindGenericInstantiationOutsideAdmittedKeyFamily
	KindUnrecognizedExpression
	KindUnrecognizedStatement
	KindUntypedNilOutsideATypedContext
	KindVarWithoutTypedDefinition
	KindVariadicParameterIsNotASlice
	KindZeroValueOf

	// kindEnd is the exclusive upper bound of the enum — one past the last
	// real kind. It is the SINGLE authoritative count: AllUnsupportedKinds
	// ranges [KindUnsupportedInvalid+1, kindEnd), so a newly declared kind
	// is covered automatically and cannot be omitted from validation. Keep
	// it last.
	kindEnd
)

// kindName is each kind's stable class key (the inventory family name).
var kindName = map[UnsupportedKind]string{
	KindAddressOf:                               "address of",
	KindAddressOfALoopClauseVariable:            "address of a loop-clause variable",
	KindAddressOfANamedResult:                   "address of a named result",
	KindAddressOfANonAddressableExpression:      "address of a non-addressable expression",
	KindAddressOfARangeVariable:                 "address of a range variable",
	KindAddressOfATupleBoundVariable:            "address of a tuple-bound variable",
	KindAddressOfATypeSwitchVariable:            "address of a type-switch variable",
	KindAliasDeclaration:                        "alias declaration",
	KindAppendOfFixedArrayElements:              "append of fixed-array elements",
	KindAppendTo:                                "append to",
	KindAssignmentArityMismatch:                 "assignment arity mismatch",
	KindAssignmentThrough:                       "assignment through",
	KindAssignmentTo:                            "assignment to",
	KindAssignmentToNonFieldSelector:            "assignment to non-field selector",
	KindAssignmentToNonVariable:                 "assignment to non-variable",
	KindAssignmentToken:                         "assignment token",
	KindBasicType:                               "basic type",
	KindBlankImportInitSideEffects:              "blank import (init side effects)",
	KindBlankNamedResult:                        "blank named result",
	KindBlankSlotArity:                          "blank slot arity",
	KindBlankSlotInUnrecognizedTupleForm:        "blank slot in unrecognized tuple form",
	KindBlankStructField:                        "blank struct field",
	KindBlankVariableDeclaration:                "blank variable declaration",
	KindBodylessFunction:                        "bodyless function",
	KindBranch:                                  "branch",
	KindBreakInsideATypeSwitchClause:            "break inside a type switch clause",
	KindBuiltin:                                 "builtin",
	KindBuiltinStatement:                        "builtin statement",
	KindCallOf:                                  "call of",
	KindCallOutsideTheTranslatedUnit:            "call outside the translated unit (",
	KindCallOutsideTheTranslatedUnitUnqualified: "call outside the translated unit (unqualified)",
	KindCallToAGenericExternalMethod:            "call to a generic external method (",
	KindCallWithoutSignatureEvidence:            "call without signature evidence",
	KindCapOf:                                   "cap of",
	KindChannelSendStatement:                    "channel send statement",
	KindChannelType:                             "channel type",
	KindClearOf:                                 "clear of",
	KindCompositeLiteralOf:                      "composite literal of",
	KindCompoundAssignmentArity:                 "compound assignment arity",
	KindCompoundAssignmentOn:                    "compound assignment on",
	KindCompoundAssignmentToTheBlankIdentifier:  "compound assignment to the blank identifier",
	KindConstantOfType:                          "constant of type",
	KindConversionAsStatement:                   "conversion as statement",
	KindConversionFrom:                          "conversion from",
	KindConversionFromUntypedNilTo:              "conversion from untyped nil to",
	KindCopyBetween:                             "copy between",
	KindCopyOfFixedArrayElements:                "copy of fixed-array elements",
	KindDeferBelowTheFunctionSTopLevelBlockRunsAtFunctionExitNeedsTheDeferStackLowering: "defer below the function's top-level block (runs at function exit; needs the defer-stack lowering)",
	KindDeferInAFunctionWithNamedResultsDeferredResultMutation:                          "defer in a function with named results (deferred result mutation)",
	KindDeferredNonCallExpression:                                                       "deferred non-call expression",
	KindDereferenceOf:                                                                   "dereference of",
	KindEqualityBetweenAnInterfaceAnd:                                                   "equality between an interface and",
	KindEqualityOn:                                                                      "equality on",
	KindEqualityOnArrayOf:                                                               "equality on array of",
	KindEqualityPlanFor:                                                                 "equality plan for",
	KindEqualityPlanForExternal:                                                         "equality plan for external",
	KindExpressionStatement:                                                             "expression statement",
	KindExpressionWithoutTypeEvidence:                                                   "expression without type evidence",
	KindFieldAccessOn:                                                                   "field access on",
	KindFieldAssignmentOn:                                                               "field assignment on",
	KindFloat32Arithmetic:                                                               "float32 arithmetic",
	KindFullSliceExpressionOn:                                                           "full slice expression on",
	KindFunctionWithoutTypedDefinition:                                                  "function without typed definition",
	KindGenericCall:                                                                     "generic call",
	KindGenericCallInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation: "generic call instantiated with a value-copy carrier (copy semantics vary per instantiation)",
	KindGenericCallWithoutInstantiationEvidence:                                       "generic call without instantiation evidence",
	KindGenericExternalMethodCall:                                                     "generic external method call",
	KindGenericFunctionInstantiatedWithAStructValueCopySemanticsVaryPerInstantiation:  "generic function instantiated with a struct value (copy semantics vary per instantiation)",
	KindGenericFunctionInstantiatedWithAnUnreviewedTypeArgument:                       "generic function instantiated with an unreviewed type argument (",
	KindGenericFunctionType:                                                           "generic function type",
	KindGenericMethodCall:                                                             "generic method call",
	KindGenericMethodExpression:                                                       "generic method expression",
	KindGenericMethodValue:                                                            "generic method value",
	KindGenericTypeInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation: "generic type instantiated with a value-copy carrier (copy semantics vary per instantiation)",
	KindGenericTypeInstantiatedWithAnUnreviewedTypeArgument:                           "generic type instantiated with an unreviewed type argument (",
	KindGoroutineStatement:                                                            "goroutine statement",
	KindIdentifier:                                                                    "identifier",
	KindIncDecOf:                                                                      "inc/dec of",
	KindIndexOn:                                                                       "index on",
	KindIndexedAssignmentOn:                                                           "indexed assignment on",
	KindInterfaceMethodExpression:                                                     "interface method expression",
	KindInterfaceValueOfAnInstantiatedGenericType:                                     "interface value of an instantiated generic type",
	KindInterfaceValueOfType:                                                          "interface value of type",
	KindKeyedArrayLiteral:                                                             "keyed array literal",
	KindKeyedSliceLiteral:                                                             "keyed slice literal",
	KindLabelOnANonLoopStatement:                                                      "label on a non-loop statement",
	KindLabelOnARangeOverFuncLoop:                                                     "label on a range-over-func loop",
	KindLabeledBranchInsideARangeOverFuncBody:                                         "labeled branch inside a range-over-func body",
	KindLenOf:  "len of",
	KindMakeOf: "make of",
	KindGenericInstantiationOutsideAdmittedKeyFamily: "generic instantiation outside the admitted key family",
	KindMapKeyType:            "map key type",
	KindMapLiteralWithoutKeys: "map literal without keys",
	KindMethodCallOn:          "method call on",
	KindMethodCallOutsideTheTranslatedUnitUnqualified:                "method call outside the translated unit (unqualified)",
	KindMethodCallWithoutSignatureEvidence:                           "method call without signature evidence",
	KindMethodExpressionOnAnUnnamedReceiverType:                      "method expression on an unnamed receiver type",
	KindMethodExpressionOutsideTheTranslatedUnit:                     "method expression outside the translated unit (",
	KindMethodOnUnnamedReceiverType:                                  "method on unnamed receiver type",
	KindMethodValueBindTimeReceiverCapture:                           "method value (bind-time receiver capture)",
	KindMethodValueOn:                                                "method value on",
	KindMethodValueOnAnUnnamedReceiverType:                           "method value on an unnamed receiver type",
	KindMethodValueOutsideTheTranslatedUnit:                          "method value outside the translated unit (",
	KindMethodWithoutCanonicalIdentity:                               "method without canonical identity (",
	KindMethodWithoutCanonicalSlot:                                   "method without canonical slot (",
	KindMixedKeyedAndPositionalLiteral:                               "mixed keyed and positional literal",
	KindMultiResultCallInExpressionPosition:                          "multi-result call in expression position",
	KindMultiResultForwardingIntoAVariadicCall:                       "multi-result forwarding into a variadic call",
	KindMultiValueVarInitializer:                                     "multi-value var initializer",
	KindNestedError:                                                  "nested error",
	KindNewOf:                                                        "new of",
	KindNilComparisonOn:                                              "nil comparison on",
	KindNilOfType:                                                    "nil of type",
	KindNonFieldSelector:                                             "non-field selector",
	KindNonIntegralIntegerConstant:                                   "non-integral integer constant",
	KindNonStructNamedType:                                           "non-struct named type",
	KindNonValueVarSpec:                                              "non-value var spec",
	KindNonVarDeclarationStatement:                                   "non-var declaration statement",
	KindOperator:                                                     "operator",
	KindOrderingOn:                                                   "ordering on",
	KindPackageLevel:                                                 "package-level",
	KindPackageLevelMultiValueVarInitializer:                         "package-level multi-value var initializer",
	KindPackageLevelMultiVariableInitializer:                         "package-level multi-variable initializer",
	KindPanicWith:                                                    "panic with",
	KindPointerReceiverMethodCallOn:                                  "pointer-receiver method call on",
	KindPointerReceiverMethodValueOn:                                 "pointer-receiver method value on",
	KindPointerToNonNamedType:                                        "pointer to non-named type",
	KindPointerToNonStructType:                                       "pointer to non-struct type",
	KindPointerToTypeOutsideTheTranslatedUnit:                        "pointer to type outside the translated unit:",
	KindPromotedGenericMethod:                                        "promoted generic method (",
	KindPromotedMethodFromATypeOutsideTheTranslatedUnit:              "promoted method from a type outside the translated unit (",
	KindPromotedMethodWithoutCanonicalIdentity:                       "promoted method without canonical identity (",
	KindPromotedSelectionThrough:                                     "promoted selection through",
	KindPromotionThroughANonStructEmbedding:                          "promotion through a non-struct embedding (",
	KindPromotionThroughAnEmbeddedPointer:                            "promotion through an embedded pointer (",
	KindPromotionThroughAnUnnamedEmbedding:                           "promotion through an unnamed embedding (",
	KindRangeOver:                                                    "range over",
	KindRangeOverAnIntegerWithASecondVariable:                        "range over an integer with a second variable",
	KindRangeVariableIsNotAnIdentifier:                               "range variable is not an identifier",
	KindRangeWithAssignmentForm:                                      "range with assignment form",
	KindReferenceToAFunctionOutsideTheTranslatedUnit:                 "reference to a function outside the translated unit",
	KindResliceOf:                                                    "reslice of",
	KindReturnArityMismatch:                                          "return arity mismatch",
	KindRuntimeTypeIdentityOf:                                        "runtime type identity of",
	KindSelectStatement:                                              "select statement",
	KindShortDeclarationArityMismatch:                                "short declaration arity mismatch",
	KindShortDeclarationOfNonIdentifier:                              "short declaration of non-identifier",
	KindShortDeclarationReusingANonVariable:                          "short declaration reusing a non-variable",
	KindShortDeclarationReusingAnExistingVariableWithoutATupleSource: "short declaration reusing an existing variable without a tuple source",
	KindStoreIntoASliceOfExternalValues:                              "store into a slice of external values",
	KindStoreIntoAnArrayOfExternalValues:                             "store into an array of external values",
	KindStructType:                                                   "struct type",
	KindSwitchCaseOf:                                                 "switch case of",
	KindSwitchTagOf:                                                  "switch tag of",
	KindTwoRangeVariablesOverAOneValueSequence:                       "two range variables over a one-value sequence",
	KindType:               "type",
	KindTypeAssertionOn:    "type assertion on",
	KindTypeInCallPosition: "type in call position",
	KindTypeSwitchClauseWithAnInterfaceTypeMethodSetTest: "type switch clause with an interface type (method-set test)",
	KindTypeSwitchGuardForm:                              "type switch guard form",
	KindTypeSwitchOn:                                     "type switch on",
	KindTypeWithoutTypedDefinition:                       "type without typed definition",
	KindUnaryOperator:                                    "unary operator",
	KindUnrecognizedExpression:                           "unrecognized expression",
	KindUnrecognizedStatement:                            "unrecognized statement",
	KindUntypedNilOutsideATypedContext:                   "untyped nil outside a typed context",
	KindVarWithoutTypedDefinition:                        "var without typed definition",
	KindVariadicParameterIsNotASlice:                     "variadic parameter is not a slice",
	KindZeroValueOf:                                      "zero value of",
}

// String is the kind's stable class key, used as the site Class.
func (k UnsupportedKind) String() string {
	if s, ok := kindName[k]; ok {
		return s
	}
	return "invalid-unsupported-kind"
}

// AllUnsupportedKinds returns every declared kind, derived by range over
// the enum's own bounds [KindUnsupportedInvalid+1, kindEnd). The enum is
// the single authoritative definition — there is no hand-maintained
// parallel list, so a newly declared kind is included automatically and
// cannot silently evade the disposition-totality validation.
func AllUnsupportedKinds() []UnsupportedKind {
	out := make([]UnsupportedKind, 0, kindEnd-1)
	for k := KindUnsupportedInvalid + 1; k < kindEnd; k++ {
		out = append(out, k)
	}
	return out
}
