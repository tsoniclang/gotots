package rangestatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
) (api.StatementEmission, error) {
	context, targetLabel := context.TakeStatementLabel()
	if source == nil || source.X == nil || source.Body == nil ||
		!validClause(source) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	targetLabel, err := context.SelectControlTarget(targetLabel)
	if err != nil {
		return api.StatementEmission{}, err
	}
	sourceType := context.TypesInfo().TypeOf(source.X)
	if sourceType == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryExpression, source.X)
	}
	if _, generic := api.GenericTypeParameter(sourceType); generic {
		if target, handled, err := emitGenericMap(
			context,
			children,
			source,
			sourceType,
			targetLabel,
		); handled || err != nil {
			return target, err
		}
	}
	if signature, ok := callable.Signature(sourceType); ok {
		return emitIterator(
			context,
			children,
			source,
			sourceType,
			signature,
			targetLabel,
		)
	}
	if channel, ok := channelmodel.Resolve(sourceType); ok {
		return emitChannel(
			context,
			children,
			source,
			channel,
			targetLabel,
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return emitArray(context, children, source, array, targetLabel)
	}
	if _, element, ok := pointertype.Resolve(sourceType); ok {
		if array, arrayOK := arrayvalue.Resolve(context, element); arrayOK {
			return emitPointerArray(
				context,
				children,
				source,
				array,
				false,
				targetLabel,
			)
		}
	}
	if defined, ok := definedtype.ResolvePointer(sourceType); ok {
		pointer, _ := defined.Pointer()
		if array, arrayOK := arrayvalue.Resolve(
			context,
			pointer.Elem(),
		); arrayOK {
			return emitPointerArray(
				context,
				children,
				source,
				array,
				true,
				targetLabel,
			)
		}
	}
	if _, _, ok := slicevalue.Source(sourceType); ok {
		return emitSlice(context, children, source, targetLabel)
	}
	if supportsString(sourceType) {
		return emitString(context, children, source, targetLabel)
	}
	if _, ok := maprepresentation.Source(context, sourceType); ok {
		return emitMap(context, children, source, targetLabel)
	}
	if supportsInteger(context, sourceType) {
		return emitInteger(context, children, source, targetLabel)
	}
	return api.StatementEmission{},
		api.Unsupported(context, api.CategoryExpression, source.X)
}

func emitGenericMap(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	sourceType types.Type,
	targetLabel string,
) (api.StatementEmission, bool, error) {
	if source.Key == nil || source.Value == nil {
		return api.StatementEmission{}, false, nil
	}
	keyType := context.TypesInfo().TypeOf(source.Key)
	valueType := context.TypesInfo().TypeOf(source.Value)
	if keyType == nil || valueType == nil {
		return api.StatementEmission{}, false, nil
	}
	mapType := types.NewMap(keyType, valueType)
	if !types.AssignableTo(sourceType, mapType) ||
		!types.ConvertibleTo(sourceType, mapType) {
		return api.StatementEmission{}, false, nil
	}
	model, ok := maprepresentation.Source(context, mapType)
	if !ok {
		return api.StatementEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	projected, err := context.Values().Transfer(
		context.WithRole(api.RoleRangeExpression),
		source.X,
		sourceType,
		mapType,
		api.ValueTransferRepresentation,
		operand,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	target, err := emitMapValue(
		context,
		children,
		source,
		model,
		projected,
		targetLabel,
	)
	return target, true, err
}

func validClause(source *ast.RangeStmt) bool {
	switch source.Tok {
	case token.ILLEGAL:
		return source.Key == nil && source.Value == nil
	case token.DEFINE, token.ASSIGN:
		return source.Key != nil
	default:
		return false
	}
}

func supportsString(sourceType types.Type) bool {
	if basictype.SupportsString(sourceType) {
		return true
	}
	defined, ok := definedtype.ResolveBasic(sourceType)
	return ok && basictype.SupportsString(defined.Underlying())
}

func supportsInteger(context api.Context, sourceType types.Type) bool {
	if basictype.SupportsInteger(context.TypesSizes(), sourceType) {
		return true
	}
	defined, ok := definedtype.ResolveBasic(sourceType)
	return ok &&
		basictype.SupportsInteger(context.TypesSizes(), defined.Underlying())
}

func iteration(
	sourceType types.Type,
	value api.ExpressionEmission,
) (assignment.RangeIterationValue, error) {
	return assignment.NewRangeIterationValue(sourceType, value)
}
