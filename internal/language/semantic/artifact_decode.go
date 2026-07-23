package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func decodeProviderShardWithWire(
	encoded []byte,
	authority Authority,
) (Package, providerShard, error) {
	var shard providerShard
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&shard); err != nil {
		return Package{}, providerShard{}, fmt.Errorf(
			"semantic provider shard decode failed: %w", err,
		)
	}
	if err := requireJSONEnd(decoder); err != nil {
		return Package{}, providerShard{}, err
	}
	if shard.Version != ProviderArtifactVersion {
		return Package{}, providerShard{}, fmt.Errorf(
			"semantic provider shard version %d is unsupported",
			shard.Version,
		)
	}
	pkg, err := identity.ParsePackageID(shard.Package)
	if err != nil {
		return Package{}, providerShard{}, err
	}
	input := PackageInput{
		ID: pkg, Provenance: PackageProvenance(shard.Provenance),
	}
	for _, encoded := range shard.Definitions {
		record, err := decodeDefinition(encoded, authority)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Definitions = append(input.Definitions, record)
	}
	for _, encoded := range shard.Resolutions {
		record, err := decodeResolution(encoded)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Resolutions = append(input.Resolutions, record)
	}
	for _, encoded := range shard.Declarations {
		record, err := decodeDeclaration(encoded, authority)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Declarations = append(input.Declarations, record)
	}
	for _, encoded := range shard.Bindings {
		record, err := decodeBinding(encoded, authority)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Bindings = append(input.Bindings, record)
	}
	for _, encoded := range shard.Types {
		record, err := decodeType(encoded)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Types = append(input.Types, record)
		witness, err := NewTypeWitness(pkg, record.ID(), authority)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.TypeWitnesses = append(input.TypeWitnesses, witness)
	}
	for _, encoded := range shard.Operations {
		record, err := decodeOperation(encoded)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Operations = append(input.Operations, record)
	}
	for _, encoded := range shard.Unsupported {
		record, err := decodeUnsupported(encoded, authority)
		if err != nil {
			return Package{}, providerShard{}, err
		}
		input.Unsupported = append(input.Unsupported, record)
	}
	decoded, err := NewPackage(input)
	return decoded, shard, err
}

func decodeDefinition(
	encoded wireDefinition,
	authority Authority,
) (DefinitionSemantics, error) {
	definition, err := identity.ParseDefinitionID(encoded.Definition)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	pkg, err := identity.ParsePackageID(encoded.Package)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	declarations, err := parseDeclarations(encoded.Declarations)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	signature, err := parseOptionalType(encoded.Signature)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	receiver, err := parseOptionalBinding(encoded.Receiver)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	bindings, err := parseBindings(encoded.Bindings)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	initializers, err := parseOccurrences(
		encoded.InitializerEntries,
	)
	if err != nil {
		return DefinitionSemantics{}, err
	}
	return NewDefinitionSemantics(DefinitionSemanticsSpec{
		Definition:         definition,
		Package:            pkg,
		Form:               DefinitionForm(encoded.Form),
		Authority:          authority,
		Name:               encoded.Name,
		Declarations:       declarations,
		Signature:          signature,
		Receiver:           receiver,
		Bindings:           bindings,
		InitializerEntries: initializers,
		Implicit:           identity.ImplicitDefinitionOp(encoded.Implicit),
	})
}

