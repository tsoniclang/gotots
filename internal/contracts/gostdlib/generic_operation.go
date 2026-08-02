package gostdlib

import (
	"sort"
	"strconv"
	"strings"
)

type GenericOperationKind string

const (
	GenericOperationInvalid              GenericOperationKind = ""
	GenericOperationCopy                 GenericOperationKind = "copy"
	GenericOperationZero                 GenericOperationKind = "zero"
	GenericOperationEqual                GenericOperationKind = "equal"
	GenericOperationBinaryLess           GenericOperationKind = "binary-less"
	GenericOperationConvert              GenericOperationKind = "convert"
	GenericOperationMapConstruct         GenericOperationKind = "map-construct"
	GenericOperationToStorage            GenericOperationKind = "to-storage"
	GenericOperationFromStorage          GenericOperationKind = "from-storage"
	GenericOperationToContainerStorage   GenericOperationKind = "to-container-storage"
	GenericOperationFromContainerStorage GenericOperationKind = "from-container-storage"
	GenericOperationInterfaceAssertOK    GenericOperationKind = "interface-assert-ok"
)

func (k GenericOperationKind) Valid() bool {
	switch k {
	case GenericOperationCopy,
		GenericOperationZero,
		GenericOperationEqual,
		GenericOperationBinaryLess,
		GenericOperationConvert,
		GenericOperationMapConstruct,
		GenericOperationToStorage,
		GenericOperationFromStorage,
		GenericOperationToContainerStorage,
		GenericOperationFromContainerStorage,
		GenericOperationInterfaceAssertOK:
		return true
	default:
		return false
	}
}

type ContractTypeKind string

const (
	ContractTypeInvalid           ContractTypeKind = ""
	ContractTypeParameter         ContractTypeKind = "type-parameter"
	ContractTypeBool              ContractTypeKind = "bool"
	ContractTypeInt               ContractTypeKind = "int"
	ContractTypeSlice             ContractTypeKind = "slice"
	ContractTypeMap               ContractTypeKind = "map"
	ContractTypeCallableParameter ContractTypeKind = "callable-parameter"
	contractTypeMaximumDepth                       = 32
)

type ContractTypeDocument struct {
	Kind              ContractTypeKind      `json:"kind"`
	TypeParameter     *int                  `json:"typeParameter,omitempty"`
	CallableParameter *int                  `json:"callableParameter,omitempty"`
	Key               *ContractTypeDocument `json:"key,omitempty"`
	Element           *ContractTypeDocument `json:"element,omitempty"`
}

func ContractTypeParameterReference(
	index int,
) ContractTypeDocument {
	selected := index
	return ContractTypeDocument{
		Kind:          ContractTypeParameter,
		TypeParameter: &selected,
	}
}

func ContractCallableParameterReference(
	index int,
) ContractTypeDocument {
	selected := index
	return ContractTypeDocument{
		Kind:              ContractTypeCallableParameter,
		CallableParameter: &selected,
	}
}

func ContractBoolReference() ContractTypeDocument {
	return ContractTypeDocument{Kind: ContractTypeBool}
}

func ContractIntReference() ContractTypeDocument {
	return ContractTypeDocument{Kind: ContractTypeInt}
}

func ContractSliceReference(
	element ContractTypeDocument,
) ContractTypeDocument {
	return ContractTypeDocument{
		Kind:    ContractTypeSlice,
		Element: cloneContractTypePointer(&element),
	}
}

func ContractMapReference(
	key ContractTypeDocument,
	element ContractTypeDocument,
) ContractTypeDocument {
	return ContractTypeDocument{
		Kind:    ContractTypeMap,
		Key:     cloneContractTypePointer(&key),
		Element: cloneContractTypePointer(&element),
	}
}

type GenericOperationDocument struct {
	Kind       GenericOperationKind   `json:"kind"`
	Parameters []ContractTypeDocument `json:"parameters"`
	Results    []ContractTypeDocument `json:"results"`
}

