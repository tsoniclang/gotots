package sourcefact

import (
	"go/types"
	"sort"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const interfaceImplementationSchema = "gotots-go-interface-implementation-fact-v1"

type interfaceImplementation struct {
	contractType types.Type
	contract     *types.Interface
	key          string
}

func InterfaceImplementations(
	context api.Context,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if artifact == nil {
		return api.StatementEmission{}, &Error{Reason: "interface implementation owner is invalid"}
	}
	sourceType, ok := artifact.InterfaceAdapterType()
	if !ok {
		return api.StatementEmission{}, &Error{Reason: "interface implementation owner is invalid"}
	}
	implementations, err := selectedInterfaceImplementations(artifact, requirements)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := exactDeclarationTarget(
		context.Factory(),
		[]string{artifact.TargetName()},
		artifactTargetConstructedType,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	emissions := make([]api.StatementEmission, 0, len(implementations))
	for _, implementation := range implementations {
		arguments, err := interfaceImplementationArguments(
			context.Factory(),
			sourceType,
			implementation,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emission, err := attribute.Apply(
			context,
			target,
			api.RuntimeSourceInterfaceFact,
			arguments...,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emissions = append(emissions, emission)
	}
	return combine(emissions)
}

func selectedInterfaceImplementations(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]interfaceImplementation, error) {
	baseline := 0
	byKey := make(map[string]interfaceImplementation)
	for _, requirement := range requirements {
		selected, ok := requirement.InterfaceAdapter()
		if !ok || selected != artifact {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "interface adapter received a foreign requirement"}
		}
		_, contractType, contract, key, demanded := requirement.InterfaceAdapterContract()
		if !demanded {
			baseline++
			continue
		}
		if contract == nil {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "interface implementation contract is invalid"}
		}
		contract = contract.Complete()
		if contractType == nil || !contract.IsMethodSet() || key == "" {
			return nil, &Error{Subject: artifact.TargetName(), Reason: "interface implementation contract is invalid"}
		}
		candidate := interfaceImplementation{
			contractType: contractType,
			contract:     contract,
			key:          key,
		}
		if previous, duplicate := byKey[key]; duplicate {
			if !types.Identical(previous.contractType, contractType) ||
				!types.Identical(previous.contract, contract) {
				return nil, &Error{Subject: artifact.TargetName(), Reason: "interface implementation key is ambiguous"}
			}
			continue
		}
		byKey[key] = candidate
	}
	if baseline != 1 {
		return nil, &Error{Subject: artifact.TargetName(), Reason: "interface implementation baseline is not singular"}
	}
	result := make([]interfaceImplementation, 0, len(byKey))
	for _, implementation := range byKey {
		result = append(result, implementation)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].key < result[right].key
	})
	return result, nil
}

func interfaceImplementationArguments(
	factory tsgo.Factory,
	sourceType types.Type,
	implementation interfaceImplementation,
) ([]tsgo.Expression, error) {
	if !types.Implements(sourceType, implementation.contract) {
		return nil, &Error{Subject: implementation.key, Reason: "source type does not implement interface contract"}
	}
	arguments := []tsgo.Expression{
		text(factory, interfaceImplementationSchema),
		text(factory, "implicit-implementation"),
		text(factory, sourceTypeKind(sourceType)),
		text(factory, environmentcontract.StableTypeString(sourceType)),
		text(factory, implementation.key),
		text(factory, environmentcontract.StableTypeString(implementation.contractType)),
		count(factory, implementation.contract.NumMethods()),
	}
	methodSet := types.NewMethodSet(sourceType)
	for index := range implementation.contract.NumMethods() {
		method := implementation.contract.Method(index)
		selection := methodSet.Lookup(method.Pkg(), method.Name())
		if selection == nil {
			return nil, &Error{Subject: implementation.key, Reason: "implemented method selection is absent"}
		}
		selected, ok := selection.Obj().(*types.Func)
		if !ok {
			return nil, &Error{Subject: implementation.key, Reason: "implemented member is not callable"}
		}
		interfaceIdentity := interfaceImplementationMethodIdentity(
			implementation.key,
			method,
			index,
			nil,
		)
		selectedIdentity, err := selectedImplementationMethodIdentity(
			sourceType,
			selected,
			selection.Index(),
		)
		if err != nil {
			return nil, err
		}
		arguments = append(
			arguments,
			count(factory, index),
			text(factory, interfaceIdentity),
			text(factory, selectedIdentity),
			truth(factory, selection.Indirect()),
			count(factory, len(selection.Index())),
		)
		for pathIndex, selectedIndex := range selection.Index() {
			arguments = append(
				arguments,
				text(factory, strconv.Itoa(pathIndex)),
				count(factory, selectedIndex),
			)
		}
	}
	return arguments, nil
}

func selectedImplementationMethodIdentity(
	sourceType types.Type,
	method *types.Func,
	path []int,
) (string, error) {
	if method == nil || sourceType == nil || len(path) == 0 {
		return "", &Error{Reason: "selected implementation method identity is invalid"}
	}
	signature, ok := method.Type().(*types.Signature)
	if ok && signature.Recv() != nil && method.Pkg() != nil {
		contract, err := environmentcontract.Describe(method.Origin())
		if err != nil {
			return "", err
		}
		return contract.Identity(), nil
	}
	return interfaceImplementationMethodIdentity(
		"selected="+environmentcontract.StableTypeString(sourceType),
		method,
		path[len(path)-1],
		path,
	), nil
}

func interfaceImplementationMethodIdentity(
	owner string,
	method *types.Func,
	ordinal int,
	path []int,
) string {
	packagePath := ""
	if method != nil && method.Pkg() != nil {
		packagePath = method.Pkg().Path()
	}
	name := ""
	signature := ""
	if method != nil {
		name = method.Name()
		signature = environmentcontract.StableTypeString(method.Type())
	}
	identity := owner + "|method=" + packagePath + "." + name +
		"|ordinal=" + strconv.Itoa(ordinal) + "|signature=" + signature
	for index, selected := range path {
		identity += "|path=" + strconv.Itoa(index) + ":" + strconv.Itoa(selected)
	}
	return identity
}
