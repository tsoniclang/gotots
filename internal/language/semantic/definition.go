package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type DefinitionSemanticsSpec struct {
	Definition identity.DefinitionID
	Package    identity.PackageID
	Form       DefinitionForm
	Authority  Authority
	Name       string

	Declarations       []identity.SemanticDeclarationID
	Signature          identity.SemanticTypeID
	Receiver           identity.SemanticBindingID
	Bindings           []identity.SemanticBindingID
	InitializerEntries []identity.OccurrenceID
	Implicit           identity.ImplicitDefinitionOp
}

type DefinitionSemantics struct {
	spec DefinitionSemanticsSpec
}

func NewDefinitionSemantics(
	spec DefinitionSemanticsSpec,
) (DefinitionSemantics, error) {
	spec = cloneDefinitionSemanticsSpec(spec)
	if spec.Definition.IsZero() ||
		spec.Package.IsZero() ||
		!spec.Form.Valid() ||
		!spec.Authority.Valid() ||
		spec.Name == "" {
		return DefinitionSemantics{}, fmt.Errorf(
			"definition semantics requires identity, package, form, authority, and name",
		)
	}
	seenDeclarations := map[identity.SemanticDeclarationID]bool{}
	for _, declaration := range spec.Declarations {
		if declaration.IsZero() || seenDeclarations[declaration] {
			return DefinitionSemantics{}, fmt.Errorf(
				"definition %s has invalid declaration set",
				spec.Definition,
			)
		}
		seenDeclarations[declaration] = true
	}
	seenBindings := map[identity.SemanticBindingID]bool{}
	for _, binding := range spec.Bindings {
		if binding.IsZero() || seenBindings[binding] {
			return DefinitionSemantics{}, fmt.Errorf(
				"definition %s has invalid binding set",
				spec.Definition,
			)
		}
		seenBindings[binding] = true
	}
	seenInitializers := map[identity.OccurrenceID]bool{}
	for _, initializer := range spec.InitializerEntries {
		if initializer.IsZero() ||
			(!spec.Definition.File().IsZero() &&
				initializer.Span().File() !=
					spec.Definition.File()) ||
			seenInitializers[initializer] {
			return DefinitionSemantics{}, fmt.Errorf(
				"definition %s has invalid initializer entries",
				spec.Definition,
			)
		}
		seenInitializers[initializer] = true
	}
	if err := validateDefinitionForm(spec); err != nil {
		return DefinitionSemantics{}, err
	}
	return DefinitionSemantics{spec: spec}, nil
}

func validateDefinitionForm(
	spec DefinitionSemanticsSpec,
) error {
	hasCallable := !spec.Signature.IsZero()
	hasInitializers := len(spec.InitializerEntries) != 0
	hasImplicit := spec.Implicit.Valid()
	if err := validateDefinitionIdentityForm(spec); err != nil {
		return err
	}
	switch spec.Form {
	case DefinitionFormCallable:
		if !hasCallable ||
			hasInitializers ||
			hasImplicit {
			return fmt.Errorf(
				"callable definition %s lacks callable semantics",
				spec.Definition,
			)
		}
	case DefinitionFormInitializer:
		if hasCallable ||
			hasImplicit ||
			!hasInitializers {
			return fmt.Errorf(
				"initializer definition %s has incompatible semantics",
				spec.Definition,
			)
		}
	case DefinitionFormBodyless:
		builtinContract := len(spec.Declarations) == 1 &&
			spec.Declarations[0].Class() ==
				identity.SemanticObjectBuiltin
		if hasCallable == builtinContract ||
			hasInitializers ||
			hasImplicit ||
			len(spec.Declarations) != 1 {
			return fmt.Errorf(
				"bodyless definition %s has incompatible semantics",
				spec.Definition,
			)
		}
	case DefinitionFormImplicit:
		if !hasImplicit ||
			hasCallable ||
			hasInitializers ||
			len(spec.Declarations) != 0 {
			return fmt.Errorf(
				"implicit definition %s lacks implicit meaning",
				spec.Definition,
			)
		}
	case DefinitionFormSynthetic:
		if hasInitializers ||
			hasImplicit ||
			len(spec.Declarations) != 1 {
			return fmt.Errorf(
				"synthetic definition %s has incompatible semantics",
				spec.Definition,
			)
		}
		adapter := spec.Definition.SyntheticRole() ==
			identity.SyntheticDefinitionAdapter
		if adapter != hasCallable {
			return fmt.Errorf(
				"synthetic definition %s signature disagrees with role",
				spec.Definition,
			)
		}
	}
	return nil
}

