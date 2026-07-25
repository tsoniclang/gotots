package tsgo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncoderPreservesAbsentVersusEmptyNodeList(t *testing.T) {
	factory := NewFactory()
	declarations := factory.VariableDeclarationList(nil, NodeFlagsNone)
	absent, err := EncodeNode(factory.VariableStatement(nil, declarations))
	if err != nil {
		t.Fatal(err)
	}
	empty, err := EncodeNode(factory.VariableStatement([]ModifierLike{}, declarations))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(absent, empty) {
		t.Fatal("absent and present-empty modifier lists encoded identically")
	}
	assertPinnedEncoding(t, absent, `
const declarations = factory.createVariableDeclarationList([], 0);
const node = factory.createVariableStatement(undefined, declarations);
process.stdout.write(Buffer.from(encodeNode(node)).toString("base64"));
`)
	assertPinnedEncoding(t, empty, `
const declarations = factory.createVariableDeclarationList([], 0);
const node = factory.createVariableStatement([], declarations);
process.stdout.write(Buffer.from(encodeNode(node)).toString("base64"));
`)
}

func TestEncoderMatchesDynamicJSDocChildOrder(t *testing.T) {
	for _, nameFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "type-first", true: "name-first"}[nameFirst], func(t *testing.T) {
			factory := NewFactory()
			node := factory.JSDocParameterTag(
				factory.Identifier("param"),
				factory.Identifier("value"),
				false,
				factory.KeywordTypeNode(KeywordTypeSyntaxKindStringKeyword),
				nameFirst,
				nil,
			)
			actual, err := EncodeNode(node)
			if err != nil {
				t.Fatal(err)
			}
			boolean := map[bool]string{false: "false", true: "true"}[nameFirst]
			assertPinnedEncoding(t, actual, `
const node = factory.createJSDocParameterTag(
    factory.createIdentifier("param"),
    factory.createIdentifier("value"),
    false,
    factory.createKeywordTypeNode(SyntaxKind.StringKeyword),
    `+boolean+`,
    undefined,
);
process.stdout.write(Buffer.from(encodeNode(node)).toString("base64"));
`)
		})
	}
}

func TestRawChildArraysDoNotCreateNodeLists(t *testing.T) {
	factory := NewFactory()
	encoded, err := EncodeNode(factory.SyntaxList([]Node{
		factory.Identifier("left"),
		factory.Identifier("right"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	nodesOffset := binary.LittleEndian.Uint32(encoded[headerOffsetNodes:])
	kindAt := func(index uint32) uint32 {
		offset := nodesOffset + index*nodeLen + nodeOffsetKind
		return binary.LittleEndian.Uint32(encoded[offset:])
	}
	if actual := kindAt(2); actual != uint32(SyntaxKindIdentifier) {
		t.Fatalf("first raw child kind = %d, want Identifier", actual)
	}
	if actual := kindAt(3); actual != uint32(SyntaxKindIdentifier) {
		t.Fatalf("second raw child kind = %d, want Identifier", actual)
	}
}

func assertPinnedEncoding(t *testing.T, actual []byte, script string) {
	t.Helper()
	expected := encodeWithPinnedScript(t, script)
	if !bytes.Equal(actual, expected) {
		offset := firstDifference(actual, expected)
		t.Fatalf(
			"encoding differs at %d (Go=%d bytes, upstream=%d bytes)\nGo: % x\nupstream: % x",
			offset,
			len(actual),
			len(expected),
			window(actual, offset),
			window(expected, offset),
		)
	}
}
