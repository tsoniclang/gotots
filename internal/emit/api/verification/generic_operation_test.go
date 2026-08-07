package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"regexp"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGenericOperationIdentifiersAreTotalUniqueTargetIdentifiers(t *testing.T) {
	targetIdentifier := regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	seen := make(map[string]GenericOperation)
	for operation := GenericOperationZero; operation <= GenericOperationReflectionValue; operation++ {
		identifier := operation.Identifier()
		if !targetIdentifier.MatchString(identifier) {
			t.Fatalf(
				"generic operation %d identifier = %q",
				operation,
				identifier,
			)
		}
		if previous, duplicate := seen[identifier]; duplicate {
			t.Fatalf(
				"generic operations %d and %d share identifier %q",
				previous,
				operation,
				identifier,
			)
		}
		seen[identifier] = operation
	}
	if GenericOperationInvalid.Identifier() != "" ||
		GenericOperation(GenericOperationReflectionValue+1).Identifier() != "" {
		t.Fatal("invalid generic operation has a target identifier")
	}
	if GenericOperationToContainerStorage != GenericOperationFromStorage+1 ||
		GenericOperationFromContainerStorage !=
			GenericOperationToContainerStorage+1 ||
		GenericOperationIndexAddress !=
			GenericOperationFromContainerStorage+1 ||
		GenericOperationSlice != GenericOperationIndexAddress+1 ||
		GenericOperationSliceFull != GenericOperationSlice+1 ||
		GenericOperationDeferredCallableRegistry != GenericOperationSliceFull+1 ||
		GenericOperationAppendSpread != GenericOperationDeferredCallableRegistry+1 ||
		GenericOperationReflectionType != GenericOperationAppendSpread+1 ||
		GenericOperationReflectionValue != GenericOperationReflectionType+1 {
		t.Fatal("new generic operations were not appended canonically")
	}
}

func TestConstraintMethodSelectionPreservesExactMethodIdentity(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		),
		false,
	)
	selectionKey := func(method *types.Func) string {
		t.Helper()
		selection, err := SelectGenericConstraintMethod(method)
		if err != nil {
			t.Fatal(err)
		}
		key, err := selection.IdentityPrefix()
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	readFirst := types.NewFunc(
		token.NoPos,
		firstPackage,
		"Read",
		signature,
	)
	readSecond := types.NewFunc(
		token.NoPos,
		secondPackage,
		"Read",
		signature,
	)
	write := types.NewFunc(
		token.NoPos,
		firstPackage,
		"Write",
		signature,
	)
	privateFirst := types.NewFunc(
		token.NoPos,
		firstPackage,
		"read",
		signature,
	)
	privateSecond := types.NewFunc(
		token.NoPos,
		secondPackage,
		"read",
		signature,
	)
	if selectionKey(readFirst) != selectionKey(readSecond) {
		t.Fatal("equivalent exported constraint methods differ by package")
	}
	if selectionKey(readFirst) == selectionKey(write) {
		t.Fatal("different constraint methods with one signature share identity")
	}
	if selectionKey(privateFirst) == selectionKey(privateSecond) {
		t.Fatal("unexported constraint methods from different packages share identity")
	}
	if _, err := SelectGenericOperation(
		GenericOperationConstraintMethod,
	); err == nil {
		t.Fatal("constraint method accepted without exact method evidence")
	}
}

func TestGenericOperationContractCarriesCrossParameterSignature(t *testing.T) {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	left := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	right := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "U", nil),
		constraint,
	)
	operationSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", left),
			types.NewVar(token.NoPos, nil, "count", right),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", left)),
		false,
	)
	ownerSignature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{left, right},
		operationSignature.Params(),
		operationSignature.Results(),
		false,
	)
	owner := types.NewFunc(token.NoPos, nil, "Shift", ownerSignature)
	selection, err := SelectGenericOperation(
		GenericOperationBinaryShiftLeft,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := NewGenericOperationContract(
		owner,
		"binary_shift_left|(T,U)->T",
		"$go$binary_shift_left",
		GenericFunctionOperationConsumer(),
		selection,
		operationSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	callable, err := NewGenericOperationSet(
		owner,
		GenericFunctionOperationConsumer(),
		[]*GenericOperationContract{contract},
	)
	if err != nil {
		t.Fatal(err)
	}
	operations := callable.Operations()
	if len(operations) != 1 ||
		!types.Identical(operations[0].Signature(), operationSignature) ||
		operations[0].Operation() != GenericOperationBinaryShiftLeft {
		t.Fatalf("cross-parameter generic operation = %#v", operations)
	}
	foreignParameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "V", nil),
		constraint,
	)
	foreignOwner := types.NewFunc(
		token.NoPos,
		nil,
		"Foreign",
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{foreignParameter},
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "value", foreignParameter),
			),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", foreignParameter),
			),
			false,
		),
	)
	if _, err := NewGenericOperationRequirement(
		foreignOwner,
		contract,
	); err == nil {
		t.Fatal("foreign generic owner accepted another callable's operation")
	}
	foreignOperation := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", foreignParameter),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", foreignParameter),
		),
		false,
	)
	if _, err := NewGenericOperationContract(
		owner,
		"copy|(V)->V",
		"$go$copy",
		GenericFunctionOperationConsumer(),
		mustGenericOperation(t, GenericOperationCopy),
		foreignOperation,
	); err == nil {
		t.Fatal("generic operation accepted a foreign type parameter")
	}
}

