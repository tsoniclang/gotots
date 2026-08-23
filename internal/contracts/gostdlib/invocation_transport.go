package gostdlib

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

const InvocationTransportSchemaVersion = 2

type InvocationTransportContractDocument struct {
	SchemaVersion   int                           `json:"schemaVersion"`
	DeclarationRoot string                        `json:"declarationRoot"`
	Transports      []InvocationTransportDocument `json:"transports"`
}

type InvocationTransportAccessKind string

const (
	InvocationTransportAccessInvalid      InvocationTransportAccessKind = ""
	InvocationTransportAccessExport       InvocationTransportAccessKind = "export"
	InvocationTransportAccessStaticMethod InvocationTransportAccessKind = "static-method"
)

func (k InvocationTransportAccessKind) Valid() bool {
	return k == InvocationTransportAccessExport ||
		k == InvocationTransportAccessStaticMethod
}

type InvocationTransportTargetDocument struct {
	Specifier         string                        `json:"specifier"`
	SourcePath        string                        `json:"sourcePath"`
	DeclarationPath   string                        `json:"declarationPath"`
	Access            InvocationTransportAccessKind `json:"access"`
	Export            string                        `json:"export"`
	Member            string                        `json:"member,omitempty"`
	TargetType        string                        `json:"targetType"`
	TargetFingerprint string                        `json:"targetFingerprint"`
}

func (d InvocationTransportTargetDocument) Key() string {
	return strings.Join(
		[]string{d.Specifier, string(d.Access), d.Export, d.Member},
		"\x00",
	)
}

type InvocationTransportConditionalDocument struct {
	CallableParameters []int                             `json:"callableParameters"`
	Replacement        InvocationTransportTargetDocument `json:"replacement"`
}

type InvocationTransportStateKind string

const (
	InvocationTransportStateInvalid InvocationTransportStateKind = ""
	InvocationTransportStateCreate  InvocationTransportStateKind = "create"
	InvocationTransportStateAlias   InvocationTransportStateKind = "alias"
	InvocationTransportStateAccess  InvocationTransportStateKind = "access"
)

func (k InvocationTransportStateKind) Valid() bool {
	return k == InvocationTransportStateCreate ||
		k == InvocationTransportStateAlias ||
		k == InvocationTransportStateAccess
}

type InvocationTransportStateDocument struct {
	Kind             InvocationTransportStateKind `json:"kind"`
	CarrierParameter *int                         `json:"carrierParameter,omitempty"`
	Read             bool                         `json:"read,omitempty"`
	WriteParameters  []int                        `json:"writeParameters,omitempty"`
}

type InvocationTransportDocument struct {
	SourceIdentity         string                                  `json:"sourceIdentity"`
	Target                 InvocationTransportTargetDocument       `json:"target"`
	InputParameters        []int                                   `json:"inputParameters,omitempty"`
	ResultOriginParameters []int                                   `json:"resultOriginParameters,omitempty"`
	State                  *InvocationTransportStateDocument       `json:"state,omitempty"`
	Conditional            *InvocationTransportConditionalDocument `json:"conditional,omitempty"`
}

func (d InvocationTransportDocument) Key() string {
	return d.Target.Key()
}

func validateInvocationTransport(
	document InvocationTransportDocument,
	field string,
	specifiers map[string]struct{},
	sourceIdentities map[string]struct{},
) error {
	if document.SourceIdentity == "" {
		return manifestError(field+".sourceIdentity", "value is empty")
	}
	if _, ok := sourceIdentities[document.SourceIdentity]; !ok {
		return manifestError(field+".sourceIdentity", "source declaration is absent")
	}
	if err := validateInvocationTransportTarget(
		document.Target,
		field+".target",
		specifiers,
	); err != nil {
		return err
	}
	if err := validateParameterIndexes(
		document.InputParameters,
		field+".inputParameters",
	); err != nil {
		return err
	}
	if err := validateParameterIndexes(
		document.ResultOriginParameters,
		field+".resultOriginParameters",
	); err != nil {
		return err
	}
	if err := validateInvocationTransportState(document.State, field); err != nil {
		return err
	}
	return validateInvocationTransportConditional(
		document,
		field,
		specifiers,
	)
}

