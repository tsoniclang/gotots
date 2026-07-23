package semantic

import (
	"encoding/json"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func encodeProviderShard(pkg Package) ([]byte, providerShard, error) {
	shard := providerShard{
		Version:    ProviderArtifactVersion,
		Package:    pkg.ID().String(),
		Provenance: uint8(pkg.Provenance()),
	}
	for _, record := range pkg.Definitions() {
		shard.Definitions = append(
			shard.Definitions, encodeDefinition(record),
		)
	}
	for _, record := range pkg.Resolutions() {
		shard.Resolutions = append(
			shard.Resolutions, encodeResolution(record),
		)
	}
	for _, record := range pkg.Declarations() {
		shard.Declarations = append(
			shard.Declarations, encodeDeclaration(record),
		)
	}
	for _, record := range pkg.Bindings() {
		shard.Bindings = append(
			shard.Bindings, encodeBinding(record),
		)
	}
	for _, record := range pkg.Types() {
		shard.Types = append(
			shard.Types, encodeType(record),
		)
	}
	for _, record := range pkg.Operations() {
		shard.Operations = append(
			shard.Operations, encodeOperation(record),
		)
	}
	for _, record := range pkg.Unsupported() {
		shard.Unsupported = append(
			shard.Unsupported, encodeUnsupported(record),
		)
	}
	encoded, err := json.Marshal(shard)
	if err != nil {
		return nil, providerShard{}, fmt.Errorf(
			"semantic provider shard encoding failed: %w", err,
		)
	}
	return encoded, shard, nil
}

func encodeDefinition(record DefinitionSemantics) wireDefinition {
	spec := record.Spec()
	return wireDefinition{
		Definition:         spec.Definition.String(),
		Package:            spec.Package.String(),
		Form:               uint8(spec.Form),
		Name:               spec.Name,
		Declarations:       declarationStrings(spec.Declarations),
		Signature:          spec.Signature.String(),
		Receiver:           spec.Receiver.String(),
		Bindings:           bindingStrings(spec.Bindings),
		InitializerEntries: occurrenceStrings(spec.InitializerEntries),
		Implicit:           uint8(spec.Implicit),
	}
}

func encodeResolution(record OccurrenceResolution) wireResolution {
	structural := record.Structural()
	return wireResolution{
		Occurrence: record.Occurrence().String(),
		Owner:      record.Owner().String(),
		Syntax:     uint16(record.Syntax()),
		Role:       uint16(record.Role()),
		Variant:    uint16(record.Variant()),
		Domain:     uint8(record.Domain()),
		Kind:       uint8(record.Kind()),
		Structural: wireStructural{
			Disposition: uint8(structural.Disposition()),
			Declaration: structural.Declaration().String(),
			Type:        structural.Type().String(),
		},
		Component:   uint8(record.Component()),
		Definition:  record.Definition().String(),
		Declaration: record.Declaration().String(),
		Binding:     record.Binding().String(),
		Type:        record.Type().String(),
		Operation:   record.Operation().String(),
		Unsupported: record.Unsupported().String(),
	}
}

func encodeDeclaration(record Declaration) wireDeclaration {
	return wireDeclaration{
		ID:       record.ID().String(),
		Package:  record.Package().String(),
		Class:    uint8(record.Class()),
		Name:     record.Name(),
		Type:     record.Type().String(),
		Exported: record.Exported(),
		Constant: encodeConstant(record.Constant()),
	}
}

func encodeBinding(record Binding) wireBinding {
	return wireBinding{
		ID:         record.ID().String(),
		Package:    record.Package().String(),
		Definition: record.Definition().String(),
		Role:       uint8(record.Role()),
		Name:       record.Name(),
		Type:       record.Type().String(),
		Source:     record.Source().String(),
		CapturedBy: definitionStrings(record.CapturedBy()),
	}
}

func encodeUnsupported(record Unsupported) wireUnsupported {
	return wireUnsupported{
		ID:       record.ID().String(),
		Reason:   uint8(record.Reason()),
		Evidence: record.Evidence(),
	}
}

func encodeType(record Type) wireType {
	spec := record.Spec()
	out := wireType{
		ID:                   record.ID().String(),
		Kind:                 uint8(spec.Kind),
		Basic:                uint8(spec.Basic),
		Declaration:          spec.Declaration.String(),
		ParameterDeclaration: spec.Parameter.Declaration().String(),
		ParameterDefinition:  spec.Parameter.Definition().String(),
		ParameterRole:        uint8(spec.Parameter.Role()),
		ParameterOrdinal:     spec.Parameter.Ordinal(),
		Arguments:            typeStrings(spec.Arguments),
		Underlying:           spec.Underlying.String(),
		Target:               spec.Target.String(),
		Constraint:           spec.Constraint.String(),
		Element:              spec.Element.String(),
		Key:                  spec.Key.String(),
		Length:               spec.Length,
		Direction:            uint8(spec.Direction),
		Signature: wireSignature{
			Receiver: spec.Signature.Receiver.String(),
			ReceiverTypeParameters: typeStrings(
				spec.Signature.ReceiverTypeParameters,
			),
			TypeParameters: typeStrings(
				spec.Signature.TypeParameters,
			),
			Parameters: typeStrings(spec.Signature.Parameters),
			Results:    typeStrings(spec.Signature.Results),
			Variadic:   spec.Signature.Variadic,
		},
		Embeddeds:  typeStrings(spec.Embeddeds),
		TypeSet:    uint8(spec.TypeSet),
		Comparable: spec.Comparable,
		Elements:   typeStrings(spec.Elements),
	}
	for _, field := range spec.Fields {
		out.Fields = append(out.Fields, wireTypeField{
			Name: field.Name, Package: field.Package.String(),
			Type: field.Type.String(), Embedded: field.Embedded,
			Tag: field.Tag, Ordinal: field.Ordinal,
		})
	}
	for _, method := range spec.Methods {
		out.Methods = append(out.Methods, wireTypeMethod{
			Name: method.Name, Package: method.Package.String(),
			Signature: method.Signature.String(),
			Ordinal:   method.Ordinal,
		})
	}
	for _, term := range spec.Terms {
		out.Terms = append(out.Terms, wireTypeTerm{
			Tilde: term.Tilde, Type: term.Type.String(),
		})
	}
	return out
}

func encodeOperation(record Operation) wireOperation {
	spec := record.Spec()
	out := wireOperation{
		ID:            spec.ID.String(),
		Kind:          uint16(spec.Kind),
		Syntax:        uint16(spec.Syntax),
		Variant:       uint16(spec.Variant),
		Role:          uint16(spec.Role),
		Token:         uint16(spec.Token),
		Mode:          uint8(spec.Mode),
		Arity:         uint8(spec.Arity),
		Place:         uint8(spec.Place),
		ResultType:    spec.ResultType.String(),
		ExpectedType:  spec.ExpectedType.String(),
		Addressable:   spec.Addressable,
		Assignable:    spec.Assignable,
		HasOk:         spec.HasOk,
		Constant:      encodeConstant(spec.Constant),
		Object:        encodeObjectReference(spec.Object),
		Operands:      occurrenceStrings(spec.Operands),
		Definitions:   definitionStrings(spec.Definitions),
		ControlTarget: spec.ControlTarget.String(),
		Label:         spec.Label.String(),
	}
	if !spec.Selection.IsZero() {
		out.Selection = wireSelection{
			Kind:     uint8(spec.Selection.Kind()),
			Receiver: spec.Selection.Receiver().String(),
			Object:   spec.Selection.Object().String(),
			Index:    spec.Selection.Index(),
			Indirect: spec.Selection.Indirect(),
		}
	}
	if !spec.Instance.IsZero() {
		out.Instance = wireInstance{
			Target:    encodeObjectReference(spec.Instance.Target()),
			Types:     typeStrings(spec.Instance.Types()),
			Signature: spec.Instance.Signature().String(),
		}
	}
	for _, implicit := range spec.Implicit {
		out.Implicit = append(out.Implicit, wireImplicit{
			Kind: uint8(implicit.Kind()), Site: implicit.Site().String(),
			Ordinal: implicit.Ordinal(),
			Source:  implicit.Source().String(),
			Target:  implicit.Target().String(),
		})
	}
	return out
}

func encodeObjectReference(
	reference ObjectReference,
) wireObjectReference {
	return wireObjectReference{
		Kind:        uint8(reference.Kind()),
		Declaration: reference.Declaration().String(),
		Binding:     reference.Binding().String(),
	}
}

func encodeConstant(constant Constant) wireConstant {
	if constant.IsZero() {
		return wireConstant{}
	}
	return wireConstant{
		Kind: uint8(constant.Kind()), Exact: constant.Exact(),
	}
}

func definitionStrings(
	values []identity.DefinitionID,
) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func occurrenceStrings(
	values []identity.OccurrenceID,
) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func declarationStrings(
	values []identity.SemanticDeclarationID,
) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func bindingStrings(
	values []identity.SemanticBindingID,
) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func typeStrings(
	values []identity.SemanticTypeID,
) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}
