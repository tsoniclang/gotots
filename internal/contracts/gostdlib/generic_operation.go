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

type GenericOperationTypeKind string

const (
	GenericOperationTypeInvalid           GenericOperationTypeKind = ""
	GenericOperationTypeParameter         GenericOperationTypeKind = "type-parameter"
	GenericOperationTypeBool              GenericOperationTypeKind = "bool"
	GenericOperationTypeInt               GenericOperationTypeKind = "int"
	GenericOperationTypeSlice             GenericOperationTypeKind = "slice"
	GenericOperationTypeMap               GenericOperationTypeKind = "map"
	GenericOperationTypeCallableParameter GenericOperationTypeKind = "callable-parameter"
	genericOperationTypeMaximumDepth                               = 32
)

type GenericOperationTypeDocument struct {
	Kind              GenericOperationTypeKind      `json:"kind"`
	TypeParameter     *int                          `json:"typeParameter,omitempty"`
	CallableParameter *int                          `json:"callableParameter,omitempty"`
	Key               *GenericOperationTypeDocument `json:"key,omitempty"`
	Element           *GenericOperationTypeDocument `json:"element,omitempty"`
}

func GenericOperationTypeParameterReference(
	index int,
) GenericOperationTypeDocument {
	selected := index
	return GenericOperationTypeDocument{
		Kind:          GenericOperationTypeParameter,
		TypeParameter: &selected,
	}
}

func GenericOperationCallableParameterReference(
	index int,
) GenericOperationTypeDocument {
	selected := index
	return GenericOperationTypeDocument{
		Kind:              GenericOperationTypeCallableParameter,
		CallableParameter: &selected,
	}
}

func GenericOperationBoolReference() GenericOperationTypeDocument {
	return GenericOperationTypeDocument{Kind: GenericOperationTypeBool}
}

func GenericOperationIntReference() GenericOperationTypeDocument {
	return GenericOperationTypeDocument{Kind: GenericOperationTypeInt}
}

func GenericOperationSliceReference(
	element GenericOperationTypeDocument,
) GenericOperationTypeDocument {
	return GenericOperationTypeDocument{
		Kind:    GenericOperationTypeSlice,
		Element: cloneGenericOperationTypePointer(&element),
	}
}

func GenericOperationMapReference(
	key GenericOperationTypeDocument,
	element GenericOperationTypeDocument,
) GenericOperationTypeDocument {
	return GenericOperationTypeDocument{
		Kind:    GenericOperationTypeMap,
		Key:     cloneGenericOperationTypePointer(&key),
		Element: cloneGenericOperationTypePointer(&element),
	}
}

type GenericOperationDocument struct {
	Kind       GenericOperationKind           `json:"kind"`
	Parameters []GenericOperationTypeDocument `json:"parameters"`
	Results    []GenericOperationTypeDocument `json:"results"`
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
		result[index].Parameters = cloneGenericOperationTypes(operation.Parameters)
		result[index].Results = cloneGenericOperationTypes(operation.Results)
	}
	return result
}

func cloneGenericOperationTypes(
	source []GenericOperationTypeDocument,
) []GenericOperationTypeDocument {
	result := make([]GenericOperationTypeDocument, len(source))
	for index := range source {
		result[index] = cloneGenericOperationType(source[index])
	}
	return result
}

func cloneGenericOperationType(
	source GenericOperationTypeDocument,
) GenericOperationTypeDocument {
	result := source
	if source.TypeParameter != nil {
		selected := *source.TypeParameter
		result.TypeParameter = &selected
	}
	if source.CallableParameter != nil {
		selected := *source.CallableParameter
		result.CallableParameter = &selected
	}
	result.Key = cloneGenericOperationTypePointer(source.Key)
	result.Element = cloneGenericOperationTypePointer(source.Element)
	return result
}

