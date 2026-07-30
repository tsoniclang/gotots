package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Valid(
	context api.Context,
	source *ast.SelectorExpr,
	selected *types.Selection,
	kind types.SelectionKind,
) bool {
	if source == nil ||
		source.X == nil ||
		source.Sel == nil ||
		selected == nil ||
		selected.Kind() != kind ||
		context.TypesInfo().Selections[source] != selected ||
		context.TypesInfo().Uses[source.Sel] != selected.Obj() {
		return false
	}
	receiver := context.TypesInfo().TypeOf(source.X)
	result := context.TypesInfo().TypeOf(source)
	return receiver != nil &&
		result != nil &&
		types.Identical(receiver, selected.Recv()) &&
		types.Identical(result, selected.Type())
}

type path struct {
	root      types.Type
	fields    []*types.Var
	effective types.Type
}

func fieldPath(selected *types.Selection) (path, bool) {
	if selected == nil || selected.Kind() != types.FieldVal {
		return path{}, false
	}
	result, ok := resolveFields(selected.Recv(), selected.Index())
	if !ok || len(result.fields) == 0 ||
		result.fields[len(result.fields)-1] != selected.Obj() {
		return path{}, false
	}
	return result, true
}

func methodPath(selected *types.Selection) (path, *types.Func, bool) {
	if selected == nil ||
		(selected.Kind() != types.MethodVal &&
			selected.Kind() != types.MethodExpr) {
		return path{}, nil, false
	}
	indices := selected.Index()
	if len(indices) == 0 {
		return path{}, nil, false
	}
	result, ok := resolveFields(
		selected.Recv(),
		indices[:len(indices)-1],
	)
	if !ok {
		return path{}, nil, false
	}
	base := result.effective
	if pointer, ok := types.Unalias(base).(*types.Pointer); ok {
		base = pointer.Elem()
	}
	methodIndex := indices[len(indices)-1]
	if methodIndex < 0 {
		return path{}, nil, false
	}
	method, ok := selectedMethod(base, methodIndex)
	if !ok || method == nil || method != selected.Obj() {
		return path{}, nil, false
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return path{}, nil, false
	}
	return result, method, true
}

func selectedMethod(sourceType types.Type, index int) (*types.Func, bool) {
	base := types.Unalias(sourceType)
	named, namedType := base.(*types.Named)
	if namedType &&
		named.TypeParams().Len() != 0 &&
		named.TypeArgs().Len() != named.TypeParams().Len() {
		return nil, false
	}
	if interfaceType, ok := base.Underlying().(*types.Interface); ok {
		interfaceType = interfaceType.Complete()
		if index >= interfaceType.NumMethods() {
			return nil, false
		}
		return interfaceType.Method(index), true
	}
	if !namedType || index >= named.NumMethods() {
		return nil, false
	}
	return named.Method(index), true
}

func resolveFields(root types.Type, indices []int) (path, bool) {
	if root == nil {
		return path{}, false
	}
	current := root
	fields := make([]*types.Var, 0, len(indices))
	for offset, index := range indices {
		structType, ok := selectedStruct(current)
		if !ok || index < 0 || index >= structType.NumFields() {
			return path{}, false
		}
		field := structType.Field(index)
		if field == nil ||
			(offset != len(indices)-1 && !field.Embedded()) {
			return path{}, false
		}
		fields = append(fields, field)
		current = field.Type()
	}
	return path{
		root:      root,
		fields:    fields,
		effective: current,
	}, true
}

func selectedStruct(sourceType types.Type) (*types.Struct, bool) {
	base := types.Unalias(sourceType)
	if named, ok := base.(*types.Named); ok {
		base = types.Unalias(named.Underlying())
	}
	if pointer, ok := base.(*types.Pointer); ok {
		base = types.Unalias(pointer.Elem())
		if named, ok := base.(*types.Named); ok {
			base = types.Unalias(named.Underlying())
		}
	}
	result, ok := base.(*types.Struct)
	return result, ok
}
