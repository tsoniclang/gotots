package rangestatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
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
	sourceType := context.TypesInfo().TypeOf(source.X)
	if sourceType == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryExpression, source.X)
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