func cloneGenericOperationTypePointer(
	source *GenericOperationTypeDocument,
) *GenericOperationTypeDocument {
	if source == nil {
		return nil
	}
	result := cloneGenericOperationType(*source)
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
			if err := validateGenericOperationType(
				operation.Parameters[parameterIndex],
				selected+".parameters["+strconv.Itoa(parameterIndex)+"]",
				0,
				make(map[*GenericOperationTypeDocument]struct{}),
			); err != nil {
				return err
			}
		}
		for resultIndex := range operation.Results {
			if err := validateGenericOperationType(
				operation.Results[resultIndex],
				selected+".results["+strconv.Itoa(resultIndex)+"]",
				0,
				make(map[*GenericOperationTypeDocument]struct{}),
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

func validateGenericOperationType(
	source GenericOperationTypeDocument,
	field string,
	depth int,
	stack map[*GenericOperationTypeDocument]struct{},
) error {
	if depth > genericOperationTypeMaximumDepth {
		return manifestError(field, "type expression is too deep")
	}
	validateChild := func(
		child *GenericOperationTypeDocument,
		childField string,
	) error {
		if child == nil {
			return manifestError(childField, "value is missing")
		}
		if _, cyclic := stack[child]; cyclic {
			return manifestError(childField, "type expression is cyclic")
		}
		stack[child] = struct{}{}
		err := validateGenericOperationType(*child, childField, depth+1, stack)
		delete(stack, child)
		return err
	}
	switch source.Kind {
	case GenericOperationTypeParameter:
		if source.TypeParameter == nil || *source.TypeParameter < 0 ||
			source.CallableParameter != nil || source.Key != nil || source.Element != nil {
			return manifestError(field, "type-parameter expression is invalid")
		}
	case GenericOperationTypeCallableParameter:
		if source.CallableParameter == nil || *source.CallableParameter < 0 ||
			source.TypeParameter != nil || source.Key != nil || source.Element != nil {
			return manifestError(field, "callable-parameter expression is invalid")
		}
	case GenericOperationTypeBool, GenericOperationTypeInt:
		if source.TypeParameter != nil || source.CallableParameter != nil ||
			source.Key != nil || source.Element != nil {
			return manifestError(field, "basic type expression is invalid")
		}
	case GenericOperationTypeSlice:
		if source.TypeParameter != nil || source.CallableParameter != nil || source.Key != nil {
			return manifestError(field, "slice type expression is invalid")
		}
		return validateChild(source.Element, field+".element")
	case GenericOperationTypeMap:
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
	same := func(left, right GenericOperationTypeDocument) bool {
		return genericOperationTypeKey(left) == genericOperationTypeKey(right)
	}
	typeParameter := func(source GenericOperationTypeDocument) bool {
		return source.Kind == GenericOperationTypeParameter
	}
	booleanResult := func() bool {
		return len(operation.Results) == 1 &&
			operation.Results[0].Kind == GenericOperationTypeBool
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
			operation.Results[0].Kind != GenericOperationTypeMap ||
			operation.Results[0].Element == nil ||
			!same(operation.Parameters[0], *operation.Results[0].Element) {
			return false
		}
		return len(operation.Parameters) == 1 ||
			operation.Parameters[1].Kind == GenericOperationTypeInt
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
			operation.Parameters[0].Kind == GenericOperationTypeCallableParameter &&
			len(operation.Results) == 2 &&
			typeParameter(operation.Results[0]) &&
			operation.Results[1].Kind == GenericOperationTypeBool
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
		result.WriteString(genericOperationTypeKey(parameter))
	}
	result.WriteString("|r")
	for _, selected := range operation.Results {
		result.WriteByte(':')
		result.WriteString(genericOperationTypeKey(selected))
	}
	return result.String()
}

func genericOperationTypeKey(source GenericOperationTypeDocument) string {
	switch source.Kind {
	case GenericOperationTypeParameter:
		if source.TypeParameter == nil {
			return "parameter(?)"
		}
		return "parameter(" + strconv.Itoa(*source.TypeParameter) + ")"
	case GenericOperationTypeCallableParameter:
		if source.CallableParameter == nil {
			return "callable-parameter(?)"
		}
		return "callable-parameter(" + strconv.Itoa(*source.CallableParameter) + ")"
	case GenericOperationTypeBool, GenericOperationTypeInt:
		return string(source.Kind)
	case GenericOperationTypeSlice:
		if source.Element == nil {
			return "slice(?)"
		}
		return "slice(" + genericOperationTypeKey(*source.Element) + ")"
	case GenericOperationTypeMap:
		if source.Key == nil || source.Element == nil {
			return "map(?,?)"
		}
		return "map(" + genericOperationTypeKey(*source.Key) + "," +
			genericOperationTypeKey(*source.Element) + ")"
	default:
		return "invalid"
	}
}
