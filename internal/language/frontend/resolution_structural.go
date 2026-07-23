package frontend

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (builder *packageBuilder) structuralResolution(
	record *occurrenceInput,
	context occurrenceContext,
	variant catalog.Variant,
) (semantic.OccurrenceResolution, error) {
	disposition := structuralDisposition(record)
	var (
		declaration identity.SemanticDeclarationID
		typeID      identity.SemanticTypeID
	)
	if structuralCompileTime(record, context) {
		disposition = semantic.StructuralCompileTimeExpression
		if context.coverageObject != nil {
			var err error
			declaration, err = builder.objects.declarationID(
				context.coverageObject,
			)
			if err != nil {
				return semantic.OccurrenceResolution{}, err
			}
		} else if context.coverageType != nil {
			var err error
			typeID, err = builder.types.build(context.coverageType)
			if err != nil {
				return semantic.OccurrenceResolution{}, err
			}
		} else {
			return semantic.OccurrenceResolution{}, &Error{
				Package:    builder.input.id,
				Definition: record.owner,
				Occurrence: record.occurrence.ID(),
				Kind:       record.occurrence.Kind(),
				Reason:     "compile-time syntax has no exact declaration or type coverage",
			}
		}
	}
	evidence, err := semantic.NewStructuralEvidence(
		disposition, declaration, typeID,
	)
	if err != nil {
		return semantic.OccurrenceResolution{}, err
	}
	return semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record, variant, semantic.ResolutionStructuralOnly,
		).withStructural(evidence),
	)
}

func structuralDisposition(
	record *occurrenceInput,
) semantic.StructuralDisposition {
	switch record.occurrence.Role() {
	case catalog.RoleDocumentation,
		catalog.RoleTrailingDocumentation,
		catalog.RoleCommentText:
		return semantic.StructuralDocumentation
	case catalog.RolePackageName:
		return semantic.StructuralPackageClause
	case catalog.RoleImportPath:
		return semantic.StructuralImportPath
	}
	switch record.occurrence.Kind() {
	case catalog.KindComment, catalog.KindCommentGroup,
		catalog.KindDirective:
		return semantic.StructuralDocumentation
	case catalog.KindFile, catalog.KindFieldList:
		return semantic.StructuralContainer
	case catalog.KindGenDecl, catalog.KindImportSpec,
		catalog.KindValueSpec, catalog.KindField:
		return semantic.StructuralDeclarationEnvelope
	default:
		return semantic.StructuralContainer
	}
}

func structuralCompileTime(
	record *occurrenceInput,
	context occurrenceContext,
) bool {
	if record.domain != catalog.ResolutionDomainOwner &&
		record.domain != catalog.ResolutionDomainHeader {
		return false
	}
	if context.coverageObject == nil && context.coverageType == nil {
		return false
	}
	return catalog.AllowsCompileTimeStructural(
		record.occurrence.Kind(),
	)
}

func (builder *packageBuilder) definitionComponent(
	record *occurrenceInput,
	variant catalog.Variant,
	definition identity.DefinitionID,
	component semantic.DefinitionComponentKind,
) (semantic.OccurrenceResolution, semantic.OperationKind, error) {
	resolution, err := semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record, variant, semantic.ResolutionDefinitionComponent,
		).withComponent(definition, component),
	)
	return resolution, semantic.OperationInvalid, err
}

func boundaryComponent(
	definition identity.DefinitionID,
) semantic.DefinitionComponentKind {
	switch definition.Kind() {
	case identity.DefinitionPackageInitializer:
		return semantic.DefinitionComponentInitializer
	case identity.DefinitionBodylessDecl:
		return semantic.DefinitionComponentBodyless
	case identity.DefinitionImplicit:
		return semantic.DefinitionComponentImplicit
	default:
		return semantic.DefinitionComponentBoundary
	}
}

func (builder *packageBuilder) unsupportedResolution(
	record *occurrenceInput,
	variant catalog.Variant,
) (semantic.OccurrenceResolution, semantic.OperationKind, error) {
	if record.owner.IsZero() {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, &Error{
				Package:    builder.input.id,
				Occurrence: record.occurrence.ID(),
				Kind:       record.occurrence.Kind(),
				Reason:     "unsupported file-level construct has no definition owner",
			}
	}
	id, err := identity.NewUnsupportedID(
		record.owner, record.occurrence.ID(),
	)
	if err != nil {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, err
	}
	unsupported, err := semantic.NewUnsupported(
		id, semantic.UnsupportedExplicitContract,
		"catalog disposition "+record.occurrence.Kind().Disposition().String(),
		builder.input.authority,
	)
	if err != nil {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, err
	}
	builder.unsupported = append(builder.unsupported, unsupported)
	resolution, err := semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record, variant, semantic.ResolutionUnsupported,
		).withUnsupported(id),
	)
	return resolution, semantic.OperationInvalid, err
}

func (builder *packageBuilder) resolutionSpec(
	record *occurrenceInput,
	variant catalog.Variant,
	kind semantic.ResolutionKind,
) resolutionSpecBuilder {
	return resolutionSpecBuilder{spec: semantic.ResolutionSpec{
		Occurrence: record.occurrence.ID(),
		Owner:      record.owner,
		Syntax:     record.occurrence.Kind(),
		Role:       record.occurrence.Role(),
		Variant:    variant,
		Domain:     record.domain,
		Kind:       kind,
	}}
}

type resolutionSpecBuilder struct {
	spec semantic.ResolutionSpec
}

func (builder resolutionSpecBuilder) withStructural(
	value semantic.StructuralEvidence,
) semantic.ResolutionSpec {
	builder.spec.Structural = value
	return builder.spec
}

func (builder resolutionSpecBuilder) withComponent(
	definition identity.DefinitionID,
	component semantic.DefinitionComponentKind,
) semantic.ResolutionSpec {
	builder.spec.Definition = definition
	builder.spec.Component = component
	return builder.spec
}

func (builder resolutionSpecBuilder) withDeclaration(
	value identity.SemanticDeclarationID,
) semantic.ResolutionSpec {
	builder.spec.Declaration = value
	return builder.spec
}

func (builder resolutionSpecBuilder) withBinding(
	value identity.SemanticBindingID,
) semantic.ResolutionSpec {
	builder.spec.Binding = value
	return builder.spec
}

func (builder resolutionSpecBuilder) withType(
	value identity.SemanticTypeID,
) semantic.ResolutionSpec {
	builder.spec.Type = value
	return builder.spec
}

func (builder resolutionSpecBuilder) withUnsupported(
	value identity.UnsupportedID,
) semantic.ResolutionSpec {
	builder.spec.Unsupported = value
	return builder.spec
}

func (builder resolutionSpecBuilder) withOperation(
	value identity.OperationID,
) semantic.ResolutionSpec {
	builder.spec.Operation = value
	return builder.spec
}

func (builder *packageBuilder) admitResolution(
	resolution semantic.OccurrenceResolution,
) error {
	id := resolution.Occurrence()
	if _, duplicate := builder.resolutionByOccurrence[id]; duplicate {
		return fmt.Errorf("duplicate semantic resolution %s", id)
	}
	builder.resolutionByOccurrence[id] = resolution
	builder.resolutions = append(builder.resolutions, resolution)
	return nil
}
