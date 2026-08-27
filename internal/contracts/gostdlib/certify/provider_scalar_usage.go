package certify

import (
	"fmt"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyProviderScalarContract(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	requirements runtimecontract.Requirements,
) error {
	providerScalarPath := providerScalarSourcePath(requirements)
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(providerScalarPath),
	))
	if err != nil {
		return err
	}
	expected := make(
		map[string]runtimecontract.PrimitiveCarrier,
		len(requirements.PrimitiveAliases()),
	)
	for _, alias := range requirements.PrimitiveAliases() {
		expected[alias.Export()] = alias.ProviderCarrier()
	}
	seen := make(map[string]struct{}, len(exports))
	for _, target := range exports {
		if _, duplicate := seen[target.Name()]; duplicate {
			return certifyError(
				"verify provider scalars",
				target.Name(),
				"export is duplicated",
			)
		}
		seen[target.Name()] = struct{}{}
		carrier, ok := expected[target.Name()]
		if !ok {
			return certifyError(
				"verify provider scalars",
				target.Name(),
				"export has no runtime-contract owner",
			)
		}
		if target.TypeParameterCount() != 0 ||
			target.DeclaredTypeString() != carrier.String() {
			return certifyError(
				"verify provider scalars",
				target.Name(),
				fmt.Sprintf(
					"declared type is %q, want %q",
					target.DeclaredTypeString(),
					carrier,
				),
			)
		}
		delete(expected, target.Name())
	}
	if len(expected) != 0 || len(exports) != len(seen) ||
		len(exports) != len(requirements.PrimitiveAliases()) {
		return certifyError(
			"verify provider scalars",
			providerScalarPath,
			fmt.Sprintf(
				"provider exports %d scalars with %d missing contract aliases",
				len(exports),
				len(expected),
			),
		)
	}
	return nil
}

func verifyProviderPointerContract(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	requirements runtimecontract.Requirements,
) error {
	pointerPath := providerPointerSourcePath(requirements)
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(pointerPath),
	))
	if err != nil {
		return err
	}
	if len(exports) != 2 {
		return certifyError(
			"verify provider pointer",
			pointerPath,
			fmt.Sprintf("provider exports %d declarations, want 2", len(exports)),
		)
	}
	seenType := false
	seenFactory := false
	for _, target := range exports {
		switch target.Name() {
		case "ProviderPointer":
			value, ok := target.TypeMember("value")
			if target.TypeParameterCount() != 1 ||
				target.DeclaredTypeString() != "ProviderPointer<T>" ||
				!ok || !value.Visible() || value.TypeString() != "T" ||
				len(target.TypeMembers()) != 1 {
				return certifyError(
					"verify provider pointer",
					target.Name(),
					"pointer type must expose exactly one mutable T-valued member named value",
				)
			}
			seenType = true
		case "providerPointer":
			parameterCount, countErr := project.CallableParameterCount(target)
			if countErr != nil {
				return countErr
			}
			parameterType, parameterErr := project.CallableParameterTypeString(target, 0)
			if parameterErr != nil {
				return parameterErr
			}
			resultType, resultErr := project.CallableReturnTypeString(target)
			if resultErr != nil {
				return resultErr
			}
			if target.TypeParameterCount() != 1 || parameterCount != 1 ||
				parameterType != "T" || resultType != "ProviderPointer<T>" {
				return certifyError(
					"verify provider pointer",
					target.Name(),
					"pointer factory must have signature <T>(value: T) => ProviderPointer<T>",
				)
			}
			seenFactory = true
		default:
			return certifyError(
				"verify provider pointer",
				target.Name(),
				"export has no runtime-contract owner",
			)
		}
	}
	if !seenType || !seenFactory {
		return certifyError(
			"verify provider pointer",
			pointerPath,
			"pointer type or factory is absent",
		)
	}
	return nil
}

func providerScalarSourcePath(
	requirements runtimecontract.Requirements,
) string {
	module := strings.TrimPrefix(requirements.ProviderScalarModule(), "./")
	return "src/" + strings.TrimSuffix(module, ".js") + ".ts"
}

