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
	executable        bool
	expected          types.Type
	arity             semantic.ResultArity
	commaOK           bool
	composite         types.Type
	signature         *types.Signature
	bindingRole       identity.SemanticBindingRole
	declaration       catalog.TokenKind
	selectedObject    types.Object
	selectedSelection *types.Selection
	coverageObject    types.Object
	coverageType      types.Type
	memberOwner       types.Type
	compileTime       bool
	zeroValue         bool
	breakTarget       packageOccurrenceRef
	continueTarget    packageOccurrenceRef
	fallthroughTarget packageOccurrenceRef
	typeSwitchAnchor  bool
}

type contextIndex struct {
	input *packageInput
	count int
}

func (index *contextIndex) context(
	id identity.OccurrenceID,
) occurrenceContext {
	return index.contextAt(index.input.occurrenceReference(id))
}

func (index *contextIndex) contextAt(
	reference packageOccurrenceRef,
) occurrenceContext {
	record := index.input.occurrenceRecord(reference)
	if record == nil || !record.contextAssigned {
		return occurrenceContext{}
	}
	return record.context
}

func buildContexts(
	input *packageInput,
	work *Work,
) (*contextIndex, error) {
	out := &contextIndex{input: input}
	var assign func(packageOccurrenceRef) error
	assign = func(reference packageOccurrenceRef) error {
		record := input.occurrenceRecord(reference)
		if record == nil {
			return fmt.Errorf(
				"semantic context names absent occurrence reference %d",
				reference,
			)
		}
		if record.contextAssigned {
			return nil
		}
		if record.contextVisiting {
			return fmt.Errorf(
				"semantic occurrence context cycle at %s",
				record.occurrence.ID(),
			)
		}
		record.contextVisiting = true
		context := occurrenceContext{
			executable: record.domain ==
				catalog.ResolutionDomainExecutable,
			arity: semantic.ResultArityOne,
		}
		parentReference := input.occurrenceParent(reference)
		if parent := input.occurrenceRecord(parentReference); parent != nil {
			if err := assign(parentReference); err != nil {
				return err
			}
			context = parent.context
			context.expected = nil
			context.arity = semantic.ResultArityOne
			context.commaOK = false
			context = childContext(
				input,
				parentReference,
				parent,
				record,
				context,
			)
		}
		context.executable = record.domain ==
			catalog.ResolutionDomainExecutable
		owner := input.occurrenceOwner(record)
		if !owner.IsZero() {
			signature, err := definitionSignature(
				input, owner,
			)
			if err != nil {
				return err
			}
			context.signature = signature
			if record.domain == catalog.ResolutionDomainHeader {
				if context.coverageObject == nil &&
					context.coverageType == nil {
					context.coverageObject = definitionObject(
						input, owner,
					)
					if context.coverageObject != nil {
						context.coverageType =
							context.coverageObject.Type()
					} else if signature != nil {
						context.coverageType = signature
					}
				}
			}
		}
		record.context = context
		record.contextAssigned = true
		record.contextVisiting = false
		out.count++
		work.ContextAssignments++
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
	parentReference packageOccurrenceRef,
	parent *occurrenceInput,
	child *occurrenceInput,
	context occurrenceContext,
) occurrenceContext {
	role := child.occurrence.Role()
	if parent.occurrence.Kind() == catalog.KindGenDecl {
		context.declaration = parent.occurrence.Token()
	}
	if role == catalog.RoleArrayLength ||
		(role == catalog.RoleInitializerValue &&
			context.declaration == catalog.TokenCONST) {
		context.compileTime = true
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
	if role == catalog.RoleArrayLength {
		context.coverageObject = nil
		if expression, ok := parent.node.(ast.Expr); ok {
			context.coverageType = expressionType(view, expression)
		}
	}
	switch node := parent.node.(type) {
	case *ast.TypeSpec:
		if role == catalog.RoleTypeExpression {
			if object, present := view.DefOf(node.Name); present {
				owner := originMemberOwner(object.Type())
				switch types.Unalias(owner).Underlying().(type) {
				case *types.Struct, *types.Interface:
					context.memberOwner = owner
				default:
					context.memberOwner = nil
				}
			}
		}
		assignTypeSpecCoverage(view, node, role, &context)
	case *ast.StructType:
		if role == catalog.RoleStructFields &&
			context.memberOwner == nil {
			if owner := expressionType(view, node); owner != nil {
				context.memberOwner = originMemberOwner(owner)
			}
		}
	case *ast.InterfaceType:
		if role == catalog.RoleInterfaceMethods &&
			context.memberOwner == nil {
			if owner := expressionType(view, node); owner != nil {
				context.memberOwner = originMemberOwner(owner)
			}
		}
	case *ast.Field:
		if role == catalog.RoleTypeExpression {
			context.memberOwner = nil
		}
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
			context.selectedSelection, _ = view.SelectionOf(node)
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
			context.breakTarget = parentReference
			context.continueTarget = parentReference
		}
	case *ast.RangeStmt:
		assignRangeContext(
			view, node, role, &context,
		)
		if role == catalog.RoleBody {
			context.breakTarget = parentReference
			context.continueTarget = parentReference
		}
	case *ast.SwitchStmt:
		if role == catalog.RoleBody {
			context.breakTarget = parentReference
		}
	case *ast.TypeSwitchStmt:
		if role == catalog.RoleBody {
			context.breakTarget = parentReference
		}
		if role == catalog.RoleTypeSwitchGuard {
			context.typeSwitchAnchor = true
		}
	case *ast.SelectStmt:
		if role == catalog.RoleBody {
			context.breakTarget = parentReference
		}
	case *ast.CaseClause:
		if role == catalog.RoleStatement {
			context.fallthroughTarget = nextCase(
				input, parentReference,
			)
		}
	}
	return context
}
