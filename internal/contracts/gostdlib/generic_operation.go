package gostdlib

import (
	"sort"
	"strconv"
	"strings"
)

type GenericOperationKind string

const (
	GenericOperationInvalid GenericOperationKind = ""
	GenericOperationCopy    GenericOperationKind = "copy"
	GenericOperationZero    GenericOperationKind = "zero"
)

func (k GenericOperationKind) Valid() bool {
	return k == GenericOperationCopy || k == GenericOperationZero
}

type GenericOperationTypeDocument struct {
	TypeParameter int `json:"typeParameter"`
}

type GenericOperationDocument struct {
	Kind       GenericOperationKind           `json:"kind"`
	Parameters []GenericOperationTypeDocument `json:"parameters"`
	Results    []GenericOperationTypeDocument `json:"results"`
}

func CanonicalGenericOperations(
	source []GenericOperationDocument,
) ([]GenericOperationDocument, error) {
	result := cloneGenericOperations(source)
	sort.Slice(result, func(left, right int) bool {
		return genericOperationDocumentKey(result[left]) <
			genericOperationDocumentKey(result[right])
	})
	if err := validateGenericOperations(result, "genericOperations"); err != nil {
		return nil, err
	}
	return result, nil
}

func cloneGenericOperations(
	source []GenericOperationDocument,
) []GenericOperationDocument {
	result := make([]GenericOperationDocument, len(source))
	for index, operation := range source {
		result[index] = operation
		result[index].Parameters = append(
			[]GenericOperationTypeDocument(nil),
			operation.Parameters...,
		)
		result[index].Results = append(
			[]GenericOperationTypeDocument(nil),
			operation.Results...,
		)
	}
	return result
}

func validateGenericOperations(
	operations []GenericOperationDocument,
	field string,
) error {
	previous := ""
	for index, operation := range operations {
		selected := field + "[" + strconv.Itoa(index) + "]"
		if !operation.Kind.Valid() {
			return manifestError(selected+".kind", "value is invalid")
		}
		for parameterIndex, parameter := range operation.Parameters {
			if parameter.TypeParameter < 0 {
				return manifestError(
					selected+".parameters["+strconv.Itoa(parameterIndex)+"]",
					"type-parameter index is negative",
				)
			}
		}
		for resultIndex, result := range operation.Results {
			if result.TypeParameter < 0 {
				return manifestError(
					selected+".results["+strconv.Itoa(resultIndex)+"]",
					"type-parameter index is negative",
				)
			}
		}
		if !validGenericOperationShape(operation) {
			return manifestError(selected, "operation signature is invalid")
		}
		key := genericOperationDocumentKey(operation)
		if previous != "" && key <= previous {
			return manifestError(field, "operations are not strictly ordered")
		}
		previous = key
	}
	return nil
}

func validGenericOperationShape(operation GenericOperationDocument) bool {
	switch operation.Kind {
	case GenericOperationCopy:
		return len(operation.Parameters) == 1 &&
			len(operation.Results) == 1 &&
			operation.Parameters[0].TypeParameter ==
				operation.Results[0].TypeParameter
	case GenericOperationZero:
		return len(operation.Parameters) == 0 &&
			len(operation.Results) == 1
	default:
		return false
	}
}

func genericOperationDocumentKey(operation GenericOperationDocument) string {
	var result strings.Builder
	result.WriteString(string(operation.Kind))
	result.WriteString("|p")
	for _, parameter := range operation.Parameters {
		result.WriteByte(':')
		result.WriteString(strconv.Itoa(parameter.TypeParameter))
	}
	result.WriteString("|r")
	for _, selected := range operation.Results {
		result.WriteByte(':')
		result.WriteString(strconv.Itoa(selected.TypeParameter))
	}
	return result.String()
}