func providerPointerSourcePath(
	requirements runtimecontract.Requirements,
) string {
	module := strings.TrimPrefix(requirements.ProviderPointerModule(), "./")
	return "src/" + strings.TrimSuffix(module, ".js") + ".ts"
}

func providerScalarAliasPaths(
	config resolvedConfig,
	requirements runtimecontract.Requirements,
) map[string]string {
	sourcePath := filepath.Join(
		config.providerRoot,
		filepath.FromSlash(providerScalarSourcePath(requirements)),
	)
	result := make(
		map[string]string,
		len(requirements.PrimitiveAliases()),
	)
	for _, alias := range requirements.PrimitiveAliases() {
		result[alias.Export()] = sourcePath
	}
	return result
}

func verifyExportSourceCallableScalars(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
	aliases map[string]string,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return missingSourceScalarSignature(evidence)
	}
	actual, err := project.CallableScalarAliases(target, aliases)
	if err != nil {
		return certifyError(
			"verify source callable scalars",
			evidence.contract.Identity(),
			err.Error(),
		)
	}
	return verifyCallableScalarAliases(
		evidence.contract.Identity(),
		signature,
		gostdlib.AccessExport,
		actual,
	)
}

func verifyMethodSourceCallableScalars(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectMember,
	access gostdlib.AccessKind,
	aliases map[string]string,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return missingSourceScalarSignature(evidence)
	}
	actual, err := project.CallableScalarAliases(target, aliases)
	if err != nil {
		return certifyError(
			"verify source callable scalars",
			evidence.contract.Identity(),
			err.Error(),
		)
	}
	return verifyCallableScalarAliases(
		evidence.contract.Identity(),
		signature,
		access,
		actual,
	)
}

func verifyProviderStructFieldScalars(
	project *tsgo.ProjectInspection,
	typeName *types.TypeName,
	field *types.Var,
	target tsgo.ProjectMember,
	aliases map[string]string,
) error {
	if project == nil || typeName == nil || field == nil {
		return certifyError(
			"verify provider struct-field scalars",
			"",
			"source or target field evidence is absent",
		)
	}
	actual, err := project.MemberScalarAliases(target, aliases)
	if err != nil {
		return certifyError(
			"verify provider struct-field scalars",
			typeName.Name()+"."+field.Name(),
			err.Error(),
		)
	}
	return verifyProviderStructFieldScalarAliases(
		typeName.Name()+"."+field.Name(),
		field.Type(),
		actual,
	)
}

func verifyProviderStructFieldScalarAliases(
	identity string,
	source types.Type,
	actual []string,
) error {
	expected := sourceScalarAliases(source)
	if !slices.Equal(actual, expected) {
		return certifyError(
			"verify provider struct-field scalars",
			identity,
			fmt.Sprintf(
				"target scalar aliases are %v, want %v",
				actual,
				expected,
			),
		)
	}
	return nil
}

func missingSourceScalarSignature(evidence goObject) error {
	return certifyError(
		"verify source callable scalars",
		evidence.contract.Identity(),
		"selected Go callable signature is absent",
	)
}

func verifyCallableScalarAliases(
	identity string,
	signature *types.Signature,
	access gostdlib.AccessKind,
	actual tsgo.ProjectCallableScalarAliases,
) error {
	expected, err := sourceCallableScalarAliases(signature, access)
	if err != nil {
		return certifyError(
			"verify source callable scalars",
			identity,
			err.Error(),
		)
	}
	if len(actual.Parameters) != len(expected.Parameters) {
		return certifyError(
			"verify source callable scalars",
			identity,
			fmt.Sprintf(
				"target has %d parameter scalar lists, selected Go shape requires %d",
				len(actual.Parameters),
				len(expected.Parameters),
			),
		)
	}
	for index := range expected.Parameters {
		if !slices.Equal(actual.Parameters[index], expected.Parameters[index]) {
			return certifyError(
				"verify source callable scalars",
				identity,
				fmt.Sprintf(
					"parameter %d scalar aliases are %v, want %v",
					index,
					actual.Parameters[index],
					expected.Parameters[index],
				),
			)
		}
	}
	if !slices.Equal(actual.Results, expected.Results) {
		return certifyError(
			"verify source callable scalars",
			identity,
			fmt.Sprintf(
				"result scalar aliases are %v, want %v",
				actual.Results,
				expected.Results,
			),
		)
	}
	return nil
}