func validateInvocationTransportTarget(
	document InvocationTransportTargetDocument,
	field string,
	specifiers map[string]struct{},
) error {
	switch {
	case !document.Access.Valid():
		return manifestError(field+".access", "value is invalid")
	case document.Export == "":
		return manifestError(field+".export", "value is empty")
	case document.Access == InvocationTransportAccessExport && document.Member != "":
		return manifestError(field+".member", "export target cannot name a member")
	case document.Access == InvocationTransportAccessStaticMethod && document.Member == "":
		return manifestError(field+".member", "static-method target has no member")
	case document.TargetType == "":
		return manifestError(field+".targetType", "value is empty")
	case !validDigest(document.TargetFingerprint):
		return manifestError(field+".targetFingerprint", "value is not a sha256 digest")
	case !sourcePath(document.SourcePath):
		return manifestError(field+".sourcePath", "value is not a provider source path")
	case !invocationDeclarationPath(document.DeclarationPath):
		return manifestError(field+".declarationPath", "value is not a provider declaration path")
	}
	if _, ok := specifiers[document.Specifier]; !ok {
		return manifestError(field+".specifier", "provider module is absent")
	}
	return nil
}

func validateInvocationTransportState(
	state *InvocationTransportStateDocument,
	field string,
) error {
	if state == nil {
		return nil
	}
	if !state.Kind.Valid() {
		return manifestError(field+".state.kind", "value is invalid")
	}
	if err := validateParameterIndexes(
		state.WriteParameters,
		field+".state.writeParameters",
	); err != nil {
		return err
	}
	switch state.Kind {
	case InvocationTransportStateCreate:
		if (state.CarrierParameter != nil && *state.CarrierParameter < 0) ||
			state.Read || len(state.WriteParameters) != 0 {
			return manifestError(field+".state", "create state shape is invalid")
		}
	case InvocationTransportStateAlias:
		if state.CarrierParameter == nil || *state.CarrierParameter < 0 ||
			state.Read || len(state.WriteParameters) != 0 {
			return manifestError(field+".state", "alias state shape is invalid")
		}
	case InvocationTransportStateAccess:
		if state.CarrierParameter == nil || *state.CarrierParameter < 0 {
			return manifestError(field+".state", "access state shape is invalid")
		}
	}
	return nil
}

func validateInvocationTransportConditional(
	document InvocationTransportDocument,
	field string,
	specifiers map[string]struct{},
) error {
	conditional := document.Conditional
	if conditional == nil {
		return nil
	}
	if len(conditional.CallableParameters) == 0 {
		return manifestError(
			field+".conditional.callableParameters",
			"set is empty",
		)
	}
	if err := validateParameterIndexes(
		conditional.CallableParameters,
		field+".conditional.callableParameters",
	); err != nil {
		return err
	}
	for _, parameter := range conditional.CallableParameters {
		if !slices.Contains(document.InputParameters, parameter) {
			return manifestError(
				field+".conditional.callableParameters",
				"value is not a certified input parameter",
			)
		}
	}
	if err := validateInvocationTransportTarget(
		conditional.Replacement,
		field+".conditional.replacement",
		specifiers,
	); err != nil {
		return err
	}
	if document.Target.Access != InvocationTransportAccessExport ||
		conditional.Replacement.Access != InvocationTransportAccessExport {
		return manifestError(field+".conditional", "replacement target is not an export")
	}
	if document.Target.Specifier != conditional.Replacement.Specifier {
		return manifestError(field+".conditional", "replacement changes provider module")
	}
	if document.Target.Key() == conditional.Replacement.Key() {
		return manifestError(field+".conditional", "replacement equals canonical target")
	}
	return nil
}

