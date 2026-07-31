package environment

import (
	"fmt"
	"go/token"
	"go/types"
)

func MethodSignature(method *types.Func) (*types.Signature, bool) {
	if method == nil {
		return nil, false
	}
	source, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, false
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		source.Params(),
		source.Results(),
		source.Variadic(),
	), true
}

func EquivalentMethods(left *types.Func, right *types.Func) bool {
	if left == nil || right == nil {
		return false
	}
	leftOrigin := left.Origin()
	rightOrigin := right.Origin()
	if leftOrigin.Name() != rightOrigin.Name() ||
		leftOrigin.Exported() != rightOrigin.Exported() {
		return false
	}
	if !leftOrigin.Exported() &&
		(leftOrigin.Pkg() == nil ||
			rightOrigin.Pkg() == nil ||
			leftOrigin.Pkg().Path() != rightOrigin.Pkg().Path()) {
		return false
	}
	parameterCount := max(
		methodReceiverParameterCount(leftOrigin),
		methodReceiverParameterCount(rightOrigin),
	)
	canonical := canonicalMethodArguments(parameterCount)
	leftSignature, leftOK := normalizedMethodSignature(left, canonical)
	rightSignature, rightOK := normalizedMethodSignature(right, canonical)
	if !leftOK || !rightOK ||
		!types.Identical(leftSignature, rightSignature) {
		return false
	}
	if leftOrigin.Exported() {
		return true
	}
	return true
}

func normalizedMethodSignature(
	method *types.Func,
	canonical []types.Type,
) (*types.Signature, bool) {
	origin := method.Origin()
	if method != origin {
		return MethodSignature(method)
	}
	receiver, count := methodReceiverOrigin(origin)
	if count == 0 {
		return MethodSignature(origin)
	}
	instantiated, err := types.Instantiate(
		types.NewContext(),
		receiver,
		canonical[:count],
		false,
	)
	if err != nil {
		return nil, false
	}
	named, ok := types.Unalias(instantiated).(*types.Named)
	if !ok {
		return nil, false
	}
	for index := range named.NumMethods() {
		candidate := named.Method(index)
		if candidate.Origin() == origin {
			return MethodSignature(candidate)
		}
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	contract = contract.Complete()
	for index := range contract.NumMethods() {
		candidate := contract.Method(index)
		if candidate.Origin() == origin {
			return MethodSignature(candidate)
		}
	}
	return nil, false
}

func methodReceiverParameterCount(method *types.Func) int {
	_, count := methodReceiverOrigin(method)
	return count
}

func methodReceiverOrigin(method *types.Func) (*types.Named, int) {
	if method == nil {
		return nil, 0
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, 0
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = types.Unalias(pointer.Elem())
	}
	named, ok := receiver.(*types.Named)
	if !ok || named.Origin() == nil {
		return nil, 0
	}
	origin := named.Origin()
	return origin, origin.TypeParams().Len()
}

func canonicalMethodArguments(count int) []types.Type {
	if count == 0 {
		return nil
	}
	owner := types.NewPackage(
		"github.com/tsoniclang/gotots/internal/contracts/environment/method",
		"methodcontract",
	)
	arguments := make([]types.Type, 0, count)
	for index := range count {
		name := types.NewTypeName(
			token.NoPos,
			owner,
			fmt.Sprintf("Parameter%d", index),
			nil,
		)
		arguments = append(
			arguments,
			types.NewNamed(name, types.Typ[types.Int], nil),
		)
	}
	return arguments
}