func decodeResolution(
	encoded wireResolution,
) (OccurrenceResolution, error) {
	occurrence, err := identity.ParseOccurrenceID(encoded.Occurrence)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	owner, err := parseOptionalDefinition(encoded.Owner)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	structural := StructuralEvidence{}
	if encoded.Structural.Disposition != 0 {
		declaration, parseErr := parseOptionalDeclaration(
			encoded.Structural.Declaration,
		)
		if parseErr != nil {
			return OccurrenceResolution{}, parseErr
		}
		typeID, parseErr := parseOptionalType(
			encoded.Structural.Type,
		)
		if parseErr != nil {
			return OccurrenceResolution{}, parseErr
		}
		structural, err = NewStructuralEvidence(
			StructuralDisposition(encoded.Structural.Disposition),
			declaration,
			typeID,
		)
		if err != nil {
			return OccurrenceResolution{}, err
		}
	}
	definition, err := parseOptionalDefinition(encoded.Definition)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	declaration, err := parseOptionalDeclaration(encoded.Declaration)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	binding, err := parseOptionalBinding(encoded.Binding)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	typeID, err := parseOptionalType(encoded.Type)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	operation, err := parseOptionalOperation(encoded.Operation)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	unsupported, err := parseOptionalUnsupported(
		encoded.Unsupported,
	)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	return NewOccurrenceResolution(ResolutionSpec{
		Occurrence:  occurrence,
		Owner:       owner,
		Syntax:      catalog.Kind(encoded.Syntax),
		Role:        catalog.Role(encoded.Role),
		Variant:     catalog.Variant(encoded.Variant),
		Domain:      catalog.ResolutionDomain(encoded.Domain),
		Kind:        ResolutionKind(encoded.Kind),
		Structural:  structural,
		Component:   DefinitionComponentKind(encoded.Component),
		Definition:  definition,
		Declaration: declaration,
		Binding:     binding,
		Type:        typeID,
		Operation:   operation,
		Unsupported: unsupported,
	})
}

func decodeDeclaration(
	encoded wireDeclaration,
	authority Authority,
) (Declaration, error) {
	id, err := identity.ParseSemanticDeclarationID(encoded.ID)
	if err != nil {
		return Declaration{}, err
	}
	pkg, err := identity.ParsePackageID(encoded.Package)
	if err != nil {
		return Declaration{}, err
	}
	typeID, err := identity.ParseSemanticTypeID(encoded.Type)
	if err != nil {
		return Declaration{}, err
	}
	constant, err := decodeConstant(encoded.Constant)
	if err != nil {
		return Declaration{}, err
	}
	return NewDeclaration(
		id,
		pkg,
		identity.SemanticObjectClass(encoded.Class),
		encoded.Name,
		typeID,
		encoded.Exported,
		constant,
		authority,
	)
}

func decodeBinding(
	encoded wireBinding,
	authority Authority,
) (Binding, error) {
	id, err := identity.ParseSemanticBindingID(encoded.ID)
	if err != nil {
		return Binding{}, err
	}
	pkg, err := identity.ParsePackageID(encoded.Package)
	if err != nil {
		return Binding{}, err
	}
	definition, err := parseOptionalDefinition(encoded.Definition)
	if err != nil {
		return Binding{}, err
	}
	typeID, err := parseOptionalType(encoded.Type)
	if err != nil {
		return Binding{}, err
	}
	source, err := parseOptionalOccurrence(encoded.Source)
	if err != nil {
		return Binding{}, err
	}
	captures, err := parseDefinitions(encoded.CapturedBy)
	if err != nil {
		return Binding{}, err
	}
	return NewBinding(
		id,
		pkg,
		definition,
		identity.SemanticBindingRole(encoded.Role),
		encoded.Name,
		typeID,
		source,
		captures,
		authority,
	)
}

