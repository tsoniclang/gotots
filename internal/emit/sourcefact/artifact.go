package sourcefact

import (
	"fmt"
	"go/types"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const artifactSchema = "gotots-go-generated-artifact-fact-v1"

func IncludeGeneratedArtifact(
	context api.Context,
	artifact *api.GeneratedArtifact,
	statements []tsgo.Statement,
	requests []api.RootRequest,
) ([]tsgo.Statement, []api.RootRequest, error) {
	fact, err := GeneratedArtifact(context, artifact, statements)
	if err != nil {
		return nil, nil, err
	}
	return append(statements, fact.Statements()...), api.CombineRequests(
		requests,
		fact.Requests(),
	), nil
}

type artifactTargetMode uint8

const (
	artifactTargetInvalid artifactTargetMode = iota
	artifactTargetType
	artifactTargetValue
	artifactTargetConstructedType
)

func GeneratedArtifact(
	context api.Context,
	artifact *api.GeneratedArtifact,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if artifact == nil || !artifact.Valid() {
		return api.StatementEmission{}, &Error{Reason: "generated artifact is invalid"}
	}
	fact, mode, kind, err := generatedArtifactContract(artifact)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := generatedArtifactTarget(
		context.Factory(),
		artifact.TargetName(),
		mode,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	arguments, err := generatedArtifactArguments(
		context.Factory(),
		artifact,
		kind,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	base, err := attribute.Apply(context, target, fact, arguments...)
	if err != nil {
		return api.StatementEmission{}, err
	}
	emissions := []api.StatementEmission{base}
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		structure, ok := artifact.StructType()
		if !ok {
			return api.StatementEmission{}, &Error{
				Subject: artifact.TargetName(),
				Reason:  "anonymous struct source is invalid",
			}
		}
		members, memberErr := generatedStructFields(
			context,
			target,
			artifact.ArtifactKey(),
			structure,
		)
		if memberErr != nil {
			return api.StatementEmission{}, memberErr
		}
		emissions = append(emissions, members)
	case api.GeneratedArtifactAnonymousInterface:
		contract, ok := artifact.InterfaceType()
		if !ok {
			return api.StatementEmission{}, &Error{
				Subject: artifact.TargetName(),
				Reason:  "anonymous interface source is invalid",
			}
		}
		members, memberErr := generatedInterfaceMethods(
			context,
			target,
			artifact.ArtifactKey(),
			contract.Complete(),
		)
		if memberErr != nil {
			return api.StatementEmission{}, memberErr
		}
		emissions = append(emissions, members)
	}
	return combine(emissions)
}

func generatedArtifactContract(
	artifact *api.GeneratedArtifact,
) (api.RuntimeSymbol, artifactTargetMode, string, error) {
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		return api.RuntimeSourceAggregateFact, artifactTargetType, "anonymous-struct", nil
	case api.GeneratedArtifactMapSpecialization:
		return api.RuntimeSourceAggregateFact, artifactTargetType, "map-specialization", nil
	case api.GeneratedArtifactInterfaceAdapter:
		return api.RuntimeSourceInterfaceFact, artifactTargetConstructedType, "interface-adapter", nil
	case api.GeneratedArtifactAnonymousInterface:
		return api.RuntimeSourceInterfaceFact, artifactTargetType, "anonymous-interface", nil
	case api.GeneratedArtifactInterfaceMethodToken:
		return api.RuntimeSourceOperationFact, artifactTargetValue, "interface-method-token", nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		return api.RuntimeSourceInterfaceFact, artifactTargetValue, "interface-dynamic-type", nil
	case api.GeneratedArtifactGenericCapability:
		return api.RuntimeSourceOperationFact, artifactTargetValue, "generic-capability", nil
	case api.GeneratedArtifactProviderInterfaceBridge:
		return api.RuntimeSourceImplementationFact, artifactTargetType, "provider-interface-bridge", nil
	case api.GeneratedArtifactProviderStatefulRepresentation:
		return api.RuntimeSourceImplementationFact, artifactTargetType, "provider-stateful-representation", nil
	case api.GeneratedArtifactDeferredCallableRegistry:
		return api.RuntimeSourceOperationFact, artifactTargetValue, "deferred-callable-registry", nil
	case api.GeneratedArtifactGenericConcretization:
		return api.RuntimeSourceCallableFact, artifactTargetValue, "generic-concretization", nil
	case api.GeneratedArtifactReflectionType:
		return api.RuntimeSourceOperationFact, artifactTargetValue, "reflection-type", nil
	default:
		return api.RuntimeInvalid, artifactTargetInvalid, "", &Error{
			Subject: artifact.TargetName(),
			Reason:  "generated artifact kind has no source-fact contract",
		}
	}
}

func generatedArtifactArguments(
	factory tsgo.Factory,
	artifact *api.GeneratedArtifact,
	kind string,
) ([]tsgo.Expression, error) {
	arguments := []tsgo.Expression{
		text(factory, artifactSchema),
		text(factory, kind),
		count(factory, int(artifact.Kind())),
		text(factory, artifact.ArtifactKey()),
		text(factory, artifact.TargetName()),
		text(factory, generatedArtifactPlacement(artifact.Placement())),
		text(factory, artifact.OutputPath()),
		text(factory, sourceTypeKind(artifact.SourceType())),
		text(factory, environmentcontract.StableTypeString(artifact.SourceType())),
	}
	switch artifact.Kind() {
	case api.GeneratedArtifactInterfaceMethodToken:
		runtime, ok := artifact.InterfaceMethodRuntime()
		if !ok {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "interface method runtime is invalid"}
		}
		arguments = append(arguments, count(factory, int(runtime)))
	case api.GeneratedArtifactGenericCapability:
		_, selection, ok := artifact.GenericCapability()
		if !ok {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "generic capability is invalid"}
		}
		arguments = append(arguments, text(factory, selection.Operation().Identifier()))
		if method, selected := selection.Method(); selected {
			contract, err := environmentcontract.Describe(method)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, text(factory, contract.Identity()))
		} else {
			arguments = append(arguments, text(factory, ""))
		}
	case api.GeneratedArtifactGenericConcretization:
		concretization, ok := artifact.GenericConcretization()
		if !ok {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "generic concretization is invalid"}
		}
		contract, err := environmentcontract.Describe(concretization.Owner())
		if err != nil {
			return nil, err
		}
		arguments = append(
			arguments,
			text(factory, contract.Identity()),
			count(factory, len(concretization.Arguments())),
		)
		for index, argument := range concretization.Arguments() {
			arguments = append(
				arguments,
				count(factory, index),
				text(factory, sourceTypeKind(argument)),
				text(factory, environmentcontract.StableTypeString(argument)),
			)
		}
	case api.GeneratedArtifactReflectionType:
		_, reflectionType, ok := artifact.ReflectionType()
		if !ok {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "reflection declaration is invalid"}
		}
		contract, err := environmentcontract.Describe(reflectionType)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, text(factory, contract.Identity()))
	case api.GeneratedArtifactProviderInterfaceBridge:
		if _, profile, selected := artifact.ProviderProfileInterfaceBridge(); selected {
			arguments = append(arguments, count(factory, len(profile)))
			for index, item := range profile {
				arguments = append(
					arguments,
					count(factory, index),
					text(factory, item.SourceIdentity()),
					text(factory, item.Export()),
					text(factory, item.TargetFingerprint()),
				)
			}
		} else {
			arguments = append(arguments, count(factory, 0))
		}
	}
	return arguments, nil
}

