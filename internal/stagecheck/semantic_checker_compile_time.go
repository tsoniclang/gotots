package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeContext(
	occurrence structure.OccurrenceRef,
) bool {
	return verifier.independentCompileTimeAnchor(
		occurrence,
	).valid()
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeAnchor(
	occurrence structure.OccurrenceRef,
) semanticOccurrenceRef {
	reference := verifier.expected.occurrences.reference(
		occurrence.ID(),
	)
	if verifier.compileTimeResolved[reference] {
		return verifier.compileTimeAnchor[reference]
	}
	var anchor semanticOccurrenceRef
	if occurrence.Role() == catalog.RoleArrayLength ||
		(occurrence.Role() == catalog.RoleInitializerValue &&
			verifier.independentConstInitializer(occurrence)) {
		anchor = reference
	}
	if !anchor.valid() && !occurrence.Parent().IsZero() {
		if parent, present :=
			verifier.expected.occurrences.get(occurrence.Parent()); present {
			anchor =
				verifier.independentCompileTimeAnchor(parent.OccurrenceRef)
		}
	}
	verifier.compileTimeAnchor[reference] = anchor
	verifier.compileTimeResolved[reference] = true
	return anchor
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeCoverage(
	occurrence structure.OccurrenceRef,
) (types.Object, types.Type) {
	anchorReference := verifier.independentCompileTimeAnchor(occurrence)
	if !anchorReference.valid() {
		return nil, nil
	}
	anchor := verifier.expected.occurrenceRecord(anchorReference)
	parent, present := verifier.expected.occurrences.get(anchor.Parent())
	if !present {
		return nil, nil
	}
	node, present := verifier.index.OccurrenceNode(parent.ID())
	if !present {
		return nil, nil
	}
	switch anchor.Role() {
	case catalog.RoleArrayLength:
		array, ok := node.(*ast.ArrayType)
		if !ok {
			return nil, nil
		}
		value, present := verifier.view.TypeOf(array)
		if !present {
			return nil, nil
		}
		return nil, value.Type
	case catalog.RoleInitializerValue:
		valueSpec, ok := node.(*ast.ValueSpec)
		if !ok || anchor.Ordinal() >= len(valueSpec.Names) {
			return nil, nil
		}
		object, _ := verifier.view.DefOf(
			valueSpec.Names[anchor.Ordinal()],
		)
		return object, nil
	default:
		return nil, nil
	}
}

func (
	verifier *checkerSemanticVerifier,
) independentConstInitializer(
	initializer structure.OccurrenceRef,
) bool {
	valueSpecOccurrence, present :=
		verifier.expected.occurrences.get(initializer.Parent())
	if !present {
		return false
	}
	valueSpecNode, present := verifier.index.OccurrenceNode(
		valueSpecOccurrence.ID(),
	)
	if !present {
		return false
	}
	if _, ok := valueSpecNode.(*ast.ValueSpec); !ok {
		return false
	}
	genDeclOccurrence, present :=
		verifier.expected.occurrences.get(valueSpecOccurrence.Parent())
	if !present {
		return false
	}
	genDeclNode, present := verifier.index.OccurrenceNode(
		genDeclOccurrence.ID(),
	)
	if !present {
		return false
	}
	genDecl, ok := genDeclNode.(*ast.GenDecl)
	return ok && genDecl.Tok == token.CONST
}
