package emit

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	definedtypedeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/definedtype"
	functiondeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/function"
	interfacetypedeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/interfacetype"
	namedstructdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	packageconstant "github.com/tsoniclang/gotots/internal/emit/declaration/packageconstant"
	addressexpression "github.com/tsoniclang/gotots/internal/emit/expression/address"
	binaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/binary"
	callexpression "github.com/tsoniclang/gotots/internal/emit/expression/call"
	compositeliteral "github.com/tsoniclang/gotots/internal/emit/expression/compositeliteral"
	dereferenceexpression "github.com/tsoniclang/gotots/internal/emit/expression/dereference"
	functionliteral "github.com/tsoniclang/gotots/internal/emit/expression/functionliteral"
	identifierexpression "github.com/tsoniclang/gotots/internal/emit/expression/identifier"
	indexexpression "github.com/tsoniclang/gotots/internal/emit/expression/index"
	complexliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/complex"
	floatliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/float"
	integerliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/integer"
	stringliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/string"
	parenthesizedexpression "github.com/tsoniclang/gotots/internal/emit/expression/parenthesized"
	selectorexpression "github.com/tsoniclang/gotots/internal/emit/expression/selector"
	sliceexpression "github.com/tsoniclang/gotots/internal/emit/expression/slice"
	typeassertion "github.com/tsoniclang/gotots/internal/emit/expression/typeassertion"
	unaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/unary"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	blockstatement "github.com/tsoniclang/gotots/internal/emit/statement/block"
	branchstatement "github.com/tsoniclang/gotots/internal/emit/statement/branch"
	expressionstatement "github.com/tsoniclang/gotots/internal/emit/statement/expressionstatement"
	forstatement "github.com/tsoniclang/gotots/internal/emit/statement/forstatement"
	ifstatement "github.com/tsoniclang/gotots/internal/emit/statement/ifstatement"
	incdecstatement "github.com/tsoniclang/gotots/internal/emit/statement/incdec"
	labelstatement "github.com/tsoniclang/gotots/internal/emit/statement/label"
	localconstant "github.com/tsoniclang/gotots/internal/emit/statement/localconstant"
	localdeclaration "github.com/tsoniclang/gotots/internal/emit/statement/localdeclaration"
	localtype "github.com/tsoniclang/gotots/internal/emit/statement/localtype"
	rangestatement "github.com/tsoniclang/gotots/internal/emit/statement/range"
	returnstatement "github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
	switchstatement "github.com/tsoniclang/gotots/internal/emit/statement/switchstatement"
	typeswitchstatement "github.com/tsoniclang/gotots/internal/emit/statement/typeswitchstatement"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	storetarget "github.com/tsoniclang/gotots/internal/emit/store"
	anonymousstructtype "github.com/tsoniclang/gotots/internal/emit/type/anonymousstruct"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	maptype "github.com/tsoniclang/gotots/internal/emit/type/map"
	namedstructtype "github.com/tsoniclang/gotots/internal/emit/type/namedstruct"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	slicetype "github.com/tsoniclang/gotots/internal/emit/type/slice"
	tupletype "github.com/tsoniclang/gotots/internal/emit/type/tuple"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/representation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type emitter struct {
	source  *load.Package
	factory tsgo.Factory
	names   *emitnaming.Owner
	values  api.Values
	integer api.IntegerRepresentation
	order   api.EvaluationOrder
	require func(types.Object) error
}

func newEmitter(
	source *load.Package,
	factory tsgo.Factory,
	registry *emitnaming.Registry,
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
	target := &emitter{
		source:  source,
		factory: factory,
		names:   emitnaming.NewOwner(packageScope, typesInfo, registry),
		integer: integer,
		order:   order,
		require: require,
	}
	target.values = representation.NewOwner(target)
	return target
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
	return e.context(names)
}

func (e *emitter) generatedContext(
	targetPath string,
	registry *emitnaming.Registry,
) (api.Context, error) {
	names := emitnaming.NewOwner(
		nil,
		nil,
		registry,
	).ForFile(
		nil,
		nil,
		e.factory,
		targetPath,
		e.require,
	)
	return e.context(names)
}

