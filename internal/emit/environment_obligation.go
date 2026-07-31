package emit

import (
	"fmt"
	"go/types"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	"github.com/tsoniclang/gotots/internal/load"
)

type EnvironmentObligationKind uint8

const (
	EnvironmentObligationInvalid EnvironmentObligationKind = iota
	EnvironmentObligationDeclaration
	EnvironmentObligationState
	EnvironmentObligationConstantProjection
	EnvironmentObligationBuiltin
)

type EnvironmentObjectKind uint8

const (
	EnvironmentObjectInvalid EnvironmentObjectKind = iota
	EnvironmentObjectConstant
	EnvironmentObjectType
	EnvironmentObjectVariable
	EnvironmentObjectFunction
	EnvironmentObjectBuiltin
)

type EnvironmentObligation struct {
	kind              EnvironmentObligationKind
	objectKind        EnvironmentObjectKind
	packageKind       load.PackageKind
	packagePath       string
	contractKey       string
	identity          string
	name              string
	receiver          string
	sourceSignature   string
	sourceValue       string
	sourceLocation    string
	targetName        string
	targetFingerprint string
	requirements      []string
}

func (o EnvironmentObligation) Kind() EnvironmentObligationKind {
	return o.kind
}

func (o EnvironmentObligation) ObjectKind() EnvironmentObjectKind {
	return o.objectKind
}

func (o EnvironmentObligation) PackageKind() load.PackageKind {
	return o.packageKind
}

func (o EnvironmentObligation) PackagePath() string {
	return o.packagePath
}

func (o EnvironmentObligation) ContractKey() string {
	return o.contractKey
}

func (o EnvironmentObligation) Identity() string {
	return o.identity
}

func (o EnvironmentObligation) Name() string {
	return o.name
}

func (o EnvironmentObligation) Receiver() string {
	return o.receiver
}

func (o EnvironmentObligation) SourceSignature() string {
	return o.sourceSignature
}

func (o EnvironmentObligation) SourceValue() string {
	return o.sourceValue
}

func (o EnvironmentObligation) SourceLocation() string {
	return o.sourceLocation
}

func (o EnvironmentObligation) TargetName() string {
	return o.targetName
}

func (o EnvironmentObligation) TargetFingerprint() string {
	return o.targetFingerprint
}

func (o EnvironmentObligation) Requirements() []string {
	return slices.Clone(o.requirements)
}

func (s *programSession) environmentObligations() (
	[]EnvironmentObligation,
	error,
) {
	builders := make(
		[]*environmentContractBuilder,
		0,
		len(s.environmentBuilders),
	)
	for _, builder := range s.environmentBuilders {
		builders = append(builders, builder)
	}
	sort.Slice(builders, func(left, right int) bool {
		return builders[left].sourcePackage.Path() <
			builders[right].sourcePackage.Path()
	})
	var obligations []EnvironmentObligation
	for _, builder := range builders {
		for _, declaration := range builder.declarations {
			var requirements []api.DeclarationRequirement
			if _, constant := declaration.object.(*types.Const); !constant {
				var err error
				requirements, err = s.environmentDeclarationRequirements(
					declaration.object,
					s.requirements.appliedFor(
						api.MustSourceArtifactOwner(declaration.object),
					),
				)
				if err != nil {
					return nil, err
				}
			}
			obligation, err := buildEnvironmentObligation(
				builder,
				EnvironmentObligationDeclaration,
				declaration.object,
				declaration.name,
				declaration.contract,
				requirements,
				types.Invalid,
			)
			if err != nil {
				return nil, err
			}
			obligations = append(obligations, obligation)
		}
		for variable, state := range builder.stateFields {
			obligation, err := buildEnvironmentObligation(
				builder,
				EnvironmentObligationState,
				variable,
				variable.Name(),
				state.contract,
				nil,
				types.Invalid,
			)
			if err != nil {
				return nil, err
			}
			obligations = append(obligations, obligation)
		}
		for _, projection := range builder.projections {
			obligation, err := buildEnvironmentObligation(
				builder,
				EnvironmentObligationConstantProjection,
				projection.source,
				projection.name,
				projection.contract,
				nil,
				projection.projection,
			)
			if err != nil {
				return nil, err
			}
			obligations = append(obligations, obligation)
		}
		for builtin, target := range builder.builtins {
			if !target.emitted || len(target.signatures) == 0 {
				continue
			}
			obligation, err := buildEnvironmentBuiltinObligation(
				builder,
				builtin,
				target,
			)
			if err != nil {
				return nil, err
			}
			obligations = append(obligations, obligation)
		}
	}
	sort.Slice(obligations, func(left, right int) bool {
		if obligations[left].identity != obligations[right].identity {
			return obligations[left].identity < obligations[right].identity
		}
		if obligations[left].kind != obligations[right].kind {
			return obligations[left].kind < obligations[right].kind
		}
		return obligations[left].targetName < obligations[right].targetName
	})
	return obligations, nil
}

