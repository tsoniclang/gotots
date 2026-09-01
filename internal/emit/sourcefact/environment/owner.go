package environmentsourcefact

import (
	"go/types"
	"strconv"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	canonicalsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct {
	context       api.Context
	sourcePackage *load.Package
	outputPath    string
	programDigest string
	registry      *emitnaming.Registry
	provider      *gostdlibcertify.Certificate
}

func New(
	context api.Context,
	sourcePackage *load.Package,
	outputPath string,
	programDigest string,
	registry *emitnaming.Registry,
	provider *gostdlibcertify.Certificate,
) (Owner, error) {
	if sourcePackage == nil || sourcePackage.Program() == nil ||
		outputPath == "" || programDigest == "" || registry == nil {
		return Owner{}, &canonicalsourcefact.Error{
			Reason: "environment source-fact owner is incomplete",
		}
	}
	return Owner{
		context:       context,
		sourcePackage: sourcePackage,
		outputPath:    outputPath,
		programDigest: programDigest,
		registry:      registry,
		provider:      provider,
	}, nil
}

func (o Owner) Origin(object types.Object) (canonicalsourcefact.DeclarationOrigin, error) {
	if object == nil {
		return canonicalsourcefact.DeclarationOrigin{}, &canonicalsourcefact.Error{
			Reason: "environment declaration source identity is invalid",
		}
	}
	contract, err := environmentidentity.Describe(object)
	if err != nil {
		return canonicalsourcefact.DeclarationOrigin{}, err
	}
	return o.originWithIdentity(object, contract.Identity())
}

func (o Owner) originWithIdentity(
	object types.Object,
	identity string,
) (canonicalsourcefact.DeclarationOrigin, error) {
	if object == nil || identity == "" {
		return canonicalsourcefact.DeclarationOrigin{}, &canonicalsourcefact.Error{
			Reason: "environment declaration source identity is invalid",
		}
	}
	ownerKind, contractKey := canonicalsourcefact.PackageOwner(o.sourcePackage)
	origin, err := canonicalsourcefact.NewEnvironmentDeclarationOrigin(
		o.sourcePackage.Path(),
		o.sourcePackage.ModulePath(),
		o.sourcePackage.ModuleVersion(),
		ownerKind,
		contractKey,
		o.outputPath,
		identity,
		o.programDigest,
		environmentcontract.SourceLocation(o.sourcePackage, object),
	)
	if err != nil {
		return canonicalsourcefact.DeclarationOrigin{}, err
	}
	if _, typeDeclaration := object.(*types.TypeName); typeDeclaration {
		return origin.WithEnvironmentBasis(identity)
	}
	return origin, nil
}

func (o Owner) MemberOrigins(
	owner *types.TypeName,
) (canonicalsourcefact.MemberOriginSet, error) {
	contract, err := environmentidentity.Describe(owner)
	if err != nil {
		return canonicalsourcefact.MemberOriginSet{}, err
	}
	var objects []types.Object
	var origins []canonicalsourcefact.DeclarationOrigin
	switch selected := owner.Type().Underlying().(type) {
	case *types.Struct:
		for index := range selected.NumFields() {
			field := selected.Field(index)
			origin, originErr := o.originWithIdentity(
				field,
				contract.Identity()+"|field="+strconv.Itoa(index)+":"+field.Name(),
			)
			if originErr != nil {
				return canonicalsourcefact.MemberOriginSet{}, originErr
			}
			origin, originErr = origin.WithEnvironmentBasis(contract.Identity())
			if originErr != nil {
				return canonicalsourcefact.MemberOriginSet{}, originErr
			}
			objects = append(objects, field)
			origins = append(origins, origin)
		}
	case *types.Interface:
		selected = selected.Complete()
		for index := range selected.NumExplicitMethods() {
			method := selected.ExplicitMethod(index)
			origin, originErr := o.Origin(method)
			if originErr != nil {
				return canonicalsourcefact.MemberOriginSet{}, originErr
			}
			origin, originErr = origin.WithEnvironmentBasis(contract.Identity())
			if originErr != nil {
				return canonicalsourcefact.MemberOriginSet{}, originErr
			}
			objects = append(objects, method)
			origins = append(origins, origin)
		}
	}
	return canonicalsourcefact.NewMemberOriginSet(objects, origins)
}

func (o Owner) ProviderDeclaration(
	object types.Object,
	selections []gostdlib.UseSelection,
) (api.StatementEmission, bool, error) {
	if o.provider == nil {
		return api.StatementEmission{}, false, nil
	}
	providerTarget, selected, err := o.registry.ProviderTarget(object)
	if err != nil || !selected {
		return api.StatementEmission{}, selected, err
	}
	target, requests, err := o.providerTarget(object, providerTarget)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	origin, err := o.Origin(object)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	declaration, err := canonicalsourcefact.EnvironmentDeclaration(
		o.context,
		object,
		origin,
		target,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	statements := declaration.Statements()
	factRequests := declaration.Requests()
	if typeName, ok := object.(*types.TypeName); ok && !typeName.IsAlias() {
		origins, originErr := o.MemberOrigins(typeName)
		if originErr != nil {
			return api.StatementEmission{}, true, originErr
		}
		members, memberErr := canonicalsourcefact.TypeMembersOnTarget(
			o.context,
			typeName,
			target,
			origins,
		)
		if memberErr != nil {
			return api.StatementEmission{}, true, memberErr
		}
		statements = append(statements, members.Statements()...)
		factRequests = api.CombineRequests(factRequests, members.Requests())
	}
	implementation, err := o.providerImplementation(
		object,
		providerTarget,
		selections,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	implementationFact, err := canonicalsourcefact.ProviderImplementationFact(
		o.context,
		object,
		target,
		implementation,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	emission, err := api.NewStatementEmission(
		append(statements, implementationFact.Statements()...),
		api.CombineRequests(requests, factRequests, implementationFact.Requests()),
	)
	return emission, true, err
}

func (o Owner) State(variable *types.Var) (api.StatementEmission, error) {
	reference, err := o.context.Names().PackageVariable(variable)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := canonicalsourcefact.NewMemberTarget(
		o.context.Factory().TypeQueryNode(
			o.context.Factory().Identifier(reference.StateName()),
			nil,
		),
		attribute.MemberProperty,
		reference.FieldName(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	origin, err := o.Origin(variable)
	if err != nil {
		return api.StatementEmission{}, err
	}
	fact, err := canonicalsourcefact.EnvironmentDeclaration(
		o.context,
		variable,
		origin,
		target,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(
		fact.Statements(),
		api.CombineRequests(reference.Requests(), fact.Requests()),
	)
}

func (o Owner) providerTarget(
	object types.Object,
	provider emitnaming.ProviderTarget,
) (canonicalsourcefact.Target, []api.RootRequest, error) {
	names, ok := o.context.Names().(*emitnaming.File)
	if !ok {
		return canonicalsourcefact.Target{}, nil, &canonicalsourcefact.Error{
			Subject: object.Name(),
			Reason:  "provider fact target has no concrete name owner",
		}
	}
	phase := api.ImportPhaseValue
	if provider.Access() == gostdlib.AccessInstanceMethod ||
		(provider.Access() == gostdlib.AccessExport && isTypeDeclaration(object)) {
		phase = api.ImportPhaseType
	}
	reference, err := names.ProviderSymbol(provider.Module(), provider.Export(), phase)
	if err != nil {
		return canonicalsourcefact.Target{}, nil, err
	}
	factory := o.context.Factory()
	var targetType tsgo.TypeNode
	switch provider.Access() {
	case gostdlib.AccessExport:
		if isTypeDeclaration(object) {
			targetType = factory.TypeReferenceNode(
				reference.EntityName(factory),
				providerTypeArguments(factory, object),
			)
		} else {
			targetType = factory.TypeQueryNode(reference.EntityName(factory), nil)
		}
		target, targetErr := canonicalsourcefact.NewTarget(targetType)
		return target, reference.Requests(), targetErr
	case gostdlib.AccessStateMember, gostdlib.AccessStaticMethod:
		targetType = factory.TypeQueryNode(reference.EntityName(factory), nil)
		memberKind := attribute.MemberProperty
		if provider.Access() == gostdlib.AccessStaticMethod {
			memberKind = attribute.MemberMethod
		}
		target, targetErr := canonicalsourcefact.NewMemberTarget(
			targetType,
			memberKind,
			provider.Member(),
		)
		return target, reference.Requests(), targetErr
	case gostdlib.AccessInstanceMethod:
		targetType = factory.TypeReferenceNode(
			reference.EntityName(factory),
			providerTypeArguments(factory, object),
		)
		target, targetErr := canonicalsourcefact.NewMemberTarget(
			targetType,
			attribute.MemberMethod,
			provider.Member(),
		)
		return target, reference.Requests(), targetErr
	default:
		return canonicalsourcefact.Target{}, nil, &canonicalsourcefact.Error{
			Subject: object.Name(), Reason: "provider fact target access is invalid",
		}
	}
}

func (o Owner) providerImplementation(
	object types.Object,
	target emitnaming.ProviderTarget,
	selections []gostdlib.UseSelection,
) (canonicalsourcefact.ProviderImplementation, error) {
	contract, err := environmentidentity.Describe(object)
	if err != nil {
		return canonicalsourcefact.ProviderImplementation{}, err
	}
	if binding, ok := o.provider.Binding(contract.Identity()); ok {
		if binding.ModuleSpecifier() != target.Module() ||
			binding.Export() != target.Export() ||
			binding.Member() != target.Member() || binding.Access() != target.Access() {
			return canonicalsourcefact.ProviderImplementation{}, &canonicalsourcefact.Error{
				Subject: object.Name(),
				Reason:  "provider fact target differs from certified binding",
			}
		}
		return canonicalsourcefact.NewProviderImplementation(
			"binding",
			o.provider.ManifestDigest(),
			binding.ModuleSpecifier(),
			binding.Export(),
			binding.Member(),
			binding.Access(),
			binding.Representation(),
			binding.DefinedValueRepresentation(),
			binding.Effect(),
			binding.ImplementationOwner(),
			binding.TargetFingerprint(),
			selections,
		)
	}
	if !target.Representation() {
		return canonicalsourcefact.ProviderImplementation{}, &canonicalsourcefact.Error{
			Subject: object.Name(),
			Reason:  "provider declaration has no certified binding or representation",
		}
	}
	representation, ok := o.provider.ProviderRepresentation(target.Module(), target.Export())
	if !ok {
		return canonicalsourcefact.ProviderImplementation{}, &canonicalsourcefact.Error{
			Subject: object.Name(),
			Reason:  "provider representation certificate is absent",
		}
	}
	implementationOwner := representation.ImplementationOwner()
	fingerprint := representation.TargetFingerprint()
	if _, methodObject := object.(*types.Func); methodObject {
		selected, owns := representation.Method(contract.Identity())
		if !owns || selected.Member() != target.Member() {
			return canonicalsourcefact.ProviderImplementation{}, &canonicalsourcefact.Error{
				Subject: object.Name(),
				Reason:  "provider representation method certificate is absent",
			}
		}
		implementationOwner = selected.ImplementationOwner()
		fingerprint = selected.TargetFingerprint()
	}
	return canonicalsourcefact.NewProviderImplementation(
		"representation",
		o.provider.ManifestDigest(),
		target.Module(),
		target.Export(),
		target.Member(),
		target.Access(),
		target.TypeRepresentation(),
		target.DefinedValueRepresentation(),
		target.Effect(),
		implementationOwner,
		fingerprint,
		selections,
	)
}

func isTypeDeclaration(object types.Object) bool {
	_, ok := object.(*types.TypeName)
	return ok
}

func providerTypeArguments(factory tsgo.Factory, object types.Object) []tsgo.TypeNode {
	owner := object
	if method, ok := object.(*types.Func); ok && method.Signature().Recv() != nil {
		owner = api.MethodReceiverTypeName(method)
	}
	parameters := api.GenericDeclarationParameters(owner)
	arguments := make([]tsgo.TypeNode, len(parameters))
	for index := range arguments {
		arguments[index] = factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNeverKeyword,
		)
	}
	return arguments
}