func sourceCallableScalarAliases(
	signature *types.Signature,
	access gostdlib.AccessKind,
) (tsgo.ProjectCallableScalarAliases, error) {
	if signature == nil {
		return tsgo.ProjectCallableScalarAliases{}, fmt.Errorf(
			"selected Go signature is absent",
		)
	}
	result := tsgo.ProjectCallableScalarAliases{}
	switch access {
	case gostdlib.AccessExport, gostdlib.AccessInstanceMethod:
	case gostdlib.AccessStaticMethod:
		if signature.Recv() == nil {
			return tsgo.ProjectCallableScalarAliases{}, fmt.Errorf(
				"static method receiver is absent",
			)
		}
		result.Parameters = append(
			result.Parameters,
			sourceScalarAliases(signature.Recv().Type()),
		)
	default:
		return tsgo.ProjectCallableScalarAliases{}, fmt.Errorf(
			"callable access %q is unsupported",
			access,
		)
	}
	for index := range signature.Params().Len() {
		result.Parameters = append(
			result.Parameters,
			sourceScalarAliases(signature.Params().At(index).Type()),
		)
	}
	for index := range signature.Results().Len() {
		result.Results = append(
			result.Results,
			sourceScalarAliases(signature.Results().At(index).Type())...,
		)
	}
	return result, nil
}

func sourceScalarAliases(source types.Type) []string {
	switch selected := source.(type) {
	case *types.Basic:
		if alias := sourceBasicScalarAlias(selected); alias != "" {
			return []string{alias}
		}
	case *types.Pointer:
		return sourceScalarAliases(selected.Elem())
	case *types.Slice:
		return sourceScalarAliases(selected.Elem())
	case *types.Array:
		return sourceScalarAliases(selected.Elem())
	case *types.Map:
		return append(
			sourceScalarAliases(selected.Key()),
			sourceScalarAliases(selected.Elem())...,
		)
	case *types.Chan:
		return sourceScalarAliases(selected.Elem())
	case *types.Tuple:
		var result []string
		for index := range selected.Len() {
			result = append(
				result,
				sourceScalarAliases(selected.At(index).Type())...,
			)
		}
		return result
	case *types.Signature:
		var result []string
		if selected.Params() != nil {
			result = append(result, sourceScalarAliases(selected.Params())...)
		}
		if selected.Results() != nil {
			result = append(result, sourceScalarAliases(selected.Results())...)
		}
		return result
	case *types.Named:
		var result []string
		for index := range selected.TypeArgs().Len() {
			result = append(
				result,
				sourceScalarAliases(selected.TypeArgs().At(index))...,
			)
		}
		return result
	case *types.Alias:
		if selected.Obj() == nil || selected.Obj().Pkg() == nil {
			return sourceScalarAliases(types.Unalias(selected))
		}
	}
	return nil
}

func sourceBasicScalarAlias(source *types.Basic) string {
	switch source.Kind() {
	case types.Bool:
		return "bool"
	case types.Int:
		return "int"
	case types.Int8:
		return "int8"
	case types.Int16:
		return "int16"
	case types.Int32:
		return "int32"
	case types.Int64:
		return "int64"
	case types.Uint:
		return "uint"
	case types.Uint8:
		return "uint8"
	case types.Uint16:
		return "uint16"
	case types.Uint32:
		return "uint32"
	case types.Uint64:
		return "uint64"
	case types.Uintptr:
		return "uintptr"
	case types.Float32:
		return "float32"
	case types.Float64:
		return "float64"
	case types.String:
		return "gostring"
	default:
		return ""
	}
}