func (e *emitter) context(names api.Names) (api.Context, error) {
	return api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		names,
		e.values,
		storage.Owner{},
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
		function, ok := object.(*types.Func)
		if !ok || context.TypesInfo().Defs[source.Name] != function {
			return api.DeclarationEmission{},
				&api.InvariantError{
					Role:   context.Role(),
					Reason: "scheduled function does not own its declaration",
				}
		}
		return functiondeclaration.Emit(
			context,
			e,
			source,
			requirements,
		)
	case *ast.GenDecl:
		if typeName, ok := object.(*types.TypeName); ok {
			if target, handled, err := interfacetypedeclaration.Emit(
				context,
				e,
				source,
				typeName,
				requirements,
			); handled {
				return target, err
			}
			if target, handled, err := definedtypedeclaration.Emit(
				context,
				e,
				source,
				typeName,
				requirements,
			); handled {
				return target, err
			}
			return namedstructdeclaration.EmitAssembly(
				context,
				e,
				source,
				typeName,
				requirements,
			)
		}
		if constant, ok := object.(*types.Const); ok {
			return packageconstant.EmitObject(
				context,
				e,
				source,
				constant,
				requirements,
			)
		}
		if len(requirements) != 0 {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "non-type non-constant declaration received target requirements",
			}
		}
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	default:
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
}

func (e *emitter) Expression(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	operandContext := interfacevalue.OperandContext(context, source)
	adapt := func(
		target api.ExpressionEmission,
		err error,
	) (api.ExpressionEmission, error) {
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return interfacevalue.AdaptExpected(context, source, target)
	}

	switch source := source.(type) {
	case *ast.BinaryExpr:
		return adapt(binaryexpression.Emit(operandContext, e, source))
	case *ast.CallExpr:
		return adapt(callexpression.Emit(operandContext, e, source))
	case *ast.CompositeLit:
		return adapt(compositeliteral.Emit(operandContext, e, source))
	case *ast.FuncLit:
		return adapt(functionliteral.Emit(operandContext, e, source))
	case *ast.Ident:
		return adapt(identifierexpression.Emit(operandContext, e, source))
	case *ast.IndexExpr:
		return adapt(indexexpression.Emit(operandContext, e, source))
	case *ast.ParenExpr:
		return adapt(parenthesizedexpression.Emit(operandContext, e, source))
	case *ast.SelectorExpr:
		return adapt(selectorexpression.Emit(operandContext, e, source))
	case *ast.SliceExpr:
		return adapt(sliceexpression.Emit(operandContext, e, source))
	case *ast.StarExpr:
		return adapt(dereferenceexpression.Emit(operandContext, e, source))
	case *ast.TypeAssertExpr:
		return adapt(typeassertion.Emit(operandContext, e, source))
	case *ast.BasicLit:
		if source.Kind == token.STRING {
			return adapt(stringliteral.Emit(operandContext, e, source))
		}
		if source.Kind == token.IMAG {
			return adapt(complexliteral.Emit(operandContext, e, source))
		}
		if source.Kind == token.FLOAT {
			return adapt(floatliteral.Emit(operandContext, e, source))
		}
		return adapt(e.IntegerConstant(operandContext, source))
	case *ast.UnaryExpr:
		return adapt(unaryexpression.Emit(operandContext, e, source))
	default:
		return api.ExpressionEmission{},
			api.Unsupported(operandContext, api.CategoryExpression, source)
	}
}

func (e *emitter) IntegerConstant(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	return integerliteral.Emit(context, e, source)
}

