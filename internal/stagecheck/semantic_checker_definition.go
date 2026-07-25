package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/source"
)

func (verifier *checkerSemanticVerifier) verifyDefinitions() error {
	seen := 0
	err := verifier.actual.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		definition := record.Definition()
		structural, present := verifier.expected.definitions[definition]
		if !present {
			if verifier.localOnly {
				return nil
			}
			return fmt.Errorf(
				"definition %s has no structural expectation",
				definition,
			)
		}
		seen++
		spec := record.Spec()
		if spec.Name != structural.Name() ||
			spec.Implicit != definition.ImplicitOp() {
			return fmt.Errorf(
				"definition %s name or implicit meaning differs",
				definition,
			)
		}
		if err := verifier.verifyDefinitionBindings(
			definition, spec,
		); err != nil {
			return err
		}
		node, present := verifier.definitionNode(definition)
		if !present {
			if definition.ImplicitOp().Valid() {
				if len(spec.Declarations) != 0 ||
					!spec.Signature.IsZero() ||
					!spec.Receiver.IsZero() ||
					len(spec.InitializerEntries) != 0 {
					return fmt.Errorf(
						"implicit definition %s carries source semantics",
						definition,
					)
				}
				return nil
			}
			return fmt.Errorf(
				"definition %s has no checker node", definition,
			)
		}
		if err := verifier.verifyDefinitionNode(
			definition, spec, node,
		); err != nil {
			return fmt.Errorf(
				"definition %s: %w", definition, err,
			)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(verifier.expected.definitions) {
		return fmt.Errorf(
			"checker visited %d definitions for %d structural expectations",
			seen,
			len(verifier.expected.definitions),
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) definitionNode(
	definition identity.DefinitionID,
) (ast.Node, bool) {
	node, present := verifier.index.CheckedDefinitionNode(definition)
	if !present {
		node, present = verifier.index.DefinitionNode(definition)
	}
	if !present && definition.SyntheticRole().Valid() {
		node, present = verifier.index.SyntheticDefinitionNode(
			definition,
		)
	}
	return node, present
}

func (verifier *checkerSemanticVerifier) verifyDefinitionBindings(
	definition identity.DefinitionID,
	spec semantic.DefinitionSemanticsSpec,
) error {
	expected := verifier.bindingsByDefinition[definition]
	if !slices.Equal(spec.Bindings, expected) {
		return fmt.Errorf(
			"definition %s bindings=%v, checker-derived=%v",
			definition, spec.Bindings, expected,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyDefinitionNode(
	definition identity.DefinitionID,
	spec semantic.DefinitionSemanticsSpec,
	node ast.Node,
) error {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		return verifier.verifyCallableDefinition(
			definition, spec, typed,
		)
	case *ast.FuncLit:
		if len(spec.Declarations) != 0 ||
			!spec.Receiver.IsZero() ||
			len(spec.InitializerEntries) != 0 {
			return fmt.Errorf(
				"function literal carries declaration, receiver, or initializer",
			)
		}
		return verifier.types.verify(
			spec.Signature,
			independentDefinitionType(verifier.view, typed),
		)
	case *ast.ValueSpec:
		return verifier.verifyValueDefinition(
			definition, spec, typed,
		)
	case *ast.TypeSpec:
		return verifier.verifySyntheticDeclaration(
			definition, spec, typed,
		)
	default:
		return fmt.Errorf("unsupported checker definition root %T", node)
	}
}

func (verifier *checkerSemanticVerifier) verifyCallableDefinition(
	definition identity.DefinitionID,
	spec semantic.DefinitionSemanticsSpec,
	node *ast.FuncDecl,
) error {
	object := verifier.independentDefinitionObject(node.Name)
	if object == nil {
		return fmt.Errorf("callable has no checker object")
	}
	wantDeclarations := 1
	if node.Name.Name == "_" || independentPackageInitializer(node) {
		wantDeclarations = 0
	}
	if definition.SyntheticRole().Valid() {
		wantDeclarations = 1
	}
	if len(spec.Declarations) != wantDeclarations {
		return fmt.Errorf(
			"callable declarations=%d, checker-derived=%d",
			len(spec.Declarations), wantDeclarations,
		)
	}
	if wantDeclarations == 1 {
		if err := verifier.verifyObjectReference(
			mustDeclarationReference(spec.Declarations[0]),
			object,
		); err != nil {
			return err
		}
	}
	if len(spec.InitializerEntries) != 0 {
		return fmt.Errorf("callable carries initializer entries")
	}
	if err := verifier.types.verify(
		spec.Signature,
		independentDefinitionType(verifier.view, node),
	); err != nil {
		return err
	}
	signature, _ := object.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		if !spec.Receiver.IsZero() {
			return fmt.Errorf("receiverless callable carries receiver binding")
		}
		return nil
	}
	return verifier.verifyObjectReference(
		mustBindingReference(spec.Receiver),
		signature.Recv(),
	)
}

func (verifier *checkerSemanticVerifier) verifyValueDefinition(
	definition identity.DefinitionID,
	spec semantic.DefinitionSemanticsSpec,
	node *ast.ValueSpec,
) error {
	var objects []types.Object
	for _, name := range node.Names {
		if name.Name == "_" {
			continue
		}
		object, present := verifier.view.DefOf(name)
		if !present {
			return fmt.Errorf(
				"initializer name %s has no checker object",
				name.Name,
			)
		}
		objects = append(objects, object)
	}
	if len(spec.Declarations) != len(objects) {
		return fmt.Errorf(
			"initializer declarations=%d, checker-derived=%d",
			len(spec.Declarations), len(objects),
		)
	}
	for index, object := range objects {
		if err := verifier.verifyObjectReference(
			mustDeclarationReference(spec.Declarations[index]),
			object,
		); err != nil {
			return err
		}
	}
	if !spec.Signature.IsZero() || !spec.Receiver.IsZero() {
		return fmt.Errorf("initializer carries callable semantics")
	}
	expected := verifier.expected.initializers[definition]
	if !slices.Equal(spec.InitializerEntries, expected) {
		return fmt.Errorf(
			"initializer entries=%v, Stage-1=%v",
			spec.InitializerEntries, expected,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifySyntheticDeclaration(
	definition identity.DefinitionID,
	spec semantic.DefinitionSemanticsSpec,
	node *ast.TypeSpec,
) error {
	object, present := verifier.view.DefOf(node.Name)
	if !present {
		return fmt.Errorf("synthetic declaration has no checker object")
	}
	if len(spec.Declarations) != 1 {
		return fmt.Errorf(
			"synthetic declaration count=%d, want 1",
			len(spec.Declarations),
		)
	}
	if err := verifier.verifyObjectReference(
		mustDeclarationReference(spec.Declarations[0]),
		object,
	); err != nil {
		return err
	}
	if definition.SyntheticRole() ==
		identity.SyntheticDefinitionAdapter {
		return verifier.types.verify(spec.Signature, object.Type())
	}
	if !spec.Signature.IsZero() {
		return fmt.Errorf(
			"non-adapter synthetic definition carries signature",
		)
	}
	return nil
}

func independentDefinitionType(
	view *source.TypeInfoView,
	node ast.Node,
) types.Type {
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, _ := view.DefOf(node.Name)
		if _, builtin := object.(*types.Builtin); builtin {
			return nil
		}
		if object != nil {
			return object.Type()
		}
	case *ast.FuncLit:
		if value, present := view.TypeOf(node); present {
			return value.Type
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentDefinitionObject(
	name *ast.Ident,
) types.Object {
	if object, present := verifier.view.DefOf(name); present {
		return object
	}
	if verifier.expected.loaded.Types() == nil {
		return nil
	}
	return verifier.expected.loaded.Types().Scope().Lookup(name.Name)
}

func independentPackageInitializer(declaration *ast.FuncDecl) bool {
	return declaration != nil &&
		declaration.Recv == nil &&
		declaration.Name != nil &&
		declaration.Name.Name == "init"
}