func buildEnvironmentObligation(
	builder *environmentContractBuilder,
	kind EnvironmentObligationKind,
	object types.Object,
	targetName string,
	contract artifactstate.Contract,
	requirements []api.DeclarationRequirement,
	projection types.BasicKind,
) (EnvironmentObligation, error) {
	if builder == nil ||
		builder.sourcePackage == nil ||
		object == nil ||
		targetName == "" {
		return EnvironmentObligation{}, &ScheduleError{
			Reason: "environment obligation identity is invalid",
		}
	}
	objectKind, err := environmentObjectKind(object)
	if err != nil {
		return EnvironmentObligation{}, err
	}
	fingerprint, ok := contract.Fingerprint()
	if !ok {
		return EnvironmentObligation{}, &ScheduleError{
			Object: object.Name(),
			Reason: "environment target contract has no canonical fingerprint",
		}
	}
	receiver := environmentReceiver(object)
	signature := environmentSourceSignature(object)
	value := environmentSourceValue(object)
	requirementKeys, err := environmentRequirementKeys(object, requirements)
	if err != nil {
		return EnvironmentObligation{}, err
	}
	identity := environmentObjectIdentity(
		builder.sourcePackage.Path(),
		objectKind,
		receiver,
		object.Name(),
	)
	if kind == EnvironmentObligationConstantProjection {
		selected, valid := api.ConstantProjectionType(projection)
		if !valid {
			return EnvironmentObligation{}, &ScheduleError{
				Object: object.Name(),
				Reason: "environment constant projection is invalid",
			}
		}
		identity += "|projection=" + emitordering.StableTypeString(selected)
		signature += "|projection=" + emitordering.StableTypeString(selected)
	}
	return EnvironmentObligation{
		kind:              kind,
		objectKind:        objectKind,
		packageKind:       builder.sourcePackage.Kind(),
		packagePath:       builder.sourcePackage.Path(),
		contractKey:       environmentContractKey(builder.sourcePackage),
		identity:          identity,
		name:              object.Name(),
		receiver:          receiver,
		sourceSignature:   signature,
		sourceValue:       value,
		sourceLocation:    environmentSourceLocation(builder.sourcePackage, object),
		targetName:        targetName,
		targetFingerprint: fingerprint,
		requirements:      requirementKeys,
	}, nil
}

func buildEnvironmentBuiltinObligation(
	builder *environmentContractBuilder,
	builtin *types.Builtin,
	target environmentBuiltin,
) (EnvironmentObligation, error) {
	fingerprint, ok := target.contract.Fingerprint()
	if !ok || builtin == nil || len(target.signatures) == 0 {
		return EnvironmentObligation{}, &ScheduleError{
			Reason: "environment builtin obligation is invalid",
		}
	}
	signatures := make([]string, len(target.signatures))
	for index, signature := range target.signatures {
		signatures[index] = emitordering.StableTypeString(signature)
	}
	sort.Strings(signatures)
	identity := environmentObjectIdentity(
		builder.sourcePackage.Path(),
		EnvironmentObjectBuiltin,
		"",
		builtin.Name(),
	)
	return EnvironmentObligation{
		kind:              EnvironmentObligationBuiltin,
		objectKind:        EnvironmentObjectBuiltin,
		packageKind:       builder.sourcePackage.Kind(),
		packagePath:       builder.sourcePackage.Path(),
		contractKey:       environmentContractKey(builder.sourcePackage),
		identity:          identity,
		name:              builtin.Name(),
		sourceSignature:   strings.Join(signatures, "|"),
		targetName:        builtin.Name(),
		targetFingerprint: fingerprint,
	}, nil
}

func environmentObjectKind(object types.Object) (EnvironmentObjectKind, error) {
	switch object.(type) {
	case *types.Const:
		return EnvironmentObjectConstant, nil
	case *types.TypeName:
		return EnvironmentObjectType, nil
	case *types.Var:
		return EnvironmentObjectVariable, nil
	case *types.Func:
		return EnvironmentObjectFunction, nil
	case *types.Builtin:
		return EnvironmentObjectBuiltin, nil
	default:
		return EnvironmentObjectInvalid, &ScheduleError{
			Object: object.Name(),
			Reason: "environment object kind is unsupported",
		}
	}
}

