package sourcefact

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	implementationselection "github.com/tsoniclang/gotots/internal/emit/api/callableimplementation"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const implementationSchema = "gotots-go-implementation-fact-v2"

func CallableImplementation(
	context api.Context,
	function *types.Func,
	selected callableimplementation.Implementation,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if function == nil || selected.Function() != function {
		return api.StatementEmission{}, &Error{
			Reason: "callable implementation selection has a foreign function",
		}
	}
	variant := api.CallableImplementationVariantSource
	if selected.Variant() == callableimplementation.VariantKernel {
		variant = api.CallableImplementationVariantKernel
	}
	module := selected.Module()
	selection, err := implementationselection.NewSelection(
		selected.SourceIdentity(),
		module.OutputPath(),
		selected.Export(),
		variant,
		module.PackagePath(),
		module.ModulePath(),
		module.ModuleVersion(),
		module.Digest(),
		module.SourceDigest(),
		module.EquivalenceEnvelope(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return Implementation(context, function, selection, statements)
}

func Implementation(
	context api.Context,
	function *types.Func,
	selection api.CallableImplementationSelection,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if function == nil || function.Origin() != function || !selection.Valid() {
		return api.StatementEmission{}, &Error{Reason: "callable implementation fact is invalid"}
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.StatementEmission{}, err
	}
	envelope := selection.EquivalenceEnvelope()
	arguments := []tsgo.Expression{
		text(context.Factory(), implementationSchema),
		text(context.Factory(), contract.Identity()),
		text(context.Factory(), selection.SourceIdentity()),
		text(context.Factory(), implementationVariant(selection.Variant())),
		text(context.Factory(), selection.OutputPath()),
		text(context.Factory(), selection.Export()),
		text(context.Factory(), selection.PackagePath()),
		text(context.Factory(), selection.ModulePath()),
		text(context.Factory(), selection.ModuleVersion()),
		text(context.Factory(), selection.ModuleDigest()),
		text(context.Factory(), selection.SourceDigest()),
		text(context.Factory(), string(envelope.Kind)),
		text(context.Factory(), envelope.RelaxedBehavior),
		count(context.Factory(), len(envelope.PreservedObservables)),
	}
	for index, observable := range envelope.PreservedObservables {
		arguments = append(arguments, count(context.Factory(), index), text(context.Factory(), observable))
	}
	arguments = append(arguments, count(context.Factory(), len(envelope.Evidence)))
	for index, evidence := range envelope.Evidence {
		arguments = append(arguments, count(context.Factory(), index), text(context.Factory(), evidence))
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok {
		return api.StatementEmission{}, &Error{Reason: "callable implementation signature is invalid"}
	}
	if signature.Recv() != nil {
		targetMethod, err := context.Names().MethodTarget(function)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if targetMethod.Kind() == api.MethodTargetClassMember {
			receiver := api.MethodReceiverTypeName(function)
			if receiver == nil {
				return api.StatementEmission{}, &Error{Reason: "callable implementation method receiver is invalid"}
			}
			reference, referenceErr := context.Names().TypeReference(receiver)
			if referenceErr != nil {
				return api.StatementEmission{}, referenceErr
			}
			target := genericType(
				context.Factory(),
				reference.Name(),
				len(api.GenericDeclarationParameters(receiver)),
			)
			emission, applyErr := attribute.ApplyMember(
				context,
				target,
				attribute.MemberMethod,
				targetMethod.Name(),
				api.RuntimeSourceImplementationFact,
				arguments...,
			)
			if applyErr != nil {
				return api.StatementEmission{}, applyErr
			}
			return api.NewStatementEmission(
				emission.Statements(),
				api.CombineRequests(
					emission.Requests(),
					reference.Requests(),
					targetMethod.Requests(),
				),
			)
		}
	}
	name, err := context.Names().Declare(function)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := exactDeclarationTarget(
		context.Factory(),
		[]string{name, name + api.GenericKernelSuffix},
		artifactTargetValue,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return attribute.Apply(
		context,
		target,
		api.RuntimeSourceImplementationFact,
		arguments...,
	)
}

func implementationVariant(variant api.CallableImplementationVariant) string {
	switch variant {
	case api.CallableImplementationVariantSource:
		return "source"
	case api.CallableImplementationVariantKernel:
		return "kernel"
	default:
		return "invalid"
	}
}
