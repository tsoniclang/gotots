package sourcefact

import (
	"fmt"
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const memberSchema = "gotots-go-source-member-fact-v3"

func TypeMembers(
	context api.Context,
	owner *types.TypeName,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	if owner == nil {
		return api.StatementEmission{}, &Error{Reason: "type member owner is nil"}
	}
	name, err := context.Names().Declare(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target := genericType(
		context.Factory(),
		name,
		len(api.GenericDeclarationParameters(owner)),
	)
	factTarget, err := NewTarget(target)
	if err != nil {
		return api.StatementEmission{}, err
	}
	contract, err := packageDeclarationContract(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return typeMembersOnTarget(
		context,
		owner,
		factTarget,
		contract.identity,
		origins,
	)
}

func LocalTypeMembers(
	context api.Context,
	owner *types.TypeName,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	contract, err := localTypeDeclarationContract(context, owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	name, err := context.Names().Declare(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := NewTarget(genericType(
		context.Factory(),
		name,
		len(api.GenericDeclarationParameters(owner)),
	))
	if err != nil {
		return api.StatementEmission{}, err
	}
	return typeMembersOnTarget(
		context,
		owner,
		target,
		contract.identity,
		origins,
	)
}

func TypeMembersOnTarget(
	context api.Context,
	owner *types.TypeName,
	target Target,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	if owner == nil {
		return api.StatementEmission{}, &Error{Reason: "type member owner is nil"}
	}
	contract, err := packageDeclarationContract(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return typeMembersOnTarget(context, owner, target, contract.identity, origins)
}

func typeMembersOnTarget(
	context api.Context,
	owner *types.TypeName,
	target Target,
	ownerIdentity string,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	if owner == nil || ownerIdentity == "" {
		return api.StatementEmission{}, &Error{Reason: "type member owner is invalid"}
	}
	switch selected := owner.Type().Underlying().(type) {
	case *types.Struct:
		return structFields(context, target, ownerIdentity, selected, origins)
	case *types.Interface:
		return interfaceMethods(
			context,
			target,
			ownerIdentity,
			selected.Complete(),
			origins,
		)
	default:
		return api.NewStatementEmission(nil, nil)
	}
}

func ConcreteMethod(
	context api.Context,
	method *types.Func,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if method == nil {
		return api.StatementEmission{}, &Error{Reason: "concrete method is nil"}
	}
	method = method.Origin()
	targetMethod, err := context.Names().MethodTarget(method)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if targetMethod.Kind() != api.MethodTargetClassMember {
		return api.StatementEmission{}, &Error{
			Subject: method.FullName(),
			Reason:  "method does not use a class-member target",
		}
	}
	receiver := api.MethodReceiverTypeName(method)
	reference, err := context.Names().TypeReference(receiver)
	if err != nil {
		return api.StatementEmission{}, err
	}
	contract, err := packageDeclarationContract(method)
	if err != nil {
		return api.StatementEmission{}, err
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return api.StatementEmission{}, &Error{
			Subject: method.FullName(),
			Reason:  "method has no signature",
		}
	}
	target, memberName, err := exactConcreteMethodTarget(
		context.Factory(),
		reference.Name(),
		len(api.GenericDeclarationParameters(receiver)),
		targetMethod.Name(),
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	declarationArguments := declarationArguments(
		context.Factory(),
		method,
		contract,
		origin,
	)
	declarationArguments = append(
		declarationArguments,
		origin.arguments(context.Factory())...,
	)
	declaration, err := attribute.ApplyMember(
		context,
		target,
		attribute.MemberMethod,
		memberName,
		api.RuntimeSourceDeclarationFact,
		declarationArguments...,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	callable, err := attribute.ApplyMember(
		context,
		target,
		attribute.MemberMethod,
		memberName,
		api.RuntimeSourceCallableFact,
		callableArguments(
			context.Factory(),
			contract.identity,
			signature,
		)...,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	combined, err := combine([]api.StatementEmission{declaration, callable})
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(
		combined.Statements(),
		api.CombineRequests(
			combined.Requests(),
			reference.Requests(),
			targetMethod.Requests(),
		),
	)
}

func exactConcreteMethodTarget(
	factory tsgo.Factory,
	typeName string,
	typeParameterCount int,
	methodName string,
	statements []tsgo.Statement,
) (tsgo.TypeNode, string, error) {
	if typeName == "" || methodName == "" || typeParameterCount < 0 {
		return nil, "", &Error{Reason: "concrete method fact target is invalid"}
	}
	accepted := map[string]struct{}{
		methodName:                           {},
		methodName + api.GenericKernelSuffix: {},
	}
	var target tsgo.TypeNode
	selectedName := ""
	for _, statement := range statements {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != typeName {
			continue
		}
		for _, member := range class.Members() {
			method, ok := member.(tsgo.MethodDeclaration)
			if !ok {
				continue
			}
			identifier, ok := method.Name().(tsgo.Identifier)
			if !ok {
				continue
			}
			if _, selected := accepted[identifier.Text()]; !selected {
				continue
			}
			if target != nil {
				return nil, "", &Error{
					Subject: methodName,
					Reason:  "concrete method fact target is ambiguous",
				}
			}
			selectedName = identifier.Text()
			if syntaxModifier(method.Modifiers(), tsgo.SyntaxKindStaticKeyword) {
				target = factory.TypeQueryNode(factory.Identifier(typeName), nil)
			} else {
				target = genericType(factory, typeName, typeParameterCount)
			}
		}
	}
	if target == nil {
		return nil, "", &Error{
			Subject: methodName,
			Reason:  "concrete method fact target is absent",
		}
	}
	return target, selectedName, nil
}

func syntaxModifier(modifiers []tsgo.ModifierLike, kind tsgo.SyntaxKind) bool {
	for _, modifier := range modifiers {
		if modifier.Kind() == kind {
			return true
		}
	}
	return false
}

func structFields(
	context api.Context,
	target Target,
	ownerIdentity string,
	structure *types.Struct,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	emissions := make([]api.StatementEmission, 0, structure.NumFields())
	for index := range structure.NumFields() {
		field := structure.Field(index)
		origin, found := origins.field(index, field)
		if !found {
			expected := "absent"
			if index < len(origins.orderedField) {
				expected = origins.orderedField[index].Name() + ":" +
					environmentcontract.StableTypeString(origins.orderedField[index].Type())
			}
			return api.StatementEmission{}, &Error{
				Subject: field.Name(),
				Reason: fmt.Sprintf(
					"struct field source origin is absent at ordinal %d of %d; selected %s:%s, origin %s",
					index,
					len(origins.orderedField),
					field.Name(),
					environmentcontract.StableTypeString(field.Type()),
					expected,
				),
			}
		}
		if origins.boundedEnvironment() && !field.Exported() && !field.Embedded() {
			continue
		}
		member, err := context.Names().Member(field)
		if err != nil {
			return api.StatementEmission{}, err
		}
		packagePath := ""
		if field.Pkg() != nil {
			packagePath = field.Pkg().Path()
		}
		arguments := []tsgo.Expression{
			text(context.Factory(), memberSchema),
			text(context.Factory(), "field"),
			text(context.Factory(), ownerIdentity),
			count(context.Factory(), index),
			text(context.Factory(), field.Name()),
			text(context.Factory(), member),
			text(context.Factory(), packagePath),
			text(context.Factory(), environmentcontract.StableTypeString(field.Type())),
			text(context.Factory(), structure.Tag(index)),
			truth(context.Factory(), field.Embedded()),
			truth(context.Factory(), field.Exported()),
			truth(context.Factory(), field.Name() == "_"),
			text(context.Factory(), "authored"),
		}
		arguments = append(arguments, origin.arguments(context.Factory())...)
		emission, err := target.apply(
			context,
			api.RuntimeSourceDeclarationFact,
			arguments...,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emissions = append(emissions, emission)
	}
	return combine(emissions)
}

func interfaceMethods(
	context api.Context,
	target Target,
	ownerIdentity string,
	interfaceType *types.Interface,
	origins MemberOriginSet,
) (api.StatementEmission, error) {
	emissions := make([]api.StatementEmission, 0, interfaceType.NumMethods())
	for index := range interfaceType.NumMethods() {
		method := interfaceType.Method(index)
		if origins.boundedEnvironment() && !method.Exported() {
			continue
		}
		member, err := context.Names().InterfaceMethodName(method)
		if err != nil {
			return api.StatementEmission{}, err
		}
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			return api.StatementEmission{}, &Error{
				Subject: method.FullName(),
				Reason:  "interface method has no signature",
			}
		}
		arguments := callableArguments(
			context.Factory(),
			interfaceMethodIdentity(ownerIdentity, method, index),
			signature,
		)
		arguments = append(arguments,
			text(context.Factory(), "interface-method"),
			count(context.Factory(), index),
		)
		origin, authored := origins.method(method)
		if authored {
			arguments = append(arguments, text(context.Factory(), "authored"))
			arguments = append(arguments, origin.arguments(context.Factory())...)
		} else {
			arguments = append(arguments, text(context.Factory(), "inherited"))
		}
		memberTarget, err := NewMemberTarget(
			target.typeNode,
			attribute.MemberMethod,
			member,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emission, err := memberTarget.apply(
			context,
			api.RuntimeSourceCallableFact,
			arguments...,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emissions = append(emissions, emission)
	}
	return combine(emissions)
}

func interfaceMethodIdentity(
	ownerIdentity string,
	method *types.Func,
	ordinal int,
) string {
	packagePath := ""
	if method.Pkg() != nil {
		packagePath = method.Pkg().Path()
	}
	return ownerIdentity + "|interface-method=" + packagePath + "." +
		method.Name() + "|ordinal=" + strconv.Itoa(ordinal)
}

func SourceVariableMemberArguments(
	factory tsgo.Factory,
	variable *types.Var,
	member string,
	origin DeclarationOrigin,
) ([]tsgo.Expression, error) {
	contract, err := environmentcontract.Describe(variable)
	if err != nil {
		return nil, err
	}
	arguments := []tsgo.Expression{
		text(factory, memberSchema),
		text(factory, "package-variable"),
		text(factory, contract.Identity()),
		text(factory, variable.Pkg().Path()),
		text(factory, variable.Name()),
		text(factory, member),
		text(factory, environmentcontract.StableTypeString(variable.Type())),
		truth(factory, variable.Exported()),
	}
	return append(arguments, origin.arguments(factory)...), nil
}