func environmentObjectIdentity(
	packagePath string,
	kind EnvironmentObjectKind,
	receiver string,
	name string,
) string {
	return packagePath + "|kind=" + strconv.Itoa(int(kind)) +
		"|receiver=" + receiver +
		"|name=" + name
}

func environmentReceiver(object types.Object) string {
	function, ok := object.(*types.Func)
	if !ok {
		return ""
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return ""
	}
	return emitordering.StableTypeString(signature.Recv().Type())
}

func environmentSourceSignature(object types.Object) string {
	switch selected := object.(type) {
	case *types.TypeName:
		return "defined=" + emitordering.StableTypeString(selected.Type()) +
			"|underlying=" +
			emitordering.StableTypeString(selected.Type().Underlying())
	case *types.Func:
		signature, _ := selected.Type().(*types.Signature)
		return emitordering.StableTypeString(selected.Type()) +
			"|params=" + tupleNames(signature.Params()) +
			"|results=" + tupleNames(signature.Results())
	default:
		if object.Type() == nil {
			return ""
		}
		return emitordering.StableTypeString(object.Type())
	}
}

func environmentSourceValue(object types.Object) string {
	selected, ok := object.(*types.Const)
	if !ok || selected.Val() == nil {
		return ""
	}
	return selected.Val().ExactString()
}

func tupleNames(tuple *types.Tuple) string {
	if tuple == nil {
		return ""
	}
	names := make([]string, tuple.Len())
	for index := range tuple.Len() {
		names[index] = tuple.At(index).Name()
	}
	return strings.Join(names, ",")
}

func environmentContractKey(sourcePackage *load.Package) string {
	if sourcePackage.Kind() == load.PackageStandardLibraryContract {
		return sourcePackage.ToolchainKey()
	}
	return sourcePackage.ExternalContractKey()
}

func environmentSourceLocation(
	sourcePackage *load.Package,
	object types.Object,
) string {
	if sourcePackage == nil ||
		sourcePackage.FileSet() == nil ||
		!object.Pos().IsValid() {
		return ""
	}
	position := sourcePackage.FileSet().PositionFor(object.Pos(), false)
	if !position.IsValid() {
		return ""
	}
	sourcePath := filepath.ToSlash(position.Filename)
	if sourcePackage.Kind() == load.PackageStandardLibraryContract {
		root := filepath.Join(runtime.GOROOT(), "src")
		if relative, err := filepath.Rel(root, position.Filename); err == nil &&
			relative != "." &&
			!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			sourcePath = filepath.ToSlash(relative)
		}
	}
	return sourcePath + ":" +
		strconv.Itoa(position.Line) + ":" +
		strconv.Itoa(position.Column)
}

func environmentRequirementKeys(
	object types.Object,
	requirements []api.DeclarationRequirement,
) ([]string, error) {
	keys := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		var detail string
		switch requirement.Kind() {
		case api.DeclarationRequirementGenericCallableProfile:
			profile, ok := requirement.GenericCallableProfile()
			if !ok {
				return nil, environmentRequirementError(object, requirement)
			}
			detail = profile.Key()
		case api.DeclarationRequirementGenericRepresentation:
			owner, parameter, facet, ok :=
				requirement.GenericRepresentation()
			index, indexed :=
				api.GenericDeclarationParameterIndex(owner, parameter)
			if !ok || !indexed {
				return nil, environmentRequirementError(object, requirement)
			}
			detail = strconv.Itoa(index) + ":" + facet.String()
		case api.DeclarationRequirementCallableControl:
			_, enclosing, callable, control, ok :=
				requirement.CallableControl()
			if !ok ||
				enclosing != nil ||
				callable != nil ||
				control != api.CallableControlRecovery {
				return nil, environmentRequirementError(object, requirement)
			}
			detail = "recovery"
		default:
			return nil, environmentRequirementError(object, requirement)
		}
		keys = append(
			keys,
			strconv.Itoa(int(requirement.Kind()))+":"+detail,
		)
	}
	sort.Strings(keys)
	return keys, nil
}

func environmentRequirementError(
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	name := ""
	if object != nil {
		name = object.Name()
	}
	return &ScheduleError{
		Object: name,
		Reason: fmt.Sprintf(
			"environment obligation requirement kind %d is not canonical",
			requirement.Kind(),
		),
	}
}
