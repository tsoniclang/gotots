package stagecheck

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (
	verifier *checkerSemanticVerifier,
) independentInferredArrayEllipsisType(
	occurrence structure.OccurrenceRef,
	node ast.Node,
) types.Type {
	if occurrence.Kind() != catalog.KindEllipsis ||
		occurrence.Role() != catalog.RoleArrayLength {
		return nil
	}
	ellipsis, ellipsisNode := node.(*ast.Ellipsis)
	parent, present := verifier.index.OccurrenceNode(
		occurrence.Parent(),
	)
	if !ellipsisNode || !present {
		return nil
	}
	array, arrayNode := parent.(*ast.ArrayType)
	if !arrayNode || array.Len != ellipsis {
		return nil
	}
	value, typed := verifier.view.TypeOf(array)
	if !typed || !value.IsType() {
		return nil
	}
	underlying, inferred := types.Unalias(
		value.Type,
	).Underlying().(*types.Array)
	if !inferred || underlying.Len() < 0 {
		return nil
	}
	return value.Type
}
