package emit

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	functiondeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/function"
	namedstructdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	packageconstant "github.com/tsoniclang/gotots/internal/emit/declaration/packageconstant"
	binaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/binary"
	callexpression "github.com/tsoniclang/gotots/internal/emit/expression/call"
	compositeliteral "github.com/tsoniclang/gotots/internal/emit/expression/compositeliteral"
	dereferenceexpression "github.com/tsoniclang/gotots/internal/emit/expression/dereference"
	functionliteral "github.com/tsoniclang/gotots/internal/emit/expression/functionliteral"
	identifierexpression "github.com/tsoniclang/gotots/internal/emit/expression/identifier"
	integerliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/integer"
	parenthesizedexpression "github.com/tsoniclang/gotots/internal/emit/expression/parenthesized"
	selectorexpression "github.com/tsoniclang/gotots/internal/emit/expression/selector"
	unaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/unary"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	blockstatement "github.com/tsoniclang/gotots/internal/emit/statement/block"
	branchstatement "github.com/tsoniclang/gotots/internal/emit/statement/branch"
	expressionstatement "github.com/tsoniclang/gotots/internal/emit/statement/expressionstatement"
	forstatement "github.com/tsoniclang/gotots/internal/emit/statement/forstatement"
	ifstatement "github.com/tsoniclang/gotots/internal/emit/statement/ifstatement"
	incdecstatement "github.com/tsoniclang/gotots/internal/emit/statement/incdec"
	localconstant "github.com/tsoniclang/gotots/internal/emit/statement/localconstant"
	localdeclaration "github.com/tsoniclang/gotots/internal/emit/statement/localdeclaration"
	returnstatement "github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
	switchstatement "github.com/tsoniclang/gotots/internal/emit/statement/switchstatement"
	storetarget "github.com/tsoniclang/gotots/internal/emit/store"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	namedstructtype "github.com/tsoniclang/gotots/internal/emit/type/namedstruct"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	tupletype "github.com/tsoniclang/gotots/internal/emit/type/tuple"
	"github.com/tsoniclang/gotots/internal/emit/value/representation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type emitter struct {
	source  *load.Package
	factory tsgo.Factory
	names   *nameOwner
	values  api.Values
	integer api.IntegerRepresentation
	order   api.EvaluationOrder
	require func(types.Object) error
}

func newEmitter(
	source *load.Package,
	factory tsgo.Factory,
	registry *declarationRegistry,
	integer api.IntegerRepresentation,
	order api.EvaluationOrder,
	require func(types.Object) error,
) *emitter {
	var typesInfo *types.Info
	var packageScope *types.Scope
	if source != nil {
		typesInfo = source.TypesInfo()
		packageScope = source.Types().Scope()
	}
	return &emitter{
		source:  source,
		factory: factory,
		names:   newNameOwnerWithRegistry(packageScope, typesInfo, registry),
		values:  representation.Owner{},
		integer: integer,
		order:   order,
		require: require,
	}
}

func (e *emitter) fileContext(
	sourceFile *ast.File,
	targetPath string,
) (api.Context, error) {
	return e.targetContext(sourceFile, targetPath)
}

func (e *emitter) targetContext(
	sourceFile *ast.File,
	targetPath string,
) (api.Context, error) {
	names := e.names.ForFile(
		sourceFile,
		e.source.Types().Scope(),
		e.factory,
		targetPath,
		e.require,
	)
	return api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		names,
		e.values,
		e.integer,
		e.order,
	)
}

func (e *emitter) declarationObject(
	context api.Context,
	source ast.Decl,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	switch source := source.(type) {
	case *ast.FuncDecl:
		if len(requirements) != 0 {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function declaration received target requirements",
			}
		}
		function, ok := object.(*types.Func)
		if !ok || context.TypesInfo().Defs[source.Name] != function {
			return api.DeclarationEmission{},
				&api.InvariantError{
					Role:   context.Role(),
					Reason: "scheduled function does not own its declaration",
				}
		}
		return functiondeclaration.Emit(context, e, source)
	case *ast.GenDecl:
		if typeName, ok := object.(*types.TypeName); ok {
			return namedstructdeclaration.EmitAssembly(
				context,
				e,
				source,
				typeName,
				requirements,
			)
		}
		if len(requirements) != 0 {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "non-type declaration received target requirements",
			}
		}
		constant, ok := object.(*types.Const)
		if !ok {
			return api.DeclarationEmission{},
				api.Unsupported(context, api.CategoryDeclaration, source)
		}
		return packageconstant.EmitObject(context, e, source, constant)
	default:
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
}

