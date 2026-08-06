package gostdlib

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const providerProtocolSchema = "provider-protocol/v1"

type ProviderProtocolMethodDocument struct {
	Name       string                 `json:"name"`
	Parameters []ContractTypeDocument `json:"parameters,omitempty"`
	Results    []ContractTypeDocument `json:"results,omitempty"`
}

type ProviderProtocolInterfaceDocument struct {
	Methods []ProviderProtocolMethodDocument `json:"methods"`
}

func CanonicalProviderProtocolInterface(
	source ProviderProtocolInterfaceDocument,
) (ProviderProtocolInterfaceDocument, error) {
	result := cloneProviderProtocolInterface(source)
	sort.Slice(result.Methods, func(left, right int) bool {
		return result.Methods[left].Name < result.Methods[right].Name
	})
	if err := validateProviderProtocolInterface(
		result,
		"providerProtocol",
	); err != nil {
		return ProviderProtocolInterfaceDocument{}, err
	}
	return result, nil
}

func BuildProviderProtocolInterfaceIdentity(
	source ProviderProtocolInterfaceDocument,
) (string, error) {
	canonical, err := CanonicalProviderProtocolInterface(source)
	if err != nil {
		return "", err
	}
	var payload strings.Builder
	payload.WriteString(providerProtocolSchema)
	payload.WriteByte('\n')
	for _, method := range canonical.Methods {
		payload.WriteString(providerProtocolMethodKey(method))
		payload.WriteByte('\n')
	}
	digest := sha256.Sum256([]byte(payload.String()))
	return "go:provider-protocol|" + hex.EncodeToString(digest[:]), nil
}

func ProviderProtocolMethodSource(
	interfaceIdentity string,
	method ProviderProtocolMethodDocument,
) (string, string, error) {
	if interfaceIdentity == "" {
		return "", "", manifestError(
			"providerProtocol.identity",
			"value is empty",
		)
	}
	canonical, err := CanonicalProviderProtocolInterface(
		ProviderProtocolInterfaceDocument{
			Methods: []ProviderProtocolMethodDocument{method},
		},
	)
	if err != nil {
		return "", "", err
	}
	selected := canonical.Methods[0]
	return interfaceIdentity + "|method=" + selected.Name,
		providerProtocolMethodSignature(selected), nil
}

func ProviderProtocolMethod(
	source ProviderProtocolInterfaceDocument,
	name string,
) (ProviderProtocolMethodDocument, bool) {
	canonical, err := CanonicalProviderProtocolInterface(source)
	if err != nil {
		return ProviderProtocolMethodDocument{}, false
	}
	index, found := slices.BinarySearchFunc(
		canonical.Methods,
		name,
		func(method ProviderProtocolMethodDocument, selected string) int {
			return strings.Compare(method.Name, selected)
		},
	)
	if !found {
		return ProviderProtocolMethodDocument{}, false
	}
	return cloneProviderProtocolMethod(canonical.Methods[index]), true
}

func ProviderProtocolCallableParameters(
	source ProviderProtocolInterfaceDocument,
) ([]int, error) {
	canonical, err := CanonicalProviderProtocolInterface(source)
	if err != nil {
		return nil, err
	}
	selected := make(map[int]struct{})
	var collect func(ContractTypeDocument)
	collect = func(reference ContractTypeDocument) {
		if reference.Kind == ContractTypeCallableParameter &&
			reference.CallableParameter != nil {
			selected[*reference.CallableParameter] = struct{}{}
		}
		for _, child := range []*ContractTypeDocument{reference.Key, reference.Element} {
			if child != nil {
				collect(*child)
			}
		}
	}
	for _, method := range canonical.Methods {
		for _, reference := range method.Parameters {
			collect(reference)
		}
		for _, reference := range method.Results {
			collect(reference)
		}
	}
	result := make([]int, 0, len(selected))
	for index := range selected {
		result = append(result, index)
	}
	sort.Ints(result)
	return result, nil
}

func validateProviderProtocolInterface(
	source ProviderProtocolInterfaceDocument,
	field string,
) error {
	if len(source.Methods) == 0 {
		return manifestError(field+".methods", "set is empty")
	}
	previous := ""
	for index, method := range source.Methods {
		selected := field + ".methods[" + strconv.Itoa(index) + "]"
		if !isExportedProviderProtocolMethod(method.Name) || method.Name <= previous {
			return manifestError(
				field+".methods",
				"names are unexported, duplicated, or not strictly ordered",
			)
		}
		previous = method.Name
		for parameterIndex, parameter := range method.Parameters {
			if err := validateContractType(
				parameter,
				selected+".parameters["+strconv.Itoa(parameterIndex)+"]",
				0,
				make(map[*ContractTypeDocument]struct{}),
			); err != nil {
				return err
			}
		}
		for resultIndex, result := range method.Results {
			if err := validateContractType(
				result,
				selected+".results["+strconv.Itoa(resultIndex)+"]",
				0,
				make(map[*ContractTypeDocument]struct{}),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func isExportedProviderProtocolMethod(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func cloneProviderProtocolInterface(
	source ProviderProtocolInterfaceDocument,
) ProviderProtocolInterfaceDocument {
	result := ProviderProtocolInterfaceDocument{
		Methods: make([]ProviderProtocolMethodDocument, len(source.Methods)),
	}
	for index, method := range source.Methods {
		result.Methods[index] = cloneProviderProtocolMethod(method)
	}
	return result
}

func cloneProviderProtocolMethod(
	source ProviderProtocolMethodDocument,
) ProviderProtocolMethodDocument {
	result := source
	result.Parameters = cloneContractTypes(source.Parameters)
	result.Results = cloneContractTypes(source.Results)
	return result
}

func sameProviderProtocolInterface(
	left ProviderProtocolInterfaceDocument,
	right ProviderProtocolInterfaceDocument,
) bool {
	leftKey, leftErr := BuildProviderProtocolInterfaceIdentity(left)
	rightKey, rightErr := BuildProviderProtocolInterfaceIdentity(right)
	return leftErr == nil && rightErr == nil && leftKey == rightKey
}

func providerProtocolMethodKey(method ProviderProtocolMethodDocument) string {
	return method.Name + providerProtocolMethodSignature(method)
}

func providerProtocolMethodSignature(method ProviderProtocolMethodDocument) string {
	var result strings.Builder
	result.WriteString("func(")
	for index, parameter := range method.Parameters {
		if index != 0 {
			result.WriteByte(',')
		}
		result.WriteString(contractTypeKey(parameter))
	}
	result.WriteString(")(")
	for index, selected := range method.Results {
		if index != 0 {
			result.WriteByte(',')
		}
		result.WriteString(contractTypeKey(selected))
	}
	result.WriteByte(')')
	return result.String()
}