func generatedArtifactPlacement(placement api.GeneratedArtifactPlacement) string {
	switch placement {
	case api.GeneratedArtifactPlacementCompilation:
		return "compilation"
	case api.GeneratedArtifactPlacementLexical:
		return "lexical"
	case api.GeneratedArtifactPlacementContract:
		return "contract"
	default:
		return "invalid"
	}
}

func sourceTypeKind(sourceType types.Type) string {
	switch types.Unalias(sourceType).(type) {
	case *types.Basic:
		return "basic"
	case *types.Array:
		return "array"
	case *types.Slice:
		return "slice"
	case *types.Struct:
		return "struct"
	case *types.Pointer:
		return "pointer"
	case *types.Tuple:
		return "tuple"
	case *types.Signature:
		return "signature"
	case *types.Interface:
		return "interface"
	case *types.Map:
		return "map"
	case *types.Chan:
		return "channel"
	case *types.Named:
		return "named"
	case *types.TypeParam:
		return "type-parameter"
	default:
		return "other"
	}
}

func generatedArtifactTarget(
	factory tsgo.Factory,
	name string,
	mode artifactTargetMode,
	statements []tsgo.Statement,
) (tsgo.TypeNode, error) {
	return exactDeclarationTarget(factory, []string{name}, mode, statements)
}