func validateDefinitionIdentityForm(
	spec DefinitionSemanticsSpec,
) error {
	var wantForm DefinitionForm
	wantDeclarations := -1
	switch spec.Definition.Kind() {
	case identity.DefinitionFuncDecl:
		wantForm = DefinitionFormCallable
		if spec.Name == "_" || spec.Name == "init" {
			wantDeclarations = 0
		} else {
			wantDeclarations = 1
		}
	case identity.DefinitionFuncLit:
		wantForm = DefinitionFormCallable
		wantDeclarations = 0
	case identity.DefinitionPackageInitializer:
		wantForm = DefinitionFormInitializer
	case identity.DefinitionBodylessDecl:
		wantForm = DefinitionFormBodyless
		wantDeclarations = 1
	case identity.DefinitionImplicit:
		switch {
		case spec.Definition.ImplicitOp().Valid():
			wantForm = DefinitionFormImplicit
			wantDeclarations = 0
		case spec.Definition.SyntheticRole().Valid():
			wantForm = DefinitionFormSynthetic
			wantDeclarations = 1
		}
	}
	if wantForm == DefinitionFormInvalid ||
		spec.Form != wantForm ||
		(wantDeclarations >= 0 &&
			len(spec.Declarations) != wantDeclarations) {
		return fmt.Errorf(
			"definition %s form/declaration cardinality disagrees with identity",
			spec.Definition,
		)
	}
	return nil
}

func cloneDefinitionSemanticsSpec(
	spec DefinitionSemanticsSpec,
) DefinitionSemanticsSpec {
	spec.Declarations = append(
		[]identity.SemanticDeclarationID(nil),
		spec.Declarations...,
	)
	spec.Bindings = append(
		[]identity.SemanticBindingID(nil), spec.Bindings...,
	)
	spec.InitializerEntries = append(
		[]identity.OccurrenceID(nil), spec.InitializerEntries...,
	)
	return spec
}

func (record DefinitionSemantics) Definition() identity.DefinitionID {
	return record.spec.Definition
}
func (record DefinitionSemantics) Package() identity.PackageID {
	return record.spec.Package
}
func (record DefinitionSemantics) Form() DefinitionForm {
	return record.spec.Form
}
func (record DefinitionSemantics) Authority() Authority {
	return record.spec.Authority
}
func (record DefinitionSemantics) Spec() DefinitionSemanticsSpec {
	return cloneDefinitionSemanticsSpec(record.spec)
}

type StructuralEvidence struct {
	disposition StructuralDisposition
	declaration identity.SemanticDeclarationID
	typeID      identity.SemanticTypeID
}

func NewStructuralEvidence(
	disposition StructuralDisposition,
	declaration identity.SemanticDeclarationID,
	typeID identity.SemanticTypeID,
) (StructuralEvidence, error) {
	if !disposition.Valid() {
		return StructuralEvidence{}, fmt.Errorf(
			"structural evidence requires a closed disposition",
		)
	}
	hasDeclaration := !declaration.IsZero()
	hasType := !typeID.IsZero()
	if disposition == StructuralCompileTimeExpression {
		if hasDeclaration == hasType {
			return StructuralEvidence{}, fmt.Errorf(
				"compile-time expression requires exactly one coverage target",
			)
		}
	} else if hasDeclaration || hasType {
		return StructuralEvidence{}, fmt.Errorf(
			"non-expression structural evidence carries coverage",
		)
	}
	return StructuralEvidence{
		disposition: disposition,
		declaration: declaration,
		typeID:      typeID,
	}, nil
}

func (record StructuralEvidence) Valid() bool {
	if !record.disposition.Valid() {
		return false
	}
	hasDeclaration := !record.declaration.IsZero()
	hasType := !record.typeID.IsZero()
	if record.disposition == StructuralCompileTimeExpression {
		return hasDeclaration != hasType
	}
	return !hasDeclaration && !hasType
}

