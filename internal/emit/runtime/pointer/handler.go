package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	AddressName      = "$go$address"
	CellName         = "cell"
	CellValueName    = "value"
	DereferenceName  = "dereference"
	DirectName       = "direct"
	EqualName        = "equal"
	ProjectName      = "project"
	ViewName         = "view"
	FieldName        = "field"
	FieldsName       = "fields"
	ChildName        = "child"
	ObjectFieldName  = "objectField"
	ElementName      = "element"
	IndexName        = "index"
	ArrayRegionName  = "arrayRegion"
	RegionName       = "$go$region"
	RegionMethod     = "region"
	UnsafeMemoryName = "$go$unsafeMemory"
	UnsafeViewName   = "$go$unsafeView"
	UnsafeBindName   = "$go$unsafeBind"
	unsafeRawName    = "$go$rawAccess"
)

type Capabilities struct {
	FieldPath    bool
	Region       bool
	UnsafeMemory bool
	Projection   bool
}

type builder struct {
	factory        tsgo.Factory
	className      string
	panicName      string
	denseIndexName string
	capabilities   Capabilities
}

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
	denseIndexName string,
) tsgo.Statement {
	return BuildWithCapabilities(
		factory,
		className,
		panicName,
		denseIndexName,
		Capabilities{},
	)
}

func BuildWithCapabilities(
	factory tsgo.Factory,
	className string,
	panicName string,
	denseIndexName string,
	capabilities Capabilities,
) tsgo.Statement {
	target := builder{
		factory:        factory,
		className:      className,
		panicName:      panicName,
		denseIndexName: denseIndexName,
		capabilities:   capabilities,
	}
	members := []tsgo.ClassElement{
		target.logicalProperty(),
		target.rootsProperty(),
		target.childrenProperty(),
		target.resolvedAddressProperty(),
		target.constructor(),
		target.addressGetter(),
		target.rootMethod(),
		target.childMethod(),
		target.cellMethod(),
		target.fieldMethod(),
		target.objectFieldMethod(),
		target.elementMethod(),
		target.indexMethod(),
		target.arrayRegionMethod(),
		target.equalMethod(),
		target.dereferenceMethod(),
		target.directMethod(),
		target.viewMethod(),
		target.valueGetter(),
		target.valueSetter(),
	}
	if capabilities.FieldPath {
		members = append(members, target.fieldsMethod())
	}
	if capabilities.Projection {
		members = append(members, target.projectMethod())
	}
	if capabilities.Region {
		members = append(members, target.regionMethod())
	}
	if capabilities.UnsafeMemory {
		members = append(
			members,
			target.unsafeRawProperty(),
			target.unsafeBindMethod(),
			target.unsafeMemoryMethod(),
			target.unsafeViewMethod(),
		)
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		target.id(className),
		[]tsgo.TypeParameterDeclaration{
			target.typeParameter("L", nil),
			target.typeParameter("S", nil),
		},
		nil,
		members,
	)
}

func CellValue(
	factory tsgo.Factory,
	runtimeName string,
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	pointer tsgo.Expression,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		Dereference(
			factory,
			runtimeName,
			logicalType,
			storageType,
			pointer,
		),
		nil,
		factory.Identifier(CellValueName),
		tsgo.NodeFlagsNone,
	)
}

func Dereference(
	factory tsgo.Factory,
	runtimeName string,
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	pointer tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(DereferenceName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{logicalType, storageType},
		[]tsgo.Expression{pointer},
		tsgo.NodeFlagsNone,
	)
}

func Direct(
	factory tsgo.Factory,
	runtimeName string,
	logicalType tsgo.TypeNode,
	pointer tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(DirectName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{logicalType},
		[]tsgo.Expression{pointer},
		tsgo.NodeFlagsNone,
	)
}

func Cell(
	factory tsgo.Factory,
	runtimeName string,
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(CellName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{logicalType, storageType},
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) typeParameter(
	name string,
	constraint tsgo.TypeNode,
) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id(name),
		constraint,
		nil,
		nil,
	)
}

func (b builder) typeReference(
	name string,
	arguments ...tsgo.TypeNode,
) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}

func (b builder) typeL() tsgo.TypeNode {
	return b.typeReference("L")
}

func (b builder) typeS() tsgo.TypeNode {
	return b.typeReference("S")
}

func (b builder) pointerType(
	logical tsgo.TypeNode,
	storage tsgo.TypeNode,
) tsgo.TypeNode {
	return b.typeReference(b.className, logical, storage)
}

func (b builder) objectType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindObjectKeyword,
	)
}

func (b builder) undefinedType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
	)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindBooleanKeyword,
	)
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
}

func (b builder) propertyKeyType() tsgo.TypeNode {
	return b.typeReference("PropertyKey")
}

func (b builder) addressKeyType() tsgo.TypeNode {
	return b.factory.UnionTypeNode(
		[]tsgo.TypeNode{
			b.propertyKeyType(),
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBigIntKeyword,
			),
		},
	)
}

func (b builder) parameter(
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.id(name),
		nil,
		targetType,
		nil,
	)
}

func (b builder) property(
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(
		receiver,
		nil,
		b.id(name),
		tsgo.NodeFlagsNone,
	)
}

func (b builder) call(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) binary(
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(operator),
		right,
	)
}

func (b builder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorEqualsToken, right)
}

func (b builder) undefined() tsgo.Expression {
	return b.factory.VoidExpression(
		b.factory.NumericLiteral("0", tsgo.TokenFlagsNone),
	)
}

func (b builder) variable(
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id(name),
					nil,
					targetType,
					value,
				),
			},
			flags,
		),
	)
}

func (b builder) method(
	modifiers []tsgo.ModifierLike,
	name string,
	typeParameters []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		typeParameters,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}