func mustGenericOperation(
	t *testing.T,
	operation GenericOperation,
) GenericOperationSelection {
	t.Helper()
	selection, err := SelectGenericOperation(operation)
	if err != nil {
		t.Fatal(err)
	}
	return selection
}

func TestGenericParametersRejectForeignSameShapeArtifactOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/generic", "generic")
	for _, testCase := range []struct {
		name string
		pair func() (types.Object, *types.TypeParam, types.Object, *types.TypeParam)
	}{
		{name: "function", pair: func() (
			types.Object,
			*types.TypeParam,
			types.Object,
			*types.TypeParam,
		) {
			firstParameter := genericContextTypeParameter(sourcePackage, "T")
			secondParameter := genericContextTypeParameter(sourcePackage, "T")
			return genericContextFunction(
					sourcePackage,
					"Transform",
					firstParameter,
				),
				firstParameter,
				genericContextFunction(
					sourcePackage,
					"Transform",
					secondParameter,
				),
				secondParameter
		}},
		{name: "type", pair: func() (
			types.Object,
			*types.TypeParam,
			types.Object,
			*types.TypeParam,
		) {
			first, firstParameter := genericContextNamedType(
				sourcePackage,
				"Box",
			)
			second, secondParameter := genericContextNamedType(
				sourcePackage,
				"Box",
			)
			return first, firstParameter, second, secondParameter
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			owner, parameter, foreign, foreignParameter := testCase.pair()
			if owner == foreign ||
				owner.Name() != foreign.Name() ||
				!types.Identical(
					owner.Type().Underlying(),
					foreign.Type().Underlying(),
				) {
				t.Fatal("mutation control is not a foreign same-shape owner")
			}
			ownerOrigin := GenericDeclarationOrigin(owner)
			foreignOrigin := GenericDeclarationOrigin(foreign)
			if ownerOrigin != owner ||
				foreignOrigin != foreign ||
				ownerOrigin == foreignOrigin {
				t.Fatal("generic origin collapsed foreign same-shape owners")
			}
			ownerParameters := GenericDeclarationParameters(ownerOrigin)
			foreignParameters := GenericDeclarationParameters(foreignOrigin)
			if len(ownerParameters) != 1 ||
				ownerParameters[0] != parameter ||
				len(foreignParameters) != 1 ||
				foreignParameters[0] != foreignParameter {
				t.Fatal("generic parameter authority diverged from exact origins")
			}
			context, err := (Context{}).WithSourceArtifactOwner(
				MustSourceArtifactOwner(owner),
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := context.WithGenericParameters(
				owner,
				map[*types.TypeParam]string{parameter: "T"},
			); err != nil {
				t.Fatalf("source owner rejected: %v", err)
			}
			_, err = context.WithGenericParameters(
				foreign,
				map[*types.TypeParam]string{foreignParameter: "T"},
			)
			var contextError *ContextError
			if !errors.As(err, &contextError) {
				t.Fatalf("foreign owner error = %#v, want ContextError", err)
			}
		})
	}
}

func genericContextTypeParameter(
	sourcePackage *types.Package,
	name string,
) *types.TypeParam {
	constraint := types.NewInterfaceType(nil, nil)
	constraint.Complete()
	return types.NewTypeParam(
		types.NewTypeName(token.NoPos, sourcePackage, name, nil),
		constraint,
	)
}

func genericContextFunction(
	sourcePackage *types.Package,
	name string,
	parameter *types.TypeParam,
) *types.Func {
	return types.NewFunc(
		token.NoPos,
		sourcePackage,
		name,
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{parameter},
			types.NewTuple(types.NewVar(
				token.NoPos,
				sourcePackage,
				"value",
				parameter,
			)),
			types.NewTuple(types.NewVar(
				token.NoPos,
				sourcePackage,
				"",
				parameter,
			)),
			false,
		),
	)
}

func genericContextNamedType(
	sourcePackage *types.Package,
	name string,
) (*types.TypeName, *types.TypeParam) {
	parameter := genericContextTypeParameter(sourcePackage, "T")
	owner := types.NewTypeName(token.NoPos, sourcePackage, name, nil)
	named := types.NewNamed(owner, types.NewStruct(nil, nil), nil)
	named.SetTypeParams([]*types.TypeParam{parameter})
	return owner, parameter
}
