package gostdlib

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"
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

func ResolveProviderProtocolInterface(
	source ProviderProtocolInterfaceDocument,
	owner *types.Signature,
) (*types.Interface, error) {
	if owner == nil {
		return nil, manifestError(
			"providerProtocol.owner",
			"callable signature is nil",
		)
	}
	canonical, err := CanonicalProviderProtocolInterface(source)
	if err != nil {
		return nil, err
	}
	typeParameters := callableTypeParameters(owner)
	resolveType := func(reference ContractTypeDocument) (types.Type, error) {
		return resolveContractType(reference, owner, typeParameters)
	}
	methods := make([]*types.Func, 0, len(canonical.Methods))
	for _, method := range canonical.Methods {
		parameters, err := resolveContractTuple(method.Parameters, resolveType)
		if err != nil {
			return nil, err
		}
		results, err := resolveContractTuple(method.Results, resolveType)
		if err != nil {
			return nil, err
		}
		methods = append(methods, types.NewFunc(
			token.NoPos,
			nil,
			method.Name,
			types.NewSignatureType(nil, nil, nil, parameters, results, false),
		))
	}
	return types.NewInterfaceType(methods, nil).Complete(), nil
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
		if !ast.IsExported(method.Name) || method.Name <= previous {
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

func callableTypeParameters(signature *types.Signature) []*types.TypeParam {
	result := make(
		[]*types.TypeParam,
		0,
		signature.RecvTypeParams().Len()+signature.TypeParams().Len(),
	)
	for index := range signature.RecvTypeParams().Len() {
		result = append(result, signature.RecvTypeParams().At(index))
	}
	for index := range signature.TypeParams().Len() {
		result = append(result, signature.TypeParams().At(index))
	}
	return result
}

func resolveContractType(
	reference ContractTypeDocument,
	owner *types.Signature,
	typeParameters []*types.TypeParam,
) (types.Type, error) {
	switch reference.Kind {
	case ContractTypeParameter:
		if reference.TypeParameter != nil &&
			*reference.TypeParameter >= 0 &&
			*reference.TypeParameter < len(typeParameters) {
			return typeParameters[*reference.TypeParameter], nil
		}
	case ContractTypeCallableParameter:
		if reference.CallableParameter != nil && owner.Params() != nil &&
			*reference.CallableParameter >= 0 &&
			*reference.CallableParameter < owner.Params().Len() {
			return owner.Params().At(*reference.CallableParameter).Type(), nil
		}
	case ContractTypeBool:
		return types.Typ[types.Bool], nil
	case ContractTypeInt:
		return types.Typ[types.Int], nil
	case ContractTypeSlice:
		if reference.Element != nil {
			element, err := resolveContractType(*reference.Element, owner, typeParameters)
			if err != nil {
				return nil, err
			}
			return types.NewSlice(element), nil
		}
	case ContractTypeMap:
		if reference.Key != nil && reference.Element != nil {
			key, err := resolveContractType(*reference.Key, owner, typeParameters)
			if err != nil {
				return nil, err
			}
			element, err := resolveContractType(*reference.Element, owner, typeParameters)
			if err != nil {
				return nil, err
			}
			return types.NewMap(key, element), nil
		}
	}
	return nil, manifestError(
		"providerProtocol.type",
		"type expression is outside its callable declaration",
	)
}

func resolveContractTuple(
	references []ContractTypeDocument,
	resolve func(ContractTypeDocument) (types.Type, error),
) (*types.Tuple, error) {
	variables := make([]*types.Var, 0, len(references))
	for _, reference := range references {
		selected, err := resolve(reference)
		if err != nil {
			return nil, err
		}
		variables = append(variables, types.NewVar(token.NoPos, nil, "", selected))
	}
	return types.NewTuple(variables...), nil
}
