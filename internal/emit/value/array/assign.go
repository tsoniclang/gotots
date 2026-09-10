package array

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Assign(context api.Context, source ast.Node, target tsgo.Expression, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	targetName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	valueName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	destination := context.Factory().Identifier(targetName)
	incoming := context.Factory().Identifier(valueName)
	index := context.Factory().Identifier(indexName)
	storedTarget, err := a.storage(context, api.DirectExpression(destination))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedValue, err := a.storage(context, api.DirectExpression(incoming))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	entry := callMember(context, storedValue.Value(), arraymember.Get, index)
	var update api.ExpressionEmission
	_, generic := api.GenericTypeParameter(a.ElementType())
	_, nestedArray := a.ElementType().Underlying().(*types.Array)
	_, structure := a.ElementType().Underlying().(*types.Struct)
	if generic || nestedArray || structure {
		left, loadErr := a.loadElement(context, source, callMember(context, storedTarget.Value(), arraymember.Get, index))
		if loadErr != nil {
			return api.ExpressionEmission{}, loadErr
		}
		right, loadErr := a.loadElement(context, source, entry)
		if loadErr != nil {
			return api.ExpressionEmission{}, loadErr
		}
		update, err = context.StableAssignments().AssignStable(context, source, a.ElementType(), left.Value(), right)
		if err == nil {
			update, err = api.NewExpressionEmission(append(left.Before(), update.Before()...), update.Value(), api.CombineRequests(left.Requests(), update.Requests()))
		}
		if err == nil && generic {
			update, err = a.storeElement(context, source, update)
			if err == nil {
				update, err = api.NewExpressionEmission(update.Before(), callMember(context, storedTarget.Value(), arraymember.Set, index, update.Value()), update.Requests())
			}
		}
	} else {
		update = api.DirectExpression(callMember(context, storedTarget.Value(), arraymember.Set, index, entry))
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(value.Before(),
		arrayComparisonVariable(context, tsgo.NodeFlagsConst, targetName, target),
		arrayComparisonVariable(context, tsgo.NodeFlagsConst, valueName, value.Value()),
	)
	body := append(update.Before(), context.Factory().ExpressionStatement(update.Value()))
	before = append(before, arrayConstructionLoop(context, index, a.lengthLiteral(context), "0", body))
	return api.NewExpressionEmission(before, destination, api.CombineRequests(value.Requests(), storedTarget.Requests(), storedValue.Requests(), update.Requests()))
}