func (record StructuralEvidence) Disposition() StructuralDisposition {
	return record.disposition
}
func (record StructuralEvidence) Declaration() identity.SemanticDeclarationID {
	return record.declaration
}
func (record StructuralEvidence) Type() identity.SemanticTypeID {
	return record.typeID
}

type OccurrenceResolution struct {
	occurrence  identity.OccurrenceID
	owner       identity.DefinitionID
	syntax      catalog.Kind
	role        catalog.Role
	variant     catalog.Variant
	domain      catalog.ResolutionDomain
	kind        ResolutionKind
	structural  StructuralEvidence
	component   DefinitionComponentKind
	definition  identity.DefinitionID
	declaration identity.SemanticDeclarationID
	binding     identity.SemanticBindingID
	typeID      identity.SemanticTypeID
	operation   identity.OperationID
	unsupported identity.UnsupportedID
}

type ResolutionSpec struct {
	Occurrence  identity.OccurrenceID
	Owner       identity.DefinitionID
	Syntax      catalog.Kind
	Role        catalog.Role
	Variant     catalog.Variant
	Domain      catalog.ResolutionDomain
	Kind        ResolutionKind
	Structural  StructuralEvidence
	Component   DefinitionComponentKind
	Definition  identity.DefinitionID
	Declaration identity.SemanticDeclarationID
	Binding     identity.SemanticBindingID
	Type        identity.SemanticTypeID
	Operation   identity.OperationID
	Unsupported identity.UnsupportedID
}

func NewOccurrenceResolution(
	spec ResolutionSpec,
) (OccurrenceResolution, error) {
	if spec.Occurrence.IsZero() ||
		!spec.Syntax.Valid() ||
		spec.Occurrence.KindID() != uint16(spec.Syntax) ||
		(spec.Role != catalog.RoleInvalid && !spec.Role.Valid()) ||
		!spec.Variant.Valid() ||
		!spec.Domain.Valid() ||
		!spec.Kind.Valid() {
		return OccurrenceResolution{}, fmt.Errorf(
			"occurrence resolution requires identity, syntax, variant, and kind",
		)
	}
	set := 0
	if spec.Structural.Valid() {
		set++
	}
	if spec.Component.Valid() {
		set++
	}
	if !spec.Declaration.IsZero() {
		set++
	}
	if !spec.Binding.IsZero() {
		set++
	}
	if !spec.Type.IsZero() {
		set++
	}
	if !spec.Operation.IsZero() {
		set++
	}
	if !spec.Unsupported.IsZero() {
		set++
	}
	if set != 1 {
		return OccurrenceResolution{}, fmt.Errorf(
			"occurrence %s resolution must select exactly one payload",
			spec.Occurrence,
		)
	}
	valid := false
	switch spec.Kind {
	case ResolutionStructuralOnly:
		valid = spec.Structural.Valid()
	case ResolutionDefinitionComponent:
		valid = spec.Component.Valid() &&
			!spec.Definition.IsZero()
	case ResolutionDeclaration:
		valid = !spec.Declaration.IsZero()
	case ResolutionBinding:
		valid = !spec.Binding.IsZero()
	case ResolutionType:
		valid = !spec.Type.IsZero()
	case ResolutionOperation:
		valid = !spec.Operation.IsZero() &&
			spec.Operation.Occurrence() == spec.Occurrence &&
			spec.Operation.Definition() == spec.Owner
	case ResolutionUnsupported:
		valid = !spec.Unsupported.IsZero() &&
			spec.Unsupported.Occurrence() == spec.Occurrence &&
			spec.Unsupported.Definition() == spec.Owner
	}
	if !valid {
		return OccurrenceResolution{}, fmt.Errorf(
			"occurrence %s has mismatched resolution payload",
			spec.Occurrence,
		)
	}
	class := resolutionCatalogClass(spec.Kind)
	allowed := catalog.AllowsResolution(
		spec.Syntax, spec.Role, spec.Variant, spec.Domain, class,
	)
	if spec.Kind == ResolutionStructuralOnly &&
		spec.Structural.Disposition() ==
			StructuralIntrinsicContract {
		allowed = catalog.AllowsIntrinsicContract(
			spec.Syntax, spec.Role, spec.Domain,
		)
	}
	if !allowed {
		return OccurrenceResolution{}, fmt.Errorf(
			"catalog rejects %s resolution for %s role=%s variant=%s domain=%s at %s",
			spec.Kind, spec.Syntax, spec.Role, spec.Variant,
			spec.Domain, spec.Occurrence,
		)
	}
	return OccurrenceResolution{
		occurrence: spec.Occurrence, owner: spec.Owner,
		syntax: spec.Syntax, role: spec.Role, variant: spec.Variant,
		domain: spec.Domain,
		kind:   spec.Kind, structural: spec.Structural,
		component: spec.Component, definition: spec.Definition,
		declaration: spec.Declaration, binding: spec.Binding,
		typeID: spec.Type, operation: spec.Operation,
		unsupported: spec.Unsupported,
	}, nil
}

