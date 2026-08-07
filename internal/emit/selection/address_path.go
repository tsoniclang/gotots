package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func expandAddressPath(
	context api.Context,
	root ast.Expr,
	tail path,
) (ast.Expr, path, bool) {
	if root == nil || tail.root == nil || tail.effective == nil {
		return nil, path{}, false
	}
	for {
		selector, ok := root.(*ast.SelectorExpr)
		if !ok {
			break
		}
		selected := context.TypesInfo().SelectionOf(selector)
		prefix, ok := fieldPath(context, selected)
		if !ok ||
			!Valid(context, selector, selected, types.FieldVal) ||
			!types.Identical(prefix.effective, tail.root) {
			break
		}
		fields := make([]*types.Var, 0, len(prefix.fields)+len(tail.fields))
		fields = append(fields, prefix.fields...)
		fields = append(fields, tail.fields...)
		tail.root = prefix.root
		tail.fields = fields
		root = selector.X
	}
	if !types.Identical(context.TypesInfo().TypeOf(root), tail.root) {
		return nil, path{}, false
	}
	return root, tail, true
}
