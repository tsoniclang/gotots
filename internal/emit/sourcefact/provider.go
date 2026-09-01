package sourcefact

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const providerImplementationSchema = "gotots-go-provider-implementation-fact-v1"
const externalImplementationSchema = "gotots-go-external-implementation-fact-v1"

func ExternalImplementation(
	context api.Context,
	certificate *externalcertify.Certificate,
	target api.ExternalFunctionTarget,
	function *types.Func,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if certificate == nil || function == nil {
		return api.StatementEmission{}, &Error{
			Reason: "external implementation selection is incomplete",
		}
	}
	function = function.Origin()
	module, export, ok := target.Module()
	if !ok {
		return api.StatementEmission{}, &Error{
			Subject: function.Name(),
			Reason:  "external module implementation target is invalid",
		}
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.StatementEmission{}, err
	}
	for _, binding := range certificate.Bindings() {
		if binding.SourceIdentity() != contract.Identity() {
			continue
		}
		selectedModule, selectedExport, owner, fingerprint, moduleTarget :=
			binding.ModuleTarget()
		if !moduleTarget || selectedModule != module || selectedExport != export ||
			binding.SourceSignature() != contract.Signature() {
			return api.StatementEmission{}, &Error{
				Subject: function.Name(),
				Reason:  "external implementation target differs from its certificate",
			}
		}
		return ExternalImplementationFact(
			context,
			function,
			statements,
			certificate.ManifestDigest(),
			module,
			export,
			owner,
			fingerprint,
			binding.SourceModulePath(),
			binding.SourceModuleVersion(),
			binding.SourceLocation(),
		)
	}
	return api.StatementEmission{}, &Error{
		Subject: function.Name(),
		Reason:  "external implementation certificate binding is absent",
	}
}

type ProviderImplementation struct {
	kind                string
	manifestDigest      string
	module              string
	export              string
	member              string
	access              string
	representation      string
	definedValue        string
	effect              string
	implementationOwner string
	targetFingerprint   string
	selections          []gostdlib.UseSelection
}

func NewProviderImplementation(
	kind string,
	manifestDigest string,
	module string,
	export string,
	member string,
	access gostdlib.AccessKind,
	representation gostdlib.RepresentationKind,
	definedValue gostdlib.DefinedValueRepresentationKind,
	effect gostdlib.EffectKind,
	implementationOwner string,
	targetFingerprint string,
	selections []gostdlib.UseSelection,
) (ProviderImplementation, error) {
	if (kind != "binding" && kind != "representation") ||
		manifestDigest == "" || module == "" || export == "" ||
		!access.Valid() || implementationOwner == "" ||
		targetFingerprint == "" ||
		(access == gostdlib.AccessExport && member != "") ||
		(access != gostdlib.AccessExport && member == "") {
		return ProviderImplementation{}, &Error{Reason: "provider implementation selection is incomplete"}
	}
	selected := make([]gostdlib.UseSelection, len(selections))
	copy(selected, selections)
	for _, selection := range selected {
		if selection.Kind() == gostdlib.UseSelectionNone || selection.EvidenceKey() == "" {
			return ProviderImplementation{}, &Error{Reason: "provider use selection is invalid"}
		}
	}
	return ProviderImplementation{
		kind:                kind,
		manifestDigest:      manifestDigest,
		module:              module,
		export:              export,
		member:              member,
		access:              string(access),
		representation:      string(representation),
		definedValue:        string(definedValue),
		effect:              string(effect),
		implementationOwner: implementationOwner,
		targetFingerprint:   targetFingerprint,
		selections:          selected,
	}, nil
}

func ProviderImplementationFact(
	context api.Context,
	object types.Object,
	target Target,
	selection ProviderImplementation,
) (api.StatementEmission, error) {
	if object == nil || selection.kind == "" {
		return api.StatementEmission{}, &Error{Reason: "provider implementation fact is invalid"}
	}
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	arguments := []tsgo.Expression{
		text(context.Factory(), providerImplementationSchema),
		text(context.Factory(), selection.kind),
		text(context.Factory(), contract.Identity()),
		text(context.Factory(), selection.manifestDigest),
		text(context.Factory(), selection.module),
		text(context.Factory(), selection.export),
		text(context.Factory(), selection.member),
		text(context.Factory(), selection.access),
		text(context.Factory(), selection.representation),
		text(context.Factory(), selection.definedValue),
		text(context.Factory(), selection.effect),
		text(context.Factory(), selection.implementationOwner),
		text(context.Factory(), selection.targetFingerprint),
		count(context.Factory(), len(selection.selections)),
	}
	for index, selected := range selection.selections {
		arguments = append(
			arguments,
			count(context.Factory(), index),
			text(context.Factory(), selected.Kind().String()),
			text(context.Factory(), selected.EvidenceKey()),
		)
	}
	return target.apply(
		context,
		api.RuntimeSourceImplementationFact,
		arguments...,
	)
}

func ExternalImplementationFact(
	context api.Context,
	function *types.Func,
	statements []tsgo.Statement,
	manifestDigest string,
	module string,
	export string,
	implementationOwner string,
	targetFingerprint string,
	sourceModulePath string,
	sourceModuleVersion string,
	sourceLocation string,
) (api.StatementEmission, error) {
	if function == nil || function.Origin() != function || len(statements) == 0 ||
		manifestDigest == "" || module == "" || export == "" ||
		implementationOwner == "" || targetFingerprint == "" ||
		sourceModulePath == "" || sourceLocation == "" {
		return api.StatementEmission{}, &Error{Reason: "external implementation fact is invalid"}
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, targetRequests, err := callableImplementationTarget(
		context,
		function,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	fact, err := target.apply(
		context,
		api.RuntimeSourceImplementationFact,
		text(context.Factory(), externalImplementationSchema),
		text(context.Factory(), "external-module"),
		text(context.Factory(), contract.Identity()),
		text(context.Factory(), manifestDigest),
		text(context.Factory(), module),
		text(context.Factory(), export),
		text(context.Factory(), implementationOwner),
		text(context.Factory(), targetFingerprint),
		text(context.Factory(), sourceModulePath),
		text(context.Factory(), sourceModuleVersion),
		text(context.Factory(), sourceLocation),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(
		fact.Statements(),
		api.CombineRequests(targetRequests, fact.Requests()),
	)
}

func callableImplementationTarget(
	context api.Context,
	function *types.Func,
	statements []tsgo.Statement,
) (Target, []api.RootRequest, error) {
	if function.Signature().Recv() != nil {
		method, err := context.Names().MethodTarget(function)
		if err != nil {
			return Target{}, nil, err
		}
		if method.Kind() != api.MethodTargetClassMember {
			return Target{}, nil, &Error{
				Subject: function.FullName(),
				Reason:  "external method has no class-member target",
			}
		}
		receiver := api.MethodReceiverTypeName(function)
		reference, err := context.Names().TypeReference(receiver)
		if err != nil {
			return Target{}, nil, err
		}
		target, err := NewMemberTarget(
			genericType(
				context.Factory(),
				reference.Name(),
				len(api.GenericDeclarationParameters(receiver)),
			),
			attribute.MemberMethod,
			method.Name(),
		)
		return target, api.CombineRequests(reference.Requests(), method.Requests()), err
	}
	name, err := context.Names().Declare(function)
	if err != nil {
		return Target{}, nil, err
	}
	targetType, err := declarationTarget(
		context.Factory(),
		function,
		name,
		statements,
	)
	if err != nil {
		return Target{}, nil, err
	}
	target, err := NewTarget(targetType)
	return target, nil, err
}
