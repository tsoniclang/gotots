package deferredregistry

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Reference(
	context api.Context,
	source ast.Node,
	signature *types.Signature,
) (api.NameReference, error) {
	if !api.ContainsGenericTypeParameter(signature) {
		return context.Names().DeferredCallableRegistry(signature)
	}
	operation, err := context.GenericOperation(
		source,
		api.GenericOperationDeferredCallableRegistry,
		signature,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(
		operation.Name(),
		operation.Requests()...,
	)
}