func validateInvocationTransportContract(
	document *InvocationTransportContractDocument,
	specifiers map[string]struct{},
	sourceIdentities map[string]struct{},
) error {
	if document == nil {
		return nil
	}
	switch {
	case document.SchemaVersion != InvocationTransportSchemaVersion:
		return manifestError("invocationTransportContract.schemaVersion", "unsupported schema")
	case !invocationDeclarationRoot(document.DeclarationRoot):
		return manifestError("invocationTransportContract.declarationRoot", "value is invalid")
	case len(document.Transports) == 0:
		return manifestError("invocationTransportContract.transports", "set is empty")
	}
	previous := ""
	for index, transport := range document.Transports {
		field := fmt.Sprintf("invocationTransportContract.transports[%d]", index)
		if err := validateInvocationTransport(
			transport,
			field,
			specifiers,
			sourceIdentities,
		); err != nil {
			return err
		}
		key := transport.Key()
		if previous != "" && key <= previous {
			return manifestError(
				"invocationTransportContract.transports",
				"values are not strictly ordered",
			)
		}
		previous = key
	}
	return nil
}

func invocationDeclarationRoot(value string) bool {
	return value != "" && value == path.Clean(value) && !strings.HasPrefix(value, "/")
}

func invocationDeclarationPath(value string) bool {
	return value != "" && value == path.Clean(value) &&
		!strings.HasPrefix(value, "/") &&
		!strings.HasPrefix(value, "../") &&
		strings.HasSuffix(value, ".d.ts")
}

func validateParameterIndexes(values []int, field string) error {
	for index, value := range values {
		if value < 0 {
			return manifestError(field, "parameter index is negative")
		}
		if index != 0 && value <= values[index-1] {
			return manifestError(field, "values are not strictly ordered")
		}
	}
	return nil
}

func ValidateInvocationTransportIndexes(
	document InvocationTransportDocument,
	parameterCount int,
	field string,
) error {
	if err := validateParameterIndexes(
		document.InputParameters,
		field+".inputParameters",
	); err != nil {
		return err
	}
	if err := validateParameterIndexes(
		document.ResultOriginParameters,
		field+".resultOriginParameters",
	); err != nil {
		return err
	}
	if document.State != nil {
		if err := validateParameterIndexes(
			document.State.WriteParameters,
			field+".state.writeParameters",
		); err != nil {
			return err
		}
		if document.State.CarrierParameter != nil &&
			*document.State.CarrierParameter < 0 {
			return fmt.Errorf("%s.state.carrierParameter is negative", field)
		}
	}
	indexes := append([]int(nil), document.InputParameters...)
	indexes = append(indexes, document.ResultOriginParameters...)
	if document.State != nil {
		indexes = append(indexes, document.State.WriteParameters...)
		if document.State.CarrierParameter != nil {
			indexes = append(indexes, *document.State.CarrierParameter)
		}
	}
	if document.Conditional != nil {
		indexes = append(indexes, document.Conditional.CallableParameters...)
	}
	for _, index := range indexes {
		if index >= parameterCount {
			return fmt.Errorf("parameter index %d exceeds target arity %d", index, parameterCount)
		}
	}
	return nil
}

func cloneInvocationTransports(
	source []InvocationTransportDocument,
) []InvocationTransportDocument {
	result := make([]InvocationTransportDocument, len(source))
	for index, document := range source {
		result[index] = document
		result[index].InputParameters = slices.Clone(document.InputParameters)
		result[index].ResultOriginParameters = slices.Clone(
			document.ResultOriginParameters,
		)
		if document.State != nil {
			state := *document.State
			state.WriteParameters = slices.Clone(document.State.WriteParameters)
			if document.State.CarrierParameter != nil {
				carrier := *document.State.CarrierParameter
				state.CarrierParameter = &carrier
			}
			result[index].State = &state
		}
		if document.Conditional != nil {
			conditional := *document.Conditional
			conditional.CallableParameters = slices.Clone(
				document.Conditional.CallableParameters,
			)
			result[index].Conditional = &conditional
		}
	}
	return result
}

func cloneInvocationTransportContract(
	source *InvocationTransportContractDocument,
) *InvocationTransportContractDocument {
	if source == nil {
		return nil
	}
	result := *source
	result.Transports = cloneInvocationTransports(source.Transports)
	return &result
}
