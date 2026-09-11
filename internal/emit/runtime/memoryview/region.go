package memoryview

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	KindMember   = "kind"
	ValuesMember = "values"
	OffsetMember = "offset"
	LocateMember = "at"
	IndexedKind  = "indexed"
	PointerKind  = "pointer"
)

func RegionType(factory tsgo.Factory, element tsgo.TypeNode) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(factory.Identifier("GoStorageRegion"), []tsgo.TypeNode{element})
}

func CountType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}

func PointerType(factory tsgo.Factory, element tsgo.TypeNode) tsgo.TypeNode {
	return factory.TypeReferenceNode(factory.Identifier("Pointer"), []tsgo.TypeNode{element})
}

func BuildRegion(factory tsgo.Factory, name string) tsgo.Statement {
	element := factory.TypeReferenceNode(factory.Identifier("Element"), nil)
	field := func(name string, fieldType tsgo.TypeNode) tsgo.TypeElement {
		return factory.PropertySignatureDeclaration([]tsgo.ModifierLike{factory.ReadonlyKeyword()}, factory.Identifier(name), nil, fieldType, factory.OmittedExpression())
	}
	kind := func(value string) tsgo.TypeElement {
		return field(KindMember, factory.LiteralTypeNode(factory.StringLiteral(value, tsgo.TokenFlagsNone)))
	}
	indexed := factory.TypeLiteralNode([]tsgo.TypeElement{
		kind(IndexedKind), field(ValuesMember, factory.ArrayTypeNode(element)),
		field(OffsetMember, factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)),
	})
	pointer := factory.TypeLiteralNode([]tsgo.TypeElement{
		kind(PointerKind), field(OffsetMember, CountType(factory)),
		field(LocateMember, factory.FunctionTypeNode(nil, []tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(nil, nil, factory.Identifier("index"), nil, CountType(factory), nil),
		}, PointerType(factory, element))),
	})
	return factory.TypeAliasDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{factory.TypeParameterDeclaration(nil, factory.Identifier("Element"), nil, nil, nil)},
		factory.UnionTypeNode([]tsgo.TypeNode{indexed, pointer}))
}

func Indexed(factory tsgo.Factory, element tsgo.TypeNode, values, offset tsgo.Expression) tsgo.Expression {
	return factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		property(factory, KindMember, factory.LiteralTypeNode(factory.StringLiteral(IndexedKind, tsgo.TokenFlagsNone)), factory.StringLiteral(IndexedKind, tsgo.TokenFlagsNone)),
		property(factory, ValuesMember, factory.ArrayTypeNode(element), values),
		property(factory, OffsetMember, factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword), offset),
	}, false)
}

func Pointer(factory tsgo.Factory, element tsgo.TypeNode, at, offset tsgo.Expression) tsgo.Expression {
	locationType := factory.FunctionTypeNode(nil, []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, factory.Identifier("index"), nil, CountType(factory), nil),
	}, PointerType(factory, element))
	return factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		property(factory, KindMember, factory.LiteralTypeNode(factory.StringLiteral(PointerKind, tsgo.TokenFlagsNone)), factory.StringLiteral(PointerKind, tsgo.TokenFlagsNone)),
		property(factory, LocateMember, locationType, at), property(factory, OffsetMember, CountType(factory), offset),
	}, false)
}

func property(factory tsgo.Factory, name string, valueType tsgo.TypeNode, value tsgo.Expression) tsgo.ObjectLiteralElementLike {
	return factory.PropertyAssignment(nil, factory.Identifier(name), nil, valueType, value)
}
