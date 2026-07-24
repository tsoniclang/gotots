package frontend

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (builder *packageBuilder) buildDefinitions() error {
	boundaries := map[identity.DefinitionID]structure.ExecutionBoundary{}
	for _, boundary := range builder.input.graph.Boundaries() {
		if builder.input.definition(boundary.ID().Definition()) != nil {
			boundaries[boundary.ID().Definition()] = boundary
		}
	}
	for _, definition := range sortedDefinitions(
		builder.input.definitions,
	) {
		spec, err := builder.definitionSpec(
			definition, boundaries[definition.ID()],
		)
		if err != nil {
			return err
		}
		record, err := semantic.NewDefinitionSemantics(spec)
		if err != nil {
			return fmt.Errorf(
				"materialize definition %s: %w",
				definition.ID(), err,
			)
		}
		if err := builder.draft.AddDefinition(record); err != nil {
			return err
		}
	}
	return nil
}

func (builder *packageBuilder) definitionSpec(
	definition structure.ImplementationDefinition,
	boundary structure.ExecutionBoundary,
) (semantic.DefinitionSemanticsSpec, error) {
	spec := semantic.DefinitionSemanticsSpec{
		Definition: definition.ID(),
		Package:    builder.input.id,
		Authority:  builder.input.authority,
		Name:       definition.Name(),
	}
	bindings := builder.definitionBindings(definition.ID())
	spec.Bindings = bindings
	switch definition.ID().Kind() {
	case identity.DefinitionFuncDecl:
		spec.Form = semantic.DefinitionFormCallable
		declaration, signature, receiver, err :=
			builder.callableDefinition(definition.ID(), true)
		if err != nil {
			return semantic.DefinitionSemanticsSpec{}, err
		}
		if !declaration.IsZero() {
			spec.Declarations = []identity.SemanticDeclarationID{
				declaration,
			}
		}
		spec.Signature = signature
		spec.Receiver = receiver
	case identity.DefinitionFuncLit:
		spec.Form = semantic.DefinitionFormCallable
		_, signature, _, err :=
			builder.callableDefinition(definition.ID(), false)
		if err != nil {
			return semantic.DefinitionSemanticsSpec{}, err
		}
		spec.Signature = signature
	case identity.DefinitionPackageInitializer:
		spec.Form = semantic.DefinitionFormInitializer
		declarations, err :=
			builder.initializerDeclarations(definition.ID())
		if err != nil {
			return semantic.DefinitionSemanticsSpec{}, err
		}
		spec.Declarations = declarations
		for _, entry := range boundary.Entries() {
			spec.InitializerEntries = append(
				spec.InitializerEntries, entry.ID(),
			)
		}
	case identity.DefinitionBodylessDecl:
		spec.Form = semantic.DefinitionFormBodyless
		declaration, signature, receiver, err :=
			builder.callableDefinition(definition.ID(), true)
		if err != nil {
			return semantic.DefinitionSemanticsSpec{}, err
		}
		spec.Declarations = []identity.SemanticDeclarationID{
			declaration,
		}
		spec.Signature = signature
		spec.Receiver = receiver
	case identity.DefinitionImplicit:
		if definition.ID().ImplicitOp().Valid() {
			spec.Form = semantic.DefinitionFormImplicit
			spec.Implicit = definition.ID().ImplicitOp()
			break
		}
		if definition.ID().SyntheticRole().Valid() {
			spec.Form = semantic.DefinitionFormSynthetic
			declaration, signature, err :=
				builder.syntheticDefinition(definition.ID())
			if err != nil {
				return semantic.DefinitionSemanticsSpec{}, err
			}
			spec.Declarations = []identity.SemanticDeclarationID{
				declaration,
			}
			spec.Signature = signature
			break
		}
		fallthrough
	default:
		return semantic.DefinitionSemanticsSpec{}, fmt.Errorf(
			"definition %s has no semantic form",
			definition.ID(),
		)
	}
	return spec, nil
}

