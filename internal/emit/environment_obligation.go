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

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/load"
)

type EnvironmentObligationKind uint8

const (
	EnvironmentObligationInvalid EnvironmentObligationKind = iota
	EnvironmentObligationDeclaration
	EnvironmentObligationState
	EnvironmentObligationConstantProjection
)

type EnvironmentObjectKind = environmentidentity.ObjectKind

const (
	EnvironmentObjectInvalid  = environmentidentity.ObjectInvalid
	EnvironmentObjectConstant = environmentidentity.ObjectConstant
	EnvironmentObjectType     = environmentidentity.ObjectType
	EnvironmentObjectVariable = environmentidentity.ObjectVariable
	EnvironmentObjectFunction = environmentidentity.ObjectFunction
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
	description, err := environmentidentity.Describe(object)
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
	objectKind := description.Kind()
	receiver := description.Receiver()
	signature := description.Signature()
	value := description.Value()
	requirementKeys, err := environmentRequirementKeys(object, requirements)
	if err != nil {
		return EnvironmentObligation{}, err
	}
	identity := description.Identity()
	if kind == EnvironmentObligationConstantProjection {
		selected, valid := api.ConstantProjectionType(projection)
		if !valid {
			return EnvironmentObligation{}, &ScheduleError{
				Object: object.Name(),
				Reason: "environment constant projection is invalid",
			}
		}
		identity += "|projection=" + environmentidentity.StableTypeString(selected)
		signature += "|projection=" + environmentidentity.StableTypeString(selected)
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
		case api.DeclarationRequirementGenericRepresentation:
			owner, parameter, facet, ok :=
				requirement.GenericRepresentation()
			index, indexed :=
				api.GenericDeclarationParameterIndex(owner, parameter)
			if !ok || !indexed {
				return nil, environmentRequirementError(object, requirement)
			}
			detail = strconv.Itoa(index) + ":" + facet.String()
		case api.DeclarationRequirementTypeRepresentation:
			owner, artifact, facet, ok := requirement.TypeRepresentation()
			if !ok || owner != object || artifact != nil {
				return nil, environmentRequirementError(object, requirement)
			}
			detail = facet.String()
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