func CanonicalGenericOperations(
	source []GenericOperationDocument,
) ([]GenericOperationDocument, error) {
	if err := validateGenericOperations(source, "genericOperations", false); err != nil {
		return nil, err
	}
	result := cloneGenericOperations(source)
	sort.Slice(result, func(left, right int) bool {
		return genericOperationDocumentKey(result[left]) <
			genericOperationDocumentKey(result[right])
	})
	if err := validateGenericOperations(result, "genericOperations", true); err != nil {
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
		result[index].Parameters = cloneContractTypes(operation.Parameters)
		result[index].Results = cloneContractTypes(operation.Results)
	}
	return result
}

func cloneContractTypes(
	source []ContractTypeDocument,
) []ContractTypeDocument {
	result := make([]ContractTypeDocument, len(source))
	for index := range source {
		result[index] = cloneContractType(source[index])
	}
	return result
}

func cloneContractType(
	source ContractTypeDocument,
) ContractTypeDocument {
	result := source
	if source.TypeParameter != nil {
		selected := *source.TypeParameter
		result.TypeParameter = &selected
	}
	if source.CallableParameter != nil {
		selected := *source.CallableParameter
		result.CallableParameter = &selected
	}
	result.Key = cloneContractTypePointer(source.Key)
	result.Element = cloneContractTypePointer(source.Element)
	return result
}

func cloneContractTypePointer(
	source *ContractTypeDocument,
) *ContractTypeDocument {
	if source == nil {
		return nil
	}
	result := cloneContractType(*source)
	return &result
}

func validateGenericOperations(
	operations []GenericOperationDocument,
	field string,
	requireOrder bool,
) error {
	previous := ""
	for index, operation := range operations {
		selected := field + "[" + strconv.Itoa(index) + "]"
		if !operation.Kind.Valid() {
			return manifestError(selected+".kind", "value is invalid")
		}
		for parameterIndex := range operation.Parameters {
			if err := validateContractType(
				operation.Parameters[parameterIndex],
				selected+".parameters["+strconv.Itoa(parameterIndex)+"]",
				0,
				make(map[*ContractTypeDocument]struct{}),
			); err != nil {
				return err
			}
		}
		for resultIndex := range operation.Results {
			if err := validateContractType(
				operation.Results[resultIndex],
				selected+".results["+strconv.Itoa(resultIndex)+"]",
				0,
				make(map[*ContractTypeDocument]struct{}),
			); err != nil {
				return err
			}
		}
		if !validGenericOperationShape(operation) {
			return manifestError(selected, "operation signature is invalid")
		}
		key := genericOperationDocumentKey(operation)
		if requireOrder && previous != "" && key <= previous {
			return manifestError(field, "operations are not strictly ordered")
		}
		previous = key
	}
	return nil
}

func validateContractType(
	source ContractTypeDocument,
	field string,
	depth int,
	stack map[*ContractTypeDocument]struct{},
) error {
	if depth > contractTypeMaximumDepth {
		return manifestError(field, "type expression is too deep")
	}
	validateChild := func(
		child *ContractTypeDocument,
		childField string,
	) error {
		if child == nil {
			return manifestError(childField, "value is missing")
		}
		if _, cyclic := stack[child]; cyclic {
			return manifestError(childField, "type expression is cyclic")
		}
		stack[child] = struct{}{}
		err := validateContractType(*child, childField, depth+1, stack)
		delete(stack, child)
		return err
	}
	switch source.Kind {
	case ContractTypeParameter:
		if source.TypeParameter == nil || *source.TypeParameter < 0 ||
			source.CallableParameter != nil || source.Key != nil || source.Element != nil {
			return manifestError(field, "type-parameter expression is invalid")
		}
	case ContractTypeCallableParameter:
		if source.CallableParameter == nil || *source.CallableParameter < 0 ||
			source.TypeParameter != nil || source.Key != nil || source.Element != nil {
			return manifestError(field, "callable-parameter expression is invalid")
		}
	case ContractTypeBool, ContractTypeInt:
		if source.TypeParameter != nil || source.CallableParameter != nil ||
			source.Key != nil || source.Element != nil {
			return manifestError(field, "basic type expression is invalid")
		}
	case ContractTypeSlice:
		if source.TypeParameter != nil || source.CallableParameter != nil || source.Key != nil {
			return manifestError(field, "slice type expression is invalid")
		}
		return validateChild(source.Element, field+".element")
	case ContractTypeMap:
		if source.TypeParameter != nil || source.CallableParameter != nil {
			return manifestError(field, "map type expression is invalid")
		}
		if err := validateChild(source.Key, field+".key"); err != nil {
			return err
		}
		return validateChild(source.Element, field+".element")
	default:
		return manifestError(field+".kind", "value is invalid")
	}
	return nil
}

