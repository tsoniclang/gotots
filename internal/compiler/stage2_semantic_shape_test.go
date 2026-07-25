package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func requireDeclarationCardinalityAndMethodDescriptors(
	t *testing.T,
	pkg semantic.Package,
) {
	t.Helper()
	var (
		blank             semantic.DefinitionSemantics
		declaredMethod    semantic.DefinitionSemantics
		blankBodySemantic bool
	)
	for _, definition := range semanticDefinitions(pkg) {
		switch definition.Spec().Name {
		case "_":
			blank = definition
		case "Get":
			declaredMethod = definition
		}
	}
	for _, operation := range semanticOperations(pkg) {
		if operation.ID().Definition() == blank.Definition() {
			blankBodySemantic = true
			break
		}
	}
	if blank.Definition().IsZero() ||
		blank.Form() != semantic.DefinitionFormCallable ||
		blank.Spec().Signature.IsZero() ||
		len(blank.Spec().Declarations) != 0 ||
		!blankBodySemantic {
		t.Fatalf(
			"blank function semantics=%+v body=%t",
			blank.Spec(), blankBodySemantic,
		)
	}

	input := semanticPackageInput(pkg)
	recursive := namedTypeByDeclaration(
		t,
		input,
		declarationNamed(t, input, "Recursive").ID(),
	)
	iface := semanticTypeByID(
		t, pkg, recursive.Spec().Underlying,
	)
	if iface.Kind() != semantic.TypeInterface ||
		len(iface.Spec().Methods) != 1 {
		t.Fatalf(
			"recursive interface descriptor=%+v",
			iface.Spec(),
		)
	}
	methodSignature := semanticTypeByID(
		t, pkg, iface.Spec().Methods[0].Signature,
	).Spec().Signature
	if !methodSignature.Receiver.IsZero() ||
		len(methodSignature.ReceiverTypeParameters) != 0 {
		t.Fatalf(
			"recursive method descriptor retained receiver: %+v",
			methodSignature,
		)
	}
	if declaredMethod.Definition().IsZero() {
		t.Fatal("declared method Get is absent")
	}
	declaredSignature := semanticTypeByID(
		t, pkg, declaredMethod.Spec().Signature,
	).Spec().Signature
	if declaredSignature.Receiver.IsZero() {
		t.Fatalf(
			"declared method semantics lost receiver: %+v",
			declaredSignature,
		)
	}
}

func semanticTypeByID(
	t *testing.T,
	pkg semantic.Package,
	id identity.SemanticTypeID,
) semantic.Type {
	t.Helper()
	for _, record := range semanticTypes(pkg) {
		if record.ID() == id {
			return record
		}
	}
	t.Fatalf("semantic type %s is absent", id)
	return semantic.Type{}
}
