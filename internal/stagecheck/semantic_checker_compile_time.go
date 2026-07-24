package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeContext(
	reference semanticOccurrenceRef,
) bool {
	return verifier.independentCompileTimeAnchor(
		reference,
	).valid()
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeAnchor(
	reference semanticOccurrenceRef,
) semanticOccurrenceRef {
	occurrence := verifier.expected.occurrenceRecord(reference)
	if occurrence.ID().IsZero() {
		return 0
	}
	if verifier.compileTimeResolved[reference] {
		return verifier.compileTimeAnchor[reference]
	}
	var anchor semanticOccurrenceRef
	if occurrence.Role() == catalog.RoleArrayLength ||
		(occurrence.Role() == catalog.RoleInitializerValue &&
			verifier.independentConstInitializer(reference)) {
		anchor = reference
	}
	if !anchor.valid() {
		parentReference := verifier.expected.parentReference(reference)
		if parentReference.valid() {
			anchor = verifier.independentCompileTimeAnchor(
				parentReference,
			)
		}
	}
	verifier.compileTimeAnchor[reference] = anchor
	verifier.compileTimeResolved[reference] = true
	return anchor
}

func (
	verifier *checkerSemanticVerifier,
) independentCompileTimeCoverage(
	reference semanticOccurrenceRef,
) (types.Object, types.Type) {
	anchorReference := verifier.independentCompileTimeAnchor(reference)
	if !anchorReference.valid() {
		return nil, nil
	}
	anchor := verifier.expected.occurrenceRecord(anchorReference)
	parent := verifier.expected.occurrenceRecord(
		verifier.expected.parentReference(anchorReference),
	)
	if parent.ID().IsZero() {
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
	reference semanticOccurrenceRef,
) bool {
	valueSpecReference := verifier.expected.parentReference(reference)
	valueSpecOccurrence := verifier.expected.occurrenceRecord(
		valueSpecReference,
	)
	if valueSpecOccurrence.ID().IsZero() {
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
	genDeclOccurrence := verifier.expected.occurrenceRecord(
		verifier.expected.parentReference(valueSpecReference),
	)
	if genDeclOccurrence.ID().IsZero() {
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
