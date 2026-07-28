package complex

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	MakeMember = "make"
	RealMember = "real"
	ImagMember = "imag"
)

func buildClass(
	factory tsgo.Factory,
	className string,
	brandName string,
	roundName string,
) tsgo.ClassDeclaration {
	target := builder{factory: factory}
	numberType := target.numberType()
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		target.id(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.DeclareKeyword(),
					factory.PrivateKeyword(),
					factory.ReadonlyKeyword(),
				},
				target.id(brandName),
				nil,
				target.voidType(),
				nil,
			),
			factory.ConstructorDeclaration(
				[]tsgo.ModifierLike{factory.PrivateKeyword()},
				nil,
				[]tsgo.ParameterDeclaration{
					factory.ParameterDeclaration(
						[]tsgo.ModifierLike{
							factory.PublicKeyword(),
							factory.ReadonlyKeyword(),
						},
						nil,
						target.id(RealMember),
						nil,
						numberType,
						nil,
					),
					factory.ParameterDeclaration(
						[]tsgo.ModifierLike{
							factory.PublicKeyword(),
							factory.ReadonlyKeyword(),
						},
						nil,
						target.id(ImagMember),
						nil,
						numberType,
						nil,
					),
				},
				nil,
				factory.Block(nil, true),
			),
			buildConstructorMethod(target, className, roundName),
		},
	)
}

func buildConstructorMethod(
	target builder,
	className string,
	roundName string,
) tsgo.MethodDeclaration {
	realPart := tsgo.Expression(target.id("real"))
	imaginaryPart := tsgo.Expression(target.id("imag"))
	if roundName != "" {
		realPart = target.call(target.id(roundName), realPart)
		imaginaryPart = target.call(target.id(roundName), imaginaryPart)
	}
	return target.factory.MethodDeclaration(
		[]tsgo.ModifierLike{
			target.factory.PublicKeyword(),
			target.factory.StaticKeyword(),
		},
		nil,
		target.id(MakeMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			target.parameter("real", target.numberType()),
			target.parameter("imag", target.numberType()),
		},
		target.typeReference(className),
		target.factory.Block([]tsgo.Statement{
			target.factory.ReturnStatement(target.factory.NewExpression(
				target.id(className),
				nil,
				[]tsgo.Expression{realPart, imaginaryPart},
			)),
		}, true),
	)
}