func (record OccurrenceResolution) Occurrence() identity.OccurrenceID {
	return record.occurrence
}
func (record OccurrenceResolution) Owner() identity.DefinitionID {
	return record.owner
}
func (record OccurrenceResolution) Syntax() catalog.Kind {
	return record.syntax
}
func (record OccurrenceResolution) Role() catalog.Role {
	return record.role
}
func (record OccurrenceResolution) Variant() catalog.Variant {
	return record.variant
}
func (record OccurrenceResolution) Domain() catalog.ResolutionDomain {
	return record.domain
}
func (record OccurrenceResolution) Kind() ResolutionKind {
	return record.kind
}
func (record OccurrenceResolution) Structural() StructuralEvidence {
	return record.structural
}
func (record OccurrenceResolution) Component() DefinitionComponentKind {
	return record.component
}
func (record OccurrenceResolution) Definition() identity.DefinitionID {
	return record.definition
}
func (record OccurrenceResolution) Declaration() identity.SemanticDeclarationID {
	return record.declaration
}
func (record OccurrenceResolution) Binding() identity.SemanticBindingID {
	return record.binding
}
func (record OccurrenceResolution) Type() identity.SemanticTypeID {
	return record.typeID
}
func (record OccurrenceResolution) Operation() identity.OperationID {
	return record.operation
}
func (record OccurrenceResolution) Unsupported() identity.UnsupportedID {
	return record.unsupported
}

func resolutionCatalogClass(
	kind ResolutionKind,
) catalog.ResolutionClass {
	switch kind {
	case ResolutionStructuralOnly:
		return catalog.ResolutionClassStructural
	case ResolutionDefinitionComponent:
		return catalog.ResolutionClassDefinitionComponent
	case ResolutionDeclaration:
		return catalog.ResolutionClassDeclaration
	case ResolutionBinding:
		return catalog.ResolutionClassBinding
	case ResolutionType:
		return catalog.ResolutionClassType
	case ResolutionOperation:
		return catalog.ResolutionClassOperation
	case ResolutionUnsupported:
		return catalog.ResolutionClassUnsupported
	default:
		return catalog.ResolutionClassInvalid
	}
}

type Unsupported struct {
	id        identity.UnsupportedID
	reason    UnsupportedReason
	evidence  string
	authority Authority
}

func NewUnsupported(
	id identity.UnsupportedID,
	reason UnsupportedReason,
	evidence string,
	authority Authority,
) (Unsupported, error) {
	if id.IsZero() ||
		!reason.Valid() ||
		evidence == "" ||
		!authority.Valid() {
		return Unsupported{}, fmt.Errorf(
			"unsupported record requires identity, reason, evidence, and authority",
		)
	}
	return Unsupported{
		id: id, reason: reason,
		evidence: evidence, authority: authority,
	}, nil
}

func (record Unsupported) ID() identity.UnsupportedID {
	return record.id
}
func (record Unsupported) Reason() UnsupportedReason {
	return record.reason
}
func (record Unsupported) Evidence() string {
	return record.evidence
}
func (record Unsupported) Authority() Authority {
	return record.authority
}