func decodeType(encoded wireType) (Type, error) {
	declaration, err := parseOptionalDeclaration(encoded.Declaration)
	if err != nil {
		return Type{}, err
	}
	spec := TypeSpec{
		Kind:        TypeKind(encoded.Kind),
		Basic:       BasicKind(encoded.Basic),
		Declaration: declaration,
		Length:      encoded.Length,
		Direction:   ChannelDirection(encoded.Direction),
		TypeSet:     TypeSetKind(encoded.TypeSet),
		Comparable:  encoded.Comparable,
	}
	if encoded.ParameterRole != 0 ||
		encoded.ParameterDeclaration != "" ||
		encoded.ParameterDefinition != "" {
		parameterDeclaration, err := parseOptionalDeclaration(
			encoded.ParameterDeclaration,
		)
		if err != nil {
			return Type{}, err
		}
		parameterDefinition, err := parseOptionalDefinition(
			encoded.ParameterDefinition,
		)
		if err != nil {
			return Type{}, err
		}
		spec.Parameter, err = NewTypeParameterOwner(
			parameterDeclaration,
			parameterDefinition,
			TypeParameterRole(encoded.ParameterRole),
			encoded.ParameterOrdinal,
		)
		if err != nil {
			return Type{}, err
		}
	}
	if spec.Arguments, err = parseTypes(encoded.Arguments); err != nil {
		return Type{}, err
	}
	if spec.Underlying, err = parseOptionalType(encoded.Underlying); err != nil {
		return Type{}, err
	}
	if spec.Target, err = parseOptionalType(encoded.Target); err != nil {
		return Type{}, err
	}
	if spec.Constraint, err = parseOptionalType(encoded.Constraint); err != nil {
		return Type{}, err
	}
	if spec.Element, err = parseOptionalType(encoded.Element); err != nil {
		return Type{}, err
	}
	if spec.Key, err = parseOptionalType(encoded.Key); err != nil {
		return Type{}, err
	}
	if spec.Signature, err = decodeSignature(encoded.Signature); err != nil {
		return Type{}, err
	}
	for _, field := range encoded.Fields {
		pkg, parseErr := parseOptionalPackage(field.Package)
		if parseErr != nil {
			return Type{}, parseErr
		}
		typeID, parseErr := identity.ParseSemanticTypeID(field.Type)
		if parseErr != nil {
			return Type{}, parseErr
		}
		spec.Fields = append(spec.Fields, TypeField{
			Name: field.Name, Package: pkg, Type: typeID,
			Embedded: field.Embedded, Tag: field.Tag,
			Ordinal: field.Ordinal,
		})
	}
	for _, method := range encoded.Methods {
		pkg, parseErr := parseOptionalPackage(method.Package)
		if parseErr != nil {
			return Type{}, parseErr
		}
		signature, parseErr := identity.ParseSemanticTypeID(
			method.Signature,
		)
		if parseErr != nil {
			return Type{}, parseErr
		}
		spec.Methods = append(spec.Methods, TypeMethod{
			Name: method.Name, Package: pkg,
			Signature: signature, Ordinal: method.Ordinal,
		})
	}
	if spec.Embeddeds, err = parseTypes(encoded.Embeddeds); err != nil {
		return Type{}, err
	}
	for _, term := range encoded.Terms {
		typeID, parseErr := identity.ParseSemanticTypeID(term.Type)
		if parseErr != nil {
			return Type{}, parseErr
		}
		spec.Terms = append(spec.Terms, TypeTerm{
			Tilde: term.Tilde, Type: typeID,
		})
	}
	if spec.Elements, err = parseTypes(encoded.Elements); err != nil {
		return Type{}, err
	}
	record, err := NewType(spec)
	if err != nil {
		return Type{}, fmt.Errorf(
			"semantic provider type %s: %w",
			encoded.ID, err,
		)
	}
	if record.ID().String() != encoded.ID {
		return Type{}, fmt.Errorf(
			"semantic provider type identity %s does not match canonical %s",
			encoded.ID, record.ID(),
		)
	}
	return record, nil
}

func decodeSignature(
	encoded wireSignature,
) (Signature, error) {
	var (
		out Signature
		err error
	)
	if out.Receiver, err = parseOptionalType(encoded.Receiver); err != nil {
		return Signature{}, err
	}
	if out.ReceiverTypeParameters, err = parseTypes(
		encoded.ReceiverTypeParameters,
	); err != nil {
		return Signature{}, err
	}
	if out.TypeParameters, err = parseTypes(
		encoded.TypeParameters,
	); err != nil {
		return Signature{}, err
	}
	if out.Parameters, err = parseTypes(encoded.Parameters); err != nil {
		return Signature{}, err
	}
	if out.Results, err = parseTypes(encoded.Results); err != nil {
		return Signature{}, err
	}
	out.Variadic = encoded.Variadic
	return out, nil
}

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf(
				"semantic provider artifact has trailing JSON",
			)
		}
		return err
	}
	return nil
}