func validGenericOperationShape(operation GenericOperationDocument) bool {
	same := func(left, right ContractTypeDocument) bool {
		return contractTypeKey(left) == contractTypeKey(right)
	}
	typeParameter := func(source ContractTypeDocument) bool {
		return source.Kind == ContractTypeParameter
	}
	booleanResult := func() bool {
		return len(operation.Results) == 1 &&
			operation.Results[0].Kind == ContractTypeBool
	}
	switch operation.Kind {
	case GenericOperationCopy:
		return len(operation.Parameters) == 1 &&
			len(operation.Results) == 1 &&
			same(operation.Parameters[0], operation.Results[0])
	case GenericOperationZero:
		return len(operation.Parameters) == 0 &&
			len(operation.Results) == 1
	case GenericOperationEqual, GenericOperationBinaryLess:
		return len(operation.Parameters) == 2 &&
			same(operation.Parameters[0], operation.Parameters[1]) &&
			booleanResult()
	case GenericOperationConvert:
		return len(operation.Parameters) == 1 && len(operation.Results) == 1
	case GenericOperationMapConstruct:
		if len(operation.Parameters) < 1 || len(operation.Parameters) > 2 ||
			len(operation.Results) != 1 ||
			operation.Results[0].Kind != ContractTypeMap ||
			operation.Results[0].Element == nil ||
			!same(operation.Parameters[0], *operation.Results[0].Element) {
			return false
		}
		return len(operation.Parameters) == 1 ||
			operation.Parameters[1].Kind == ContractTypeInt
	case GenericOperationToStorage,
		GenericOperationFromStorage,
		GenericOperationToContainerStorage,
		GenericOperationFromContainerStorage:
		return len(operation.Parameters) == 1 &&
			len(operation.Results) == 1 &&
			typeParameter(operation.Parameters[0]) &&
			same(operation.Parameters[0], operation.Results[0])
	case GenericOperationInterfaceAssertOK:
		return len(operation.Parameters) == 1 &&
			operation.Parameters[0].Kind == ContractTypeCallableParameter &&
			len(operation.Results) == 2 &&
			typeParameter(operation.Results[0]) &&
			operation.Results[1].Kind == ContractTypeBool
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
		result.WriteString(contractTypeKey(parameter))
	}
	result.WriteString("|r")
	for _, selected := range operation.Results {
		result.WriteByte(':')
		result.WriteString(contractTypeKey(selected))
	}
	return result.String()
}

func contractTypeKey(source ContractTypeDocument) string {
	switch source.Kind {
	case ContractTypeParameter:
		if source.TypeParameter == nil {
			return "parameter(?)"
		}
		return "parameter(" + strconv.Itoa(*source.TypeParameter) + ")"
	case ContractTypeCallableParameter:
		if source.CallableParameter == nil {
			return "callable-parameter(?)"
		}
		return "callable-parameter(" + strconv.Itoa(*source.CallableParameter) + ")"
	case ContractTypeBool, ContractTypeInt:
		return string(source.Kind)
	case ContractTypeSlice:
		if source.Element == nil {
			return "slice(?)"
		}
		return "slice(" + contractTypeKey(*source.Element) + ")"
	case ContractTypeMap:
		if source.Key == nil || source.Element == nil {
			return "map(?,?)"
		}
		return "map(" + contractTypeKey(*source.Key) + "," +
			contractTypeKey(*source.Element) + ")"
	default:
		return "invalid"
	}
}
