package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type occurrenceContext struct {
	owner             identity.DefinitionID
	executable        bool
	expected          types.Type
	arity             semantic.ResultArity
	commaOK           bool
	composite         types.Type
	signature         *types.Signature
	bindingRole       identity.SemanticBindingRole
	declaration       catalog.TokenKind
	selectedObject    types.Object
	coverageObject    types.Object
	coverageType      types.Type
	zeroValue         bool
	breakTarget       identity.OccurrenceID
	continueTarget    identity.OccurrenceID
	fallthroughTarget identity.OccurrenceID
}

type contextIndex struct {
	byOccurrence map[identity.OccurrenceID]occurrenceContext
}

func buildContexts(input *packageInput) (*contextIndex, error) {
	out := &contextIndex{
		byOccurrence: map[identity.OccurrenceID]occurrenceContext{},
	}
	visiting := map[identity.OccurrenceID]bool{}
	var assign func(identity.OccurrenceID) error
	assign = func(id identity.OccurrenceID) error {
		if _, present := out.byOccurrence[id]; present {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf(
				"semantic occurrence context cycle at %s", id,
			)
		}
		record := input.occurrences[id]
		if record == nil {
			return fmt.Errorf(
				"semantic context names absent occurrence %s", id,
			)
		}
		visiting[id] = true
		context := occurrenceContext{
			owner: record.owner,
			executable: record.domain ==
				catalog.ResolutionDomainExecutable,
			arity: semantic.ResultArityOne,
		}
		parentID := record.occurrence.Parent()
		if parent := input.occurrences[parentID]; parent != nil {
			if err := assign(parentID); err != nil {
				return err
			}
			context = out.byOccurrence[parentID]
			context.expected = nil
			context.arity = semantic.ResultArityOne
			context.commaOK = false
			context = childContext(
				input,
				parent,
				record,
				context,
			)
		}
		context.owner = record.owner
		context.executable = record.domain ==
			catalog.ResolutionDomainExecutable
		if context.owner != (identity.DefinitionID{}) {
			signature, err := definitionSignature(
				input, context.owner,
			)
			if err != nil {
				return err
			}
			context.signature = signature
			if record.domain == catalog.ResolutionDomainHeader {
				context.coverageObject = definitionObject(
					input, context.owner,
				)
				if context.coverageObject != nil {
					context.coverageType =
						context.coverageObject.Type()
				} else if context.coverageType == nil &&
					signature != nil {
					context.coverageType = signature
				}
			}
		}
		out.byOccurrence[id] = context
		delete(visiting, id)
		for _, child := range record.children {
			if err := assign(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, id := range input.order {
		if err := assign(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func childContext(
	input *packageInput,
	parent *occurrenceInput,
	child *occurrenceInput,
	context occurrenceContext,
) occurrenceContext {
	role := child.occurrence.Role()
	if parent.occurrence.Kind() == catalog.KindGenDecl {
		context.declaration = parent.occurrence.Token()
	}
	switch role {
	case catalog.RoleTypeParameters:
		context.bindingRole = identity.SemanticBindingTypeParameter
	case catalog.RoleParameters:
		context.bindingRole = identity.SemanticBindingParameter
	case catalog.RoleResults:
		context.bindingRole = identity.SemanticBindingResult
	case catalog.RoleReceiver:
		context.bindingRole = identity.SemanticBindingReceiver
	case catalog.RoleImportAlias:
		context.bindingRole = identity.SemanticBindingImport
	case catalog.RoleRangeKey, catalog.RoleRangeValue:
		context.bindingRole = identity.SemanticBindingRange
	case catalog.RoleLabelDeclaration:
		context.bindingRole = identity.SemanticBindingLabel
	}
	view := input.loaded.CheckerView()
	if expression, ok := parent.node.(ast.Expr); ok {
		if value, present := view.TypeOf(expression); present &&
			value.IsType() {
			context.coverageType = value.Type
		}
	}
	switch node := parent.node.(type) {
	case *ast.FuncDecl:
		if role == catalog.RoleFunctionSignature {
			if object, present := view.DefOf(node.Name); present {
				context.coverageObject = object
				context.coverageType = object.Type()
			}
		}
	case *ast.FuncLit:
		if role == catalog.RoleFunctionSignature {
			context.coverageType = expressionType(view, node)
		}
	case *ast.AssignStmt:
		assignAssignmentContext(
			view, node, role, child.occurrence.Ordinal(), &context,
		)
	case *ast.ValueSpec:
		assignValueSpecContext(
			view, node, context.declaration,
			role, child.occurrence.Ordinal(), &context,
		)
		assignValueSpecCoverage(
			view, node, role, child.occurrence.Ordinal(), &context,
		)
	case *ast.TypeSpec:
		assignTypeSpecCoverage(view, node, role, &context)
	case *ast.IndexExpr:
		assignReceiverTypeParameterContext(
			role, &context,
		)
	case *ast.IndexListExpr:
		assignReceiverTypeParameterContext(
			role, &context,
		)
	case *ast.SelectorExpr:
		if role == catalog.RoleSelectedName {
			context.selectedObject = selectorObject(view, node)
		}
	case *ast.CallExpr:
		assignCallContext(
			view, node, role, child.occurrence.Ordinal(), &context,
		)
	case *ast.ReturnStmt:
		assignReturnContext(
			node, role, child.occurrence.Ordinal(), &context,
		)
	case *ast.CompositeLit:
		assignCompositeContext(
			view, node, role, child.occurrence.Ordinal(), &context,
		)
	case *ast.KeyValueExpr:
		assignKeyValueContext(
			view, node, role, &context,
		)
	case *ast.SendStmt:
		if role == catalog.RoleSentValue {
			context.expected = channelElement(view, node.Chan)
		}
	case *ast.IfStmt:
		if role == catalog.RoleCondition {
			context.expected = types.Typ[types.Bool]
		}
	case *ast.ForStmt:
		if role == catalog.RoleCondition {
			context.expected = types.Typ[types.Bool]
		}
		if role == catalog.RoleBody {
			context.breakTarget = parent.occurrence.ID()
			context.continueTarget = parent.occurrence.ID()
		}
	case *ast.RangeStmt:
		assignRangeContext(
			view, node, role, &context,
		)
		if role == catalog.RoleBody {
			context.breakTarget = parent.occurrence.ID()
			context.continueTarget = parent.occurrence.ID()
		}
	case *ast.SwitchStmt, *ast.TypeSwitchStmt:
		if role == catalog.RoleBody {
			context.breakTarget = parent.occurrence.ID()
		}
	case *ast.SelectStmt:
		if role == catalog.RoleBody {
			context.breakTarget = parent.occurrence.ID()
		}
	case *ast.CaseClause:
		if role == catalog.RoleStatement {
			context.fallthroughTarget = nextCase(input, parent)
		}
	}
	return context
}

func assignReceiverTypeParameterContext(
	role catalog.Role,
	context *occurrenceContext,
) {
	if role == catalog.RoleIndex &&
		context.bindingRole == identity.SemanticBindingReceiver {
		context.bindingRole =
			identity.SemanticBindingTypeParameter
	}
}

func assignValueSpecCoverage(
	view checkerExpressionView,
	node *ast.ValueSpec,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	switch role {
	case catalog.RoleTypeExpression:
		context.coverageType = expressionType(view, node.Type)
	case catalog.RoleInitializerValue:
		if ordinal < len(node.Names) {
			object, present := view.DefOf(node.Names[ordinal])
			if present {
				context.coverageObject = object
			}
		}
	}
}

func assignTypeSpecCoverage(
	view checkerExpressionView,
	node *ast.TypeSpec,
	role catalog.Role,
	context *occurrenceContext,
) {
	if role != catalog.RoleTypeExpression &&
		role != catalog.RoleTypeParameters {
		return
	}
	if object, present := view.DefOf(node.Name); present {
		context.coverageObject = object
		context.coverageType = object.Type()
		return
	}
	context.coverageType = expressionType(view, node.Type)
}

func selectorObject(
	view checkerExpressionView,
	node *ast.SelectorExpr,
) types.Object {
	if selection, present := view.SelectionOf(node); present {
		return selection.Obj()
	}
	object, _ := view.UseOf(node.Sel)
	return object
}

func assignAssignmentContext(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	node *ast.AssignStmt,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	switch role {
	case catalog.RoleAssignmentTarget:
		context.bindingRole = identity.SemanticBindingLocal
	case catalog.RoleAssignedValue:
		if len(node.Rhs) == 1 && len(node.Lhs) > 1 {
			assignMultiValueContext(view, node.Rhs[0], context)
			return
		}
		if ordinal < len(node.Lhs) {
			context.expected = expressionType(view, node.Lhs[ordinal])
		}
	}
}

func assignValueSpecContext(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	node *ast.ValueSpec,
	declaration catalog.TokenKind,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	if role == catalog.RoleDeclarationName {
		if declaration == catalog.TokenVAR {
			context.bindingRole = identity.SemanticBindingLocal
			context.zeroValue = len(node.Values) == 0
		}
		return
	}
	if role != catalog.RoleInitializerValue {
		return
	}
	if len(node.Values) == 1 && len(node.Names) > 1 {
		assignMultiValueContext(view, node.Values[0], context)
		return
	}
	if node.Type != nil {
		context.expected = expressionType(view, node.Type)
	}
	if context.expected == nil && ordinal < len(node.Names) {
		if objectView, ok := view.(interface {
			DefOf(*ast.Ident) (types.Object, bool)
		}); ok {
			if object, present := objectView.DefOf(
				node.Names[ordinal],
			); present {
				context.expected = object.Type()
			}
		}
	}
}

func assignMultiValueContext(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	expression ast.Expr,
	context *occurrenceContext,
) {
	value, present := view.TypeOf(expression)
	if present && value.HasOk() {
		context.arity = semantic.ResultArityCommaOk
		context.commaOK = true
	}
}

func assignCallContext(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	node *ast.CallExpr,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	if role == catalog.RoleCallee {
		return
	}
	if role != catalog.RoleCallArgument {
		return
	}
	if value, present := view.TypeOf(node.Fun); present &&
		value.IsType() {
		context.expected = value.Type
		return
	}
	signature := signatureOf(expressionType(view, node.Fun))
	if signature == nil {
		return
	}
	parameters := signature.Params()
	if signature.Variadic() && ordinal >= parameters.Len()-1 {
		if parameters.Len() == 0 {
			return
		}
		last := parameters.At(parameters.Len() - 1).Type()
		if node.Ellipsis.IsValid() {
			context.expected = last
			return
		}
		if slice, ok := types.Unalias(last).Underlying().(*types.Slice); ok {
			context.expected = slice.Elem()
		}
		return
	}
	if ordinal < parameters.Len() {
		context.expected = parameters.At(ordinal).Type()
	}
}

func assignReturnContext(
	node *ast.ReturnStmt,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	if role != catalog.RoleReturnValue ||
		context.signature == nil {
		return
	}
	results := context.signature.Results()
	if len(node.Results) == 1 && results.Len() > 1 {
		context.arity = semantic.ResultArityTuple
		return
	}
	if ordinal < results.Len() {
		context.expected = results.At(ordinal).Type()
	}
}

func assignCompositeContext(
	view checkerExpressionView,
	node *ast.CompositeLit,
	role catalog.Role,
	ordinal int,
	context *occurrenceContext,
) {
	value, present := view.TypeOf(node)
	if !present {
		return
	}
	context.composite = aggregateType(value.Type)
	if role != catalog.RoleCompositeElement {
		return
	}
	context.expected = compositeElementType(
		view, context.composite, node, ordinal,
	)
}

func assignKeyValueContext(
	view checkerExpressionView,
	node *ast.KeyValueExpr,
	role catalog.Role,
	context *occurrenceContext,
) {
	switch aggregate := context.composite.(type) {
	case *types.Map:
		if role == catalog.RoleElementKey {
			context.expected = aggregate.Key()
		} else if role == catalog.RoleElementValue {
			context.expected = aggregate.Elem()
		}
	case *types.Array:
		if role == catalog.RoleElementKey {
			context.expected = types.Typ[types.Int]
		} else {
			context.expected = aggregate.Elem()
		}
	case *types.Slice:
		if role == catalog.RoleElementKey {
			context.expected = types.Typ[types.Int]
		} else {
			context.expected = aggregate.Elem()
		}
	case *types.Struct:
		if role != catalog.RoleElementValue {
			return
		}
		if identifier, ok := node.Key.(*ast.Ident); ok {
			if object, present := view.UseOf(identifier); present {
				if field, ok := object.(*types.Var); ok &&
					field.IsField() {
					context.expected = field.Type()
				}
			}
		}
	}
}

func assignRangeContext(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	node *ast.RangeStmt,
	role catalog.Role,
	context *occurrenceContext,
) {
	if role == catalog.RoleRangeKey ||
		role == catalog.RoleRangeValue {
		context.bindingRole = identity.SemanticBindingRange
	}
	operand := aggregateType(expressionType(view, node.X))
	switch value := operand.(type) {
	case *types.Array:
		if role == catalog.RoleRangeKey {
			context.expected = types.Typ[types.Int]
		} else if role == catalog.RoleRangeValue {
			context.expected = value.Elem()
		}
	case *types.Slice:
		if role == catalog.RoleRangeKey {
			context.expected = types.Typ[types.Int]
		} else if role == catalog.RoleRangeValue {
			context.expected = value.Elem()
		}
	case *types.Map:
		if role == catalog.RoleRangeKey {
			context.expected = value.Key()
		} else if role == catalog.RoleRangeValue {
			context.expected = value.Elem()
		}
	case *types.Chan:
		if role == catalog.RoleRangeKey {
			context.expected = value.Elem()
		}
	case *types.Basic:
		if value.Info()&types.IsString != 0 {
			if role == catalog.RoleRangeKey {
				context.expected = types.Typ[types.Int]
			} else if role == catalog.RoleRangeValue {
				context.expected = types.Typ[types.Rune]
			}
		}
	}
}