func exactDeclarationTarget(
	factory tsgo.Factory,
	names []string,
	mode artifactTargetMode,
	statements []tsgo.Statement,
) (tsgo.TypeNode, error) {
	accepted := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			return nil, &Error{Reason: "source-fact target name is empty"}
		}
		accepted[name] = struct{}{}
	}
	var candidates []tsgo.TypeNode
	var declarations []string
	for _, statement := range statements {
		switch selected := statement.(type) {
		case tsgo.ClassDeclaration:
			name := selected.Name().Text()
			declarations = append(declarations, "class "+name)
			if _, ok := accepted[name]; (mode == artifactTargetType || mode == artifactTargetConstructedType) && ok {
				candidates = append(candidates, genericType(factory, name, len(selected.TypeParameters())))
			}
		case tsgo.InterfaceDeclaration:
			name := selected.Name().Text()
			declarations = append(declarations, "interface "+name)
			if _, ok := accepted[name]; mode == artifactTargetType && ok {
				candidates = append(candidates, genericType(factory, name, len(selected.TypeParameters())))
			}
		case tsgo.TypeAliasDeclaration:
			name := selected.Name().Text()
			declarations = append(declarations, "type "+name)
			if _, ok := accepted[name]; mode == artifactTargetType && ok {
				candidates = append(candidates, genericType(factory, name, len(selected.TypeParameters())))
			}
		case tsgo.FunctionDeclaration:
			name := selected.Name().Text()
			declarations = append(declarations, "function "+name)
			if _, ok := accepted[name]; mode == artifactTargetValue && ok {
				candidates = append(candidates, factory.TypeQueryNode(factory.Identifier(name), nil))
			}
		case tsgo.VariableStatement:
			for _, declaration := range selected.DeclarationList().Declarations() {
				identifier, ok := declaration.Name().(tsgo.Identifier)
				name := ""
				if ok {
					name = identifier.Text()
					declarations = append(declarations, "variable "+name)
				}
				if _, acceptedName := accepted[name]; !acceptedName {
					continue
				}
				switch mode {
				case artifactTargetValue:
					candidates = append(candidates, factory.TypeQueryNode(factory.Identifier(name), nil))
				case artifactTargetConstructedType:
					candidates = append(candidates, factory.TypeReferenceNode(
						factory.Identifier("InstanceType"),
						[]tsgo.TypeNode{factory.TypeQueryNode(factory.Identifier(name), nil)},
					))
				}
			}
		default:
			declarations = append(declarations, fmt.Sprintf("%T", statement))
		}
	}
	if len(candidates) != 1 {
		return nil, &Error{
			Subject: names[0],
			Reason: fmt.Sprintf(
				"generated artifact has %d fact targets in [%s]",
				len(candidates),
				strings.Join(declarations, ", "),
			),
		}
	}
	return candidates[0], nil
}
