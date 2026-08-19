package gostdlib

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

const InvocationTransportSchemaVersion = 1

type InvocationTransportContractDocument struct {
	SchemaVersion   int                           `json:"schemaVersion"`
	DeclarationRoot string                        `json:"declarationRoot"`
	Transports      []InvocationTransportDocument `json:"transports"`
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
	SourceIdentity         string                            `json:"sourceIdentity"`
	Specifier              string                            `json:"specifier"`
	SourcePath             string                            `json:"sourcePath"`
	DeclarationPath        string                            `json:"declarationPath"`
	Export                 string                            `json:"export"`
	Member                 string                            `json:"member"`
	TargetType             string                            `json:"targetType"`
	TargetFingerprint      string                            `json:"targetFingerprint"`
	InputParameters        []int                             `json:"inputParameters,omitempty"`
	ResultOriginParameters []int                             `json:"resultOriginParameters,omitempty"`
	State                  *InvocationTransportStateDocument `json:"state,omitempty"`
}

func (d InvocationTransportDocument) Key() string {
	return strings.Join([]string{d.Specifier, d.Export, d.Member}, "\x00")
}

func validateInvocationTransport(
	document InvocationTransportDocument,
	field string,
	specifiers map[string]struct{},
	sourceIdentities map[string]struct{},
) error {
	switch {
	case document.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case document.Export == "" || document.Member == "":
		return manifestError(field, "target export or member is empty")
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
	if _, ok := sourceIdentities[document.SourceIdentity]; !ok {
		return manifestError(field+".sourceIdentity", "source declaration is absent")
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
	if document.State == nil {
		return nil
	}
	state := document.State
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
