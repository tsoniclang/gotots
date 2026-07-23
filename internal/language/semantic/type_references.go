package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func referencedTypeIDs(record Type) []identity.SemanticTypeID {
	spec := record.Spec()
	var references []identity.SemanticTypeID
	references = append(references, spec.Arguments...)
	references = append(
		references,
		spec.Underlying,
		spec.Target,
		spec.Constraint,
		spec.Element,
		spec.Key,
		spec.Signature.Receiver,
	)
	references = append(
		references, spec.Signature.ReceiverTypeParameters...,
	)
	references = append(
		references, spec.Signature.TypeParameters...,
	)
	references = append(references, spec.Signature.Parameters...)
	references = append(references, spec.Signature.Results...)
	for _, field := range spec.Fields {
		references = append(references, field.Type)
	}
	for _, method := range spec.Methods {
		references = append(references, method.Signature)
	}
	references = append(references, spec.Embeddeds...)
	for _, term := range spec.Terms {
		references = append(references, term.Type)
	}
	references = append(references, spec.Elements...)
	return references
}

func packageInputTypeRoots(
	input PackageInput,
) []identity.SemanticTypeID {
	var roots []identity.SemanticTypeID
	for _, record := range input.Definitions {
		spec := record.Spec()
		roots = append(roots, spec.Signature)
	}
	for _, record := range input.Resolutions {
		roots = append(roots, record.Type())
		if record.Kind() == ResolutionStructuralOnly {
			roots = append(roots, record.Structural().Type())
		}
	}
	for _, record := range input.Declarations {
		roots = append(roots, record.Type())
	}
	for _, record := range input.Bindings {
		roots = append(roots, record.Type())
	}
	for _, record := range input.Operations {
		spec := record.Spec()
		roots = append(
			roots,
			spec.ResultType,
			spec.ExpectedType,
			spec.Selection.Receiver(),
			spec.Instance.Signature(),
		)
		roots = append(roots, spec.Instance.Types()...)
		for _, implicit := range spec.Implicit {
			roots = append(
				roots, implicit.Source(), implicit.Target(),
			)
		}
	}
	return roots
}
