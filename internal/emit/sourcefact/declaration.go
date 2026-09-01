package sourcefact

import (
	"go/ast"
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func SourceDeclaration(
	context api.Context,
	object types.Object,
	source ast.Node,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
	requirements []api.DeclarationRequirement,
	additionalBindings []string,
) (api.StatementEmission, error) {
	typeName, typeDeclaration := object.(*types.TypeName)
	var typeSpec *ast.TypeSpec
	if typeDeclaration {
		var ok bool
		typeSpec, ok = source.(*ast.TypeSpec)
		if !ok {
			return api.StatementEmission{}, &Error{
				Subject: object.Name(),
				Reason:  "type declaration occurrence is not a type specification",
			}
		}
		var err error
		origin, err = TypeDeclarationOrigin(context, typeSpec, origin)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	declaration, err := DeclarationWithRequirements(
		context,
		object,
		origin,
		statements,
		requirements,
		additionalBindings,
	)
	if err != nil || !typeDeclaration || typeName.IsAlias() {
		return declaration, err
	}
	memberOrigins, err := TypeMemberOrigins(
		context,
		typeName,
		typeSpec,
		origin,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	members, err := TypeMembers(context, typeName, memberOrigins)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operations, err := StructOperations(context, typeName, requirements)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return combine([]api.StatementEmission{declaration, members, operations})
}

const (
	declarationSchema = "gotots-go-source-declaration-fact-v1"
	basicSchema       = "gotots-go-source-basic-fact-v1"
	aggregateSchema   = "gotots-go-source-aggregate-fact-v1"
	callableSchema    = "gotots-go-source-callable-fact-v1"
	interfaceSchema   = "gotots-go-source-interface-fact-v1"
	storageSchema     = "gotots-go-source-storage-fact-v1"
)

type declarationContract struct {
	kind      environmentcontract.ObjectKind
	identity  string
	receiver  string
	signature string
	value     string
}

func Declaration(
	context api.Context,
	object types.Object,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	name, err := context.Names().Declare(object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := declarationTarget(
		context.Factory(),
		object,
		name,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factTarget, err := NewTarget(target)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return declarationOnTarget(context, object, origin, factTarget)
}

func EnvironmentDeclaration(
	context api.Context,
	object types.Object,
	origin DeclarationOrigin,
	target Target,
) (api.StatementEmission, error) {
	if !origin.Valid() || origin.kind != "environment-contract" {
		return api.StatementEmission{}, &Error{Reason: "environment declaration origin is invalid"}
	}
	return declarationOnTarget(context, object, origin, target)
}

func LocalTypeDeclaration(
	context api.Context,
	object *types.TypeName,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	contract, err := localTypeDeclarationContract(context, object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	name, err := context.Names().Declare(object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := declarationTarget(
		context.Factory(),
		object,
		name,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factTarget, err := NewTarget(target)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return declarationOnTargetWithContract(
		context,
		object,
		origin,
		factTarget,
		contract,
	)
}

func declarationOnTarget(
	context api.Context,
	object types.Object,
	origin DeclarationOrigin,
	target Target,
) (api.StatementEmission, error) {
	if object == nil || !origin.Valid() {
		return api.StatementEmission{}, &Error{Reason: "source declaration fact is invalid"}
	}
	contract, err := packageDeclarationContract(object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return declarationOnTargetWithContract(context, object, origin, target, contract)
}

func declarationOnTargetWithContract(
	context api.Context,
	object types.Object,
	origin DeclarationOrigin,
	target Target,
	contract declarationContract,
) (api.StatementEmission, error) {
	if object == nil || !origin.Valid() || contract.identity == "" {
		return api.StatementEmission{}, &Error{Reason: "source declaration contract is invalid"}
	}
	arguments := declarationArguments(context.Factory(), object, contract, origin)
	arguments = append(arguments, origin.arguments(context.Factory())...)
	emissions := make([]api.StatementEmission, 0, 2)
	declaration, err := target.apply(
		context,
		api.RuntimeSourceDeclarationFact,
		arguments...,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	emissions = append(emissions, declaration)
	semantic, selected, err := semanticTypeFact(
		context,
		object,
		target,
		contract.identity,
		origin,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if selected {
		emissions = append(emissions, semantic)
	}
	return combine(emissions)
}

func packageDeclarationContract(
	object types.Object,
) (declarationContract, error) {
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return declarationContract{}, err
	}
	return declarationContract{
		kind:      contract.Kind(),
		identity:  contract.Identity(),
		receiver:  contract.Receiver(),
		signature: contract.Signature(),
		value:     contract.Value(),
	}, nil
}

func localTypeDeclarationContract(
	context api.Context,
	object *types.TypeName,
) (declarationContract, error) {
	if object == nil || object.Pkg() == nil ||
		object.Pkg() != context.TypesPackage() ||
		object.Parent() == nil || object.Parent() == object.Pkg().Scope() ||
		object.Parent().Lookup(object.Name()) != object {
		return declarationContract{}, &Error{Reason: "local type declaration is invalid"}
	}
	identity, err := typeidentity.LexicalNamedObjectKey(
		object,
		context.ArtifactOwner(),
		context.TypesPackage().Scope(),
	)
	if err != nil {
		return declarationContract{}, err
	}
	return declarationContract{
		kind:     environmentcontract.ObjectType,
		identity: identity,
		signature: "defined=" + environmentcontract.StableTypeString(object.Type()) +
			"|underlying=" + environmentcontract.StableTypeString(object.Type().Underlying()),
	}, nil
}

func declarationTarget(
	factory tsgo.Factory,
	object types.Object,
	name string,
	statements []tsgo.Statement,
) (tsgo.TypeNode, error) {
	if _, typeDeclaration := object.(*types.TypeName); typeDeclaration {
		return exactDeclarationTarget(
			factory,
			[]string{name},
			artifactTargetType,
			statements,
		)
	}
	names := []string{name}
	if _, callable := object.(*types.Func); callable {
		names = append(names, name+api.GenericKernelSuffix)
	}
	if _, constant := object.(*types.Const); constant {
		names = append(names, name+"$constant")
	}
	return exactDeclarationTarget(
		factory,
		names,
		artifactTargetValue,
		statements,
	)
}

func declarationArguments(
	factory tsgo.Factory,
	object types.Object,
	contract declarationContract,
	origin DeclarationOrigin,
) []tsgo.Expression {
	parameters := api.GenericDeclarationParameters(object)
	signature := contract.signature
	if _, typeDeclaration := object.(*types.TypeName); typeDeclaration {
		if basis, bounded := origin.EnvironmentBasis(); bounded {
			signature = "defined-environment-basis=" + basis
		}
	}
	arguments := []tsgo.Expression{
		text(factory, declarationSchema),
		text(factory, contract.identity),
		text(factory, objectKind(contract.kind)),
		text(factory, object.Pkg().Path()),
		text(factory, object.Name()),
		text(factory, contract.receiver),
		text(factory, signature),
		text(factory, contract.value),
		text(factory, declarationMode(object)),
		count(factory, len(parameters)),
	}
	for index, parameter := range parameters {
		arguments = append(arguments,
			text(factory, "type-parameter"),
			count(factory, index),
			text(factory, parameter.Obj().Name()),
			text(factory, environmentcontract.StableTypeString(parameter.Constraint())),
		)
	}
	return arguments
}

func objectKind(kind environmentcontract.ObjectKind) string {
	switch kind {
	case environmentcontract.ObjectConstant:
		return "constant"
	case environmentcontract.ObjectType:
		return "type"
	case environmentcontract.ObjectVariable:
		return "variable"
	case environmentcontract.ObjectFunction:
		return "function"
	case environmentcontract.ObjectBuiltin:
		return "builtin"
	default:
		return "invalid-" + strconv.Itoa(int(kind))
	}
}

func declarationMode(object types.Object) string {
	typeName, ok := object.(*types.TypeName)
	if !ok {
		return "not-type"
	}
	if typeName.IsAlias() {
		return "alias"
	}
	return "defined"
}

func semanticTypeFact(
	context api.Context,
	object types.Object,
	target Target,
	identity string,
	origin DeclarationOrigin,
) (api.StatementEmission, bool, error) {
	factory := context.Factory()
	switch selected := object.(type) {
	case *types.Func:
		signature, ok := selected.Type().(*types.Signature)
		if !ok {
			return api.StatementEmission{}, false, &Error{
				Subject: selected.Name(),
				Reason:  "function has no signature",
			}
		}
		emission, err := target.apply(
			context,
			api.RuntimeSourceCallableFact,
			callableArguments(factory, identity, signature)...,
		)
		return emission, true, err
	case *types.TypeName:
		return typeFact(context, selected, target, identity, origin)
	default:
		return api.StatementEmission{}, false, nil
	}
}

func typeFact(
	context api.Context,
	object *types.TypeName,
	target Target,
	identity string,
	origin DeclarationOrigin,
) (api.StatementEmission, bool, error) {
	factory := context.Factory()
	underlying := object.Type().Underlying()
	underlyingIdentity := environmentcontract.StableTypeString(underlying)
	environmentBasis, boundedEnvironment := origin.EnvironmentBasis()
	if boundedEnvironment {
		underlyingIdentity = "environment-contract=" + environmentBasis
	}
	var fact api.RuntimeSymbol
	arguments := []tsgo.Expression{
		text(factory, identity),
		text(factory, underlyingIdentity),
	}
	switch selected := underlying.(type) {
	case *types.Basic:
		fact = api.RuntimeSourceBasicFact
		arguments = append([]tsgo.Expression{text(factory, basicSchema)}, arguments...)
		arguments = append(arguments,
			text(factory, "basic"),
			text(factory, selected.Name()),
			count(factory, int(selected.Kind())),
		)
	case *types.Array:
		fact = api.RuntimeSourceAggregateFact
		arguments = append([]tsgo.Expression{text(factory, aggregateSchema)}, arguments...)
		arguments = append(arguments,
			text(factory, "array"),
			count(factory, int(selected.Len())),
			text(factory, environmentcontract.StableTypeString(selected.Elem())),
		)
	case *types.Slice:
		fact = api.RuntimeSourceAggregateFact
		arguments = append([]tsgo.Expression{text(factory, aggregateSchema)}, arguments...)
		arguments = append(
			arguments,
			text(factory, "slice"),
			text(factory, environmentcontract.StableTypeString(selected.Elem())),
		)
	case *types.Map:
		fact = api.RuntimeSourceAggregateFact
		arguments = append([]tsgo.Expression{text(factory, aggregateSchema)}, arguments...)
		arguments = append(
			arguments,
			text(factory, "map"),
			text(factory, environmentcontract.StableTypeString(selected.Key())),
			text(factory, environmentcontract.StableTypeString(selected.Elem())),
		)
	case *types.Chan:
		fact = api.RuntimeSourceAggregateFact
		arguments = append([]tsgo.Expression{text(factory, aggregateSchema)}, arguments...)
		arguments = append(arguments,
			text(factory, "channel"),
			text(factory, channelDirection(selected.Dir())),
			text(factory, environmentcontract.StableTypeString(selected.Elem())),
		)
	case *types.Pointer:
		fact = api.RuntimeSourceStorageFact
		arguments = append([]tsgo.Expression{text(factory, storageSchema)}, arguments...)
		arguments = append(
			arguments,
			text(factory, "typed-pointer"),
			text(factory, environmentcontract.StableTypeString(selected.Elem())),
		)
	case *types.Struct:
		fact = api.RuntimeSourceAggregateFact
		arguments = append([]tsgo.Expression{text(factory, aggregateSchema)}, arguments...)
		fieldCount := selected.NumFields()
		if boundedEnvironment {
			fieldCount = exportedFieldCount(selected)
		}
		arguments = append(arguments,
			text(factory, "struct"),
			count(factory, fieldCount),
		)
	case *types.Interface:
		selected = selected.Complete()
		fact = api.RuntimeSourceInterfaceFact
		arguments = append([]tsgo.Expression{text(factory, interfaceSchema)}, arguments...)
		arguments = append(arguments,
			text(factory, "interface"),
			count(factory, selected.NumExplicitMethods()),
			count(factory, selected.NumEmbeddeds()),
			truth(factory, selected.IsComparable()),
			truth(factory, selected.IsMethodSet()),
		)
		for index := range selected.NumEmbeddeds() {
			arguments = append(
				arguments,
				text(factory, "embedded"),
				count(factory, index),
				text(factory, environmentcontract.StableTypeString(selected.EmbeddedType(index))),
			)
		}
	case *types.Signature:
		fact = api.RuntimeSourceCallableFact
		arguments = callableArguments(factory, identity, selected)
	default:
		return api.StatementEmission{}, false, nil
	}
	emission, err := target.apply(context, fact, arguments...)
	return emission, true, err
}

func exportedFieldCount(structure *types.Struct) int {
	count := 0
	for index := range structure.NumFields() {
		field := structure.Field(index)
		if field.Exported() || field.Embedded() {
			count++
		}
	}
	return count
}

func channelDirection(direction types.ChanDir) string {
	switch direction {
	case types.SendRecv:
		return "send-receive"
	case types.SendOnly:
		return "send-only"
	case types.RecvOnly:
		return "receive-only"
	default:
		return "invalid"
	}
}

func callableArguments(
	factory tsgo.Factory,
	identity string,
	signature *types.Signature,
) []tsgo.Expression {
	arguments := []tsgo.Expression{
		text(factory, callableSchema),
		text(factory, identity),
		text(factory, receiverMode(signature)),
		truth(factory, signature.Variadic()),
		count(factory, signature.Params().Len()),
		count(factory, signature.Results().Len()),
	}
	arguments = appendTuple(arguments, factory, "parameter", signature.Params())
	arguments = appendTuple(arguments, factory, "result", signature.Results())
	return arguments
}

func appendTuple(
	arguments []tsgo.Expression,
	factory tsgo.Factory,
	role string,
	tuple *types.Tuple,
) []tsgo.Expression {
	for index := range tuple.Len() {
		item := tuple.At(index)
		arguments = append(arguments,
			text(factory, role),
			count(factory, index),
			text(factory, item.Name()),
			text(factory, environmentcontract.StableTypeString(item.Type())),
		)
	}
	return arguments
}

func receiverMode(signature *types.Signature) string {
	if signature.Recv() == nil {
		return "none"
	}
	if _, pointer := types.Unalias(signature.Recv().Type()).(*types.Pointer); pointer {
		return "pointer"
	}
	return "value"
}

func combine(
	emissions []api.StatementEmission,
) (api.StatementEmission, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range emissions {
		statements = append(statements, emission.Statements()...)
		requests = append(requests, emission.Requests()...)
	}
	return api.NewStatementEmission(statements, api.CombineRequests(requests))
}