func (e *emitter) Address(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	return addressexpression.Emit(context, e, source)
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
		case token.TYPE:
			return localtype.Emit(context, e, source)
		default:
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
	case *ast.ExprStmt:
		return expressionstatement.Emit(context, e, source)
	case *ast.EmptyStmt:
		return api.NewStatementEmission(nil, nil)
	case *ast.ForStmt:
		return forstatement.Emit(context, e, source)
	case *ast.IfStmt:
		return ifstatement.Emit(context, e, source)
	case *ast.IncDecStmt:
		return incdecstatement.Emit(context, e, source)
	case *ast.LabeledStmt:
		return labelstatement.Emit(context, e, source)
	case *ast.RangeStmt:
		return rangestatement.Emit(context, e, source)
	case *ast.ReturnStmt:
		return returnstatement.Emit(context, e, source)
	case *ast.SwitchStmt:
		return switchstatement.Emit(context, e, source)
	case *ast.TypeSwitchStmt:
		return typeswitchstatement.Emit(context, e, source)
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

func (e *emitter) Type(
	context api.Context,
	source ast.Expr,
) (api.TypeEmission, error) {
	if sourceType := context.TypesInfo().TypeOf(source); sourceType != nil {
		if array, ok := arrayvalue.Resolve(context, sourceType); ok {
			return array.EmitType(context, e, source)
		}
		if _, _, ok := pointertype.Resolve(sourceType); ok {
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
		if _, ok := types.Unalias(sourceType).(*types.Named); ok {
			if target, handled, err := definedtype.Emit(
				context,
				source,
				sourceType,
			); handled {
				return target, err
			}
			return namedstructtype.Emit(context, source, sourceType)
		}
		if signature, ok := types.Unalias(sourceType).(*types.Signature); ok {
			functionType, valid := source.(*ast.FuncType)
			if !valid {
				return api.TypeEmission{},
					api.Unsupported(context, api.CategoryType, source)
			}
			return callable.EmitSyntaxType(context, e, functionType, signature)
		}
		if _, ok := types.Unalias(sourceType).(*types.Map); ok {
			return maptype.Emit(context, source, sourceType)
		}
		if sliceType, ok := types.Unalias(sourceType).(*types.Slice); ok {
			arrayType, valid := source.(*ast.ArrayType)
			if !valid {
				return api.TypeEmission{},
					api.Unsupported(context, api.CategoryType, source)
			}
			return slicetype.EmitSyntax(
				context,
				e,
				arrayType,
				sliceType,
			)
		}
		if _, ok := anonymousstructtype.Resolve(sourceType); ok {
			structType, valid := source.(*ast.StructType)
			if !valid {
				return api.TypeEmission{},
					api.Unsupported(context, api.CategoryType, source)
			}
			return anonymousstructtype.Emit(
				context,
				e,
				structType,
				sourceType,
			)
		}
	}
	return basictype.Emit(context, source)
}

func (e *emitter) RepresentedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.EmitType(context, e, source)
	}
	if tuple, ok := types.Unalias(sourceType).(*types.Tuple); ok {
		return tupletype.Emit(context, e, source, tuple)
	}
	if _, _, ok := pointertype.Resolve(sourceType); ok {
		return pointertype.EmitRepresented(context, e, source, sourceType)
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return interfacetype.Emit(context, source, sourceType)
	}
	if _, ok := types.Unalias(sourceType).(*types.Named); ok {
		if target, handled, err := definedtype.Emit(
			context,
			source,
			sourceType,
		); handled {
			return target, err
		}
		return namedstructtype.Emit(context, source, sourceType)
	}
	if signature, ok := types.Unalias(sourceType).(*types.Signature); ok {
		return callable.EmitType(context, e, source, signature)
	}
	if _, ok := types.Unalias(sourceType).(*types.Map); ok {
		return maptype.Emit(context, source, sourceType)
	}
	if _, ok := types.Unalias(sourceType).(*types.Slice); ok {
		return slicetype.EmitRepresented(context, e, source, sourceType)
	}
	if _, ok := anonymousstructtype.Resolve(sourceType); ok {
		return anonymousstructtype.Emit(context, e, source, sourceType)
	}
	return basictype.EmitRepresented(context, source, sourceType)
}

func (e *emitter) StoreTarget(
	context api.Context,
	source ast.Expr,
) (api.StoreTargetEmission, error) {
	return storetarget.Emit(context, e, source)
}