func (builder *packageBuilder) callableDefinition(
	definition identity.DefinitionID,
	declared bool,
) (
	identity.SemanticDeclarationID,
	identity.SemanticTypeID,
	identity.SemanticBindingID,
	error,
) {
	node, present := builder.input.index.CheckedDefinitionNode(
		definition,
	)
	if !present {
		node, present = builder.input.index.DefinitionNode(definition)
	}
	if !present {
		return identity.SemanticDeclarationID{},
			identity.SemanticTypeID{},
			identity.SemanticBindingID{},
			fmt.Errorf(
				"callable definition %s has no transient root",
				definition,
			)
	}
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, present := builder.input.loaded.CheckerView().
			DefOf(node.Name)
		if !present || object == nil {
			object = intrinsicDefinitionObject(
				builder.input, node.Name.Name,
			)
		}
		if object == nil {
			return identity.SemanticDeclarationID{},
				identity.SemanticTypeID{},
				identity.SemanticBindingID{},
				fmt.Errorf(
					"callable definition %s has no checker object",
					definition,
				)
		}
		declaration := identity.SemanticDeclarationID{}
		if node.Name.Name != "_" && node.Name.Name != "init" {
			var err error
			declaration, err = builder.objects.declarationID(object)
			if err != nil {
				return identity.SemanticDeclarationID{},
					identity.SemanticTypeID{},
					identity.SemanticBindingID{}, err
			}
		}
		signature := identity.SemanticTypeID{}
		if _, builtin := object.(*types.Builtin); !builtin {
			builtSignature, err := builder.types.build(object.Type())
			if err != nil {
				return identity.SemanticDeclarationID{},
					identity.SemanticTypeID{},
					identity.SemanticBindingID{}, err
			}
			signature = builtSignature
		}
		receiver := identity.SemanticBindingID{}
		if typed, ok := object.Type().(*types.Signature); ok &&
			typed.Recv() != nil {
			receiver, _ = builder.objects.bindingID(typed.Recv())
			if receiver.IsZero() {
				return identity.SemanticDeclarationID{},
					identity.SemanticTypeID{},
					identity.SemanticBindingID{},
					fmt.Errorf(
						"method definition %s has no receiver binding",
						definition,
					)
			}
		}
		return declaration, signature, receiver, nil
	case *ast.FuncLit:
		if declared {
			return identity.SemanticDeclarationID{},
				identity.SemanticTypeID{},
				identity.SemanticBindingID{},
				fmt.Errorf(
					"declared callable %s resolved to a function literal",
					definition,
				)
		}
		typ := expressionType(
			builder.input.loaded.CheckerView(), node,
		)
		signature, err := builder.types.build(typ)
		return identity.SemanticDeclarationID{},
			signature,
			identity.SemanticBindingID{},
			err
	default:
		return identity.SemanticDeclarationID{},
			identity.SemanticTypeID{},
			identity.SemanticBindingID{},
			fmt.Errorf(
				"callable definition %s has root %T",
				definition, node,
			)
	}
}

func (builder *packageBuilder) initializerDeclarations(
	definition identity.DefinitionID,
) ([]identity.SemanticDeclarationID, error) {
	node, present := builder.input.index.DefinitionNode(definition)
	if !present {
		node, present = builder.input.index.CheckedDefinitionNode(
			definition,
		)
	}
	valueSpec, ok := node.(*ast.ValueSpec)
	if !present || !ok {
		return nil, fmt.Errorf(
			"initializer definition %s has root %T",
			definition, node,
		)
	}
	out := make(
		[]identity.SemanticDeclarationID, 0, len(valueSpec.Names),
	)
	for _, name := range valueSpec.Names {
		if name.Name == "_" {
			continue
		}
		object, present := builder.input.loaded.CheckerView().
			DefOf(name)
		if !present {
			return nil, fmt.Errorf(
				"initializer definition %s name has no checker object",
				definition,
			)
		}
		declaration, err := builder.objects.declarationID(object)
		if err != nil {
			return nil, err
		}
		out = append(out, declaration)
	}
	return out, nil
}

func (builder *packageBuilder) syntheticDefinition(
	definition identity.DefinitionID,
) (
	identity.SemanticDeclarationID,
	identity.SemanticTypeID,
	error,
) {
	node, present := builder.input.index.SyntheticDefinitionNode(
		definition,
	)
	if !present {
		return identity.SemanticDeclarationID{},
			identity.SemanticTypeID{},
			fmt.Errorf(
				"synthetic definition %s has no checked root",
				definition,
			)
	}
	object := declarationObject(
		builder.input.loaded.CheckerView(),
		node,
		definition.SyntheticName(),
	)
	if object == nil {
		return identity.SemanticDeclarationID{},
			identity.SemanticTypeID{},
			fmt.Errorf(
				"synthetic definition %s has no checker declaration",
				definition,
			)
	}
	declaration, err := builder.objects.declarationID(object)
	if err != nil {
		return identity.SemanticDeclarationID{},
			identity.SemanticTypeID{}, err
	}
	signature := identity.SemanticTypeID{}
	if definition.SyntheticRole() ==
		identity.SyntheticDefinitionAdapter {
		signature, err = builder.types.build(object.Type())
	}
	return declaration, signature, err
}

func declarationObject(
	view checkerExpressionView,
	node ast.Node,
	name string,
) types.Object {
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, _ := view.DefOf(node.Name)
		return object
	case *ast.TypeSpec:
		object, _ := view.DefOf(node.Name)
		return object
	case *ast.ValueSpec:
		for _, identifier := range node.Names {
			object, present := view.DefOf(identifier)
			if present && object.Name() == name {
				return object
			}
		}
	}
	return nil
}

func (builder *packageBuilder) definitionBindings(
	definition identity.DefinitionID,
) []identity.SemanticBindingID {
	var out []identity.SemanticBindingID
	for candidate, id := range builder.objects.bindingIDs {
		if candidate.definition == definition {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].Compare(out[right]) < 0
	})
	return out
}
