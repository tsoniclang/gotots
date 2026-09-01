package sourcefact

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func generatedStructFields(
	context api.Context,
	target tsgo.TypeNode,
	ownerIdentity string,
	structure *types.Struct,
) (api.StatementEmission, error) {
	emissions := make([]api.StatementEmission, 0, structure.NumFields())
	for index := range structure.NumFields() {
		field := structure.Field(index)
		member, err := context.Names().Member(field)
		if err != nil {
			return api.StatementEmission{}, err
		}
		packagePath := ""
		if field.Pkg() != nil {
			packagePath = field.Pkg().Path()
		}
		emission, err := attribute.Apply(
			context,
			target,
			api.RuntimeSourceDeclarationFact,
			text(context.Factory(), memberSchema),
			text(context.Factory(), "anonymous-field"),
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
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emissions = append(emissions, emission)
	}
	return combine(emissions)
}

func generatedInterfaceMethods(
	context api.Context,
	target tsgo.TypeNode,
	ownerIdentity string,
	contract *types.Interface,
) (api.StatementEmission, error) {
	emissions := make([]api.StatementEmission, 0, contract.NumMethods())
	for index := range contract.NumMethods() {
		method := contract.Method(index)
		member, err := context.Names().InterfaceMethodName(method)
		if err != nil {
			return api.StatementEmission{}, err
		}
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			return api.StatementEmission{}, &Error{
				Subject: method.FullName(),
				Reason:  "anonymous interface method has no signature",
			}
		}
		arguments := callableArguments(
			context.Factory(),
			interfaceMethodIdentity(ownerIdentity, method, index),
			signature,
		)
		arguments = append(
			arguments,
			text(context.Factory(), "anonymous-interface-method"),
			text(context.Factory(), ownerIdentity),
			count(context.Factory(), index),
		)
		emission, err := attribute.ApplyMember(
			context,
			target,
			attribute.MemberMethod,
			member,
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