func (e *emitter) Expression(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	switch source := source.(type) {
	case *ast.BinaryExpr:
		return binaryexpression.Emit(context, e, source)
	case *ast.CallExpr:
		return callexpression.Emit(context, e, source)
	case *ast.CompositeLit:
		return compositeliteral.Emit(context, e, source)
	case *ast.FuncLit:
		return functionliteral.Emit(context, e, source)
	case *ast.Ident:
		return identifierexpression.Emit(context, e, source)
	case *ast.ParenExpr:
		return parenthesizedexpression.Emit(context, e, source)
	case *ast.SelectorExpr:
		return selectorexpression.Emit(context, e, source)
	case *ast.StarExpr:
		return dereferenceexpression.Emit(context, e, source)
	case *ast.BasicLit:
		return e.IntegerConstant(context, source)
	case *ast.UnaryExpr:
		return unaryexpression.Emit(context, e, source)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func (e *emitter) IntegerConstant(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	return integerliteral.Emit(context, e, source)
}

func (e *emitter) DiscardedCall(
	context api.Context,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	return callexpression.EmitDiscarded(context, e, source)
}

func (e *emitter) Condition(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok || basic.Info()&types.IsBoolean == 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return e.Expression(context.WithExpectedType(types.Typ[types.Bool]), source)
}

func (e *emitter) Block(
	context api.Context,
	source *ast.BlockStmt,
) (api.BlockEmission, error) {
	return blockstatement.Emit(context, e, source)
}

func (e *emitter) Statement(
	context api.Context,
	source ast.Stmt,
) (api.StatementEmission, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.Emit(context, e, source)
	case *ast.BlockStmt:
		target, err := blockstatement.Emit(context, e, source)
		if err != nil {
			return api.StatementEmission{}, err
		}
		return api.DirectStatement(target.Value(), target.Requests()...), nil
	case *ast.BranchStmt:
		return branchstatement.Emit(context, source)
	case *ast.DeclStmt:
		declaration, ok := source.Decl.(*ast.GenDecl)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		switch declaration.Tok {
		case token.VAR:
			return localdeclaration.Emit(context, e, source)
		case token.CONST:
			return localconstant.Emit(context, e, source)
		default:
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
	case *ast.ExprStmt:
		return expressionstatement.Emit(context, e, source)
	case *ast.ForStmt:
		return forstatement.Emit(context, e, source)
	case *ast.IfStmt:
		return ifstatement.Emit(context, e, source)
	case *ast.IncDecStmt:
		return incdecstatement.Emit(context, e, source)
	case *ast.ReturnStmt:
		return returnstatement.Emit(context, e, source)
	case *ast.SwitchStmt:
		return switchstatement.Emit(context, e, source)
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *emitter) ScopedInitializer(
	context api.Context,
	source ast.Stmt,
) (api.StatementEmission, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.Emit(context, e, source)
	case *ast.ExprStmt:
		return expressionstatement.Emit(context, e, source)
	case *ast.IncDecStmt:
		return incdecstatement.Emit(context, e, source)
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *emitter) IfAlternate(
	context api.Context,
	source *ast.IfStmt,
) (api.StatementEmission, error) {
	return ifstatement.Emit(context, e, source)
}

func (e *emitter) ForInitializer(
	context api.Context,
	source ast.Stmt,
) (api.ForInitializerEmission, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.EmitForInitializer(context, e, source)
	case *ast.ExprStmt:
		target, err := expressionstatement.EmitExpression(context, e, source)
		if err != nil {
			return api.ForInitializerEmission{}, err
		}
		if len(target.Before()) != 0 {
			return api.ForInitializerEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return api.ExpressionForInitializer(target.Value(), target.Requests()...)
	case *ast.IncDecStmt:
		target, err := incdecstatement.EmitExpression(context, e, source)
		if err != nil {
			return api.ForInitializerEmission{}, err
		}
		return api.ExpressionForInitializer(target.Value(), target.Requests()...)
	default:
		return api.ForInitializerEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *emitter) ForPost(
	context api.Context,
	source ast.Stmt,
) (api.ExpressionEmission, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.EmitExpression(context, e, source)
	case *ast.ExprStmt:
		target, err := expressionstatement.EmitExpression(context, e, source)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(target.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return target, nil
	case *ast.IncDecStmt:
		return incdecstatement.EmitExpression(context, e, source)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *emitter) Type(
	context api.Context,
	source ast.Expr,
) (api.TypeEmission, error) {
	if sourceType := context.TypesInfo().TypeOf(source); sourceType != nil {
		if _, _, ok := pointertype.Scalar(context.TypesSizes(), sourceType); ok {
			pointerSyntax, valid := source.(*ast.StarExpr)
			if !valid {
				return api.TypeEmission{},
					api.Unsupported(context, api.CategoryType, source)
			}
			return pointertype.EmitSyntax(
				context,
				e,
				pointerSyntax,
				sourceType,
			)
		}
		if signature, ok := types.Unalias(sourceType).(*types.Signature); ok {
			functionType, valid := source.(*ast.FuncType)
			if !valid {
				return api.TypeEmission{},
					api.Unsupported(context, api.CategoryType, source)
			}
			return callable.EmitSyntaxType(context, e, functionType, signature)
		}
		if _, ok := types.Unalias(sourceType).(*types.Named); ok {
			return namedstructtype.Emit(context, source, sourceType)
		}
	}
	return basictype.Emit(context, source)
}

func (e *emitter) RepresentedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if tuple, ok := types.Unalias(sourceType).(*types.Tuple); ok {
		return tupletype.Emit(context, e, source, tuple)
	}
	if signature, ok := types.Unalias(sourceType).(*types.Signature); ok {
		return callable.EmitType(context, e, source, signature)
	}
	if _, _, ok := pointertype.Scalar(context.TypesSizes(), sourceType); ok {
		return pointertype.EmitRepresented(context, e, source, sourceType)
	}
	if _, ok := types.Unalias(sourceType).(*types.Named); ok {
		return namedstructtype.Emit(context, source, sourceType)
	}
	return basictype.EmitRepresented(context, source, sourceType)
}

func (e *emitter) StoreTarget(
	context api.Context,
	source ast.Expr,
) (api.StoreTargetEmission, error) {
	return storetarget.Emit(context, e, source)
}
