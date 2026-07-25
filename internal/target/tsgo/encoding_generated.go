// Code generated from schema/tsgo by go generate. DO NOT EDIT.

package tsgo

import "strings"

func (*assignmentOperatorTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{dataType: nodeDataChildren}
}

func (*binaryOperatorTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{dataType: nodeDataChildren}
}

func (*keywordTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{dataType: nodeDataChildren}
}

func (*tokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{dataType: nodeDataChildren}
}

func (n *endOfFileNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *numericLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedLiteral,
		tokenFlags: n.tokenFlags,
	}
}

func (n *bigIntLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedLiteral,
		tokenFlags: n.tokenFlags,
	}
}

func (n *stringLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedLiteral,
		tokenFlags: n.tokenFlags,
	}
}

func (n *jsxTextNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataString,
		commonData: boolBit(n.containsOnlyTriviaWhiteSpaces) << 24,
		text:       n.text,
	}
}

func (n *regularExpressionLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedLiteral,
		tokenFlags: n.tokenFlags,
	}
}

func (n *noSubstitutionTemplateLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedLiteral,
		tokenFlags: n.templateFlags,
	}
}

func (n *templateHeadNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedTemplate,
		rawText:    n.rawText,
		tokenFlags: n.templateFlags,
	}
}

func (n *templateMiddleNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedTemplate,
		rawText:    n.rawText,
		tokenFlags: n.templateFlags,
	}
}

func (n *templateTailNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataExtended,
		text:       n.text,
		extended:   extendedTemplate,
		rawText:    n.rawText,
		tokenFlags: n.templateFlags,
	}
}

func (n *dotTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *dotDotDotTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *questionDotTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *equalsGreaterThanTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *plusTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *minusTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *asteriskTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *exclamationTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *questionTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *colonTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *equalsTokenNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *identifierNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		text:     n.text,
	}
}

func (n *privateIdentifierNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		text:     n.text,
	}
}

func (n *caseKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *constKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *defaultKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *exportKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *falseLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *importExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *inKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *nullLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *superExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *thisExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *trueLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *privateKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *protectedKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *publicKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *staticKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *abstractKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *accessorKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *assertsKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *assertKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *asyncKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *awaitKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *declareKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *outKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *readonlyKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *overrideKeywordNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *qualifiedNameNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Left", present: n.left != nil, required: true, node: n.left},
			{name: "Right", present: n.right != nil, required: true, node: n.right},
		},
	}
}

func (n *computedPropertyNameNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *typeParameterDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Constraint", present: n.constraint != nil, required: false, node: n.constraint},
			{name: "Expression", present: n.expression != nil, required: false, node: n.expression},
			{name: "DefaultType", present: n.defaultType != nil, required: false, node: n.defaultType},
		},
	}
}

func (n *parameterDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "DotDotDotToken", present: n.dotDotDotToken != nil, required: false, node: n.dotDotDotToken},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "QuestionToken", present: n.questionToken != nil, required: false, node: n.questionToken},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *decoratorNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *propertySignatureDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "Initializer", present: n.initializer != nil, required: true, node: n.initializer},
		},
	}
}

func (n *propertyDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *methodSignatureDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *methodDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "AsteriskToken", present: n.asteriskToken != nil, required: false, node: n.asteriskToken},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *classStaticBlockDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "Body", present: n.body != nil, required: true, node: n.body},
		},
	}
}

func (n *constructorDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *getAccessorDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *setAccessorDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *callSignatureDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *constructSignatureDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *indexSignatureDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *typePredicateNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "AssertsModifier", present: n.assertsModifier != nil, required: false, node: n.assertsModifier},
			{name: "ParameterName", present: n.parameterName != nil, required: true, node: n.parameterName},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *typeReferenceNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeName", present: n.typeName != nil, required: true, node: n.typeName},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
		},
	}
}

func (n *functionTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *constructorTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *typeQueryNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "ExprName", present: n.exprName != nil, required: true, node: n.exprName},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
		},
	}
}

func (n *typeLiteralNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Members", present: true, required: true, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *arrayTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "ElementType", present: n.elementType != nil, required: true, node: n.elementType},
		},
	}
}

func (n *tupleTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *optionalTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *restTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *unionTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Types", present: true, required: true, raw: false, nodes: nodesOf(n.types)},
		},
	}
}

func (n *intersectionTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Types", present: true, required: true, raw: false, nodes: nodesOf(n.types)},
		},
	}
}

func (n *conditionalTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "CheckType", present: n.checkType != nil, required: true, node: n.checkType},
			{name: "ExtendsType", present: n.extendsType != nil, required: true, node: n.extendsType},
			{name: "TrueType", present: n.trueType != nil, required: true, node: n.trueType},
			{name: "FalseType", present: n.falseType != nil, required: true, node: n.falseType},
		},
	}
}

func (n *inferTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeParameter", present: n.typeParameter != nil, required: true, node: n.typeParameter},
		},
	}
}

func (n *parenthesizedTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *thisTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *typeOperatorNodeNode) targetEncoding() nodeEncoding {
	var operatorIndex uint32
	switch SyntaxKind(n.operator) {
	case SyntaxKindReadonlyKeyword:
		operatorIndex = 1
	case SyntaxKindUniqueKeyword:
		operatorIndex = 2
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: operatorIndex << 24,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *indexedAccessTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "ObjectType", present: n.objectType != nil, required: true, node: n.objectType},
			{name: "IndexType", present: n.indexType != nil, required: true, node: n.indexType},
		},
	}
}

func (n *mappedTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "ReadonlyToken", present: n.readonlyToken != nil, required: false, node: n.readonlyToken},
			{name: "TypeParameter", present: n.typeParameter != nil, required: true, node: n.typeParameter},
			{name: "NameType", present: n.nameType != nil, required: false, node: n.nameType},
			{name: "QuestionToken", present: n.questionToken != nil, required: false, node: n.questionToken},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Members", present: n.members != nil, required: false, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *literalTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Literal", present: n.literal != nil, required: true, node: n.literal},
		},
	}
}

func (n *namedTupleMemberNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "DotDotDotToken", present: n.dotDotDotToken != nil, required: false, node: n.dotDotDotToken},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "QuestionToken", present: n.questionToken != nil, required: false, node: n.questionToken},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *templateLiteralTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Head", present: n.head != nil, required: true, node: n.head},
			{name: "TemplateSpans", present: true, required: true, raw: false, nodes: nodesOf(n.templateSpans)},
		},
	}
}

func (n *templateLiteralTypeSpanNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "Literal", present: n.literal != nil, required: true, node: n.literal},
		},
	}
}

func (n *importTypeNodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isTypeOf) << 24,
		children: []childEncoding{
			{name: "Argument", present: n.argument != nil, required: true, node: n.argument},
			{name: "Attributes", present: n.attributes != nil, required: false, node: n.attributes},
			{name: "Qualifier", present: n.qualifier != nil, required: false, node: n.qualifier},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
		},
	}
}

func (n *objectBindingPatternNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *arrayBindingPatternNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *bindingElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "DotDotDotToken", present: n.dotDotDotToken != nil, required: false, node: n.dotDotDotToken},
			{name: "PropertyName", present: n.propertyName != nil, required: false, node: n.propertyName},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *arrayLiteralExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.multiLine) << 24,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *objectLiteralExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.multiLine) << 24,
		children: []childEncoding{
			{name: "Properties", present: true, required: true, raw: false, nodes: nodesOf(n.properties)},
		},
	}
}

func (n *propertyAccessExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "QuestionDotToken", present: n.questionDotToken != nil, required: false, node: n.questionDotToken},
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *elementAccessExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "QuestionDotToken", present: n.questionDotToken != nil, required: false, node: n.questionDotToken},
			{name: "ArgumentExpression", present: n.argumentExpression != nil, required: true, node: n.argumentExpression},
		},
	}
}

func (n *callExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "QuestionDotToken", present: n.questionDotToken != nil, required: false, node: n.questionDotToken},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
			{name: "Arguments", present: true, required: true, raw: false, nodes: nodesOf(n.arguments)},
		},
	}
}

func (n *newExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
			{name: "Arguments", present: n.arguments != nil, required: false, raw: false, nodes: nodesOf(n.arguments)},
		},
	}
}

func (n *taggedTemplateExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Tag", present: n.tag != nil, required: true, node: n.tag},
			{name: "QuestionDotToken", present: n.questionDotToken != nil, required: true, node: n.questionDotToken},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
			{name: "Template", present: n.template != nil, required: true, node: n.template},
		},
	}
}

func (n *typeAssertionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *parenthesizedExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *functionExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "AsteriskToken", present: n.asteriskToken != nil, required: false, node: n.asteriskToken},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: true, node: n.body},
		},
	}
}

func (n *arrowFunctionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "EqualsGreaterThanToken", present: n.equalsGreaterThanToken != nil, required: true, node: n.equalsGreaterThanToken},
			{name: "Body", present: n.body != nil, required: true, node: n.body},
		},
	}
}

func (n *deleteExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *typeOfExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *voidExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *awaitExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *prefixUnaryExpressionNode) targetEncoding() nodeEncoding {
	var operatorIndex uint32
	switch SyntaxKind(n.operator) {
	case SyntaxKindMinusToken:
		operatorIndex = 1
	case SyntaxKindTildeToken:
		operatorIndex = 2
	case SyntaxKindExclamationToken:
		operatorIndex = 3
	case SyntaxKindPlusPlusToken:
		operatorIndex = 4
	case SyntaxKindMinusMinusToken:
		operatorIndex = 5
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: operatorIndex << 24,
		children: []childEncoding{
			{name: "Operand", present: n.operand != nil, required: true, node: n.operand},
		},
	}
}

func (n *postfixUnaryExpressionNode) targetEncoding() nodeEncoding {
	var operatorIndex uint32
	switch SyntaxKind(n.operator) {
	case SyntaxKindMinusMinusToken:
		operatorIndex = 1
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: operatorIndex << 24,
		children: []childEncoding{
			{name: "Operand", present: n.operand != nil, required: true, node: n.operand},
		},
	}
}

func (n *binaryExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "Left", present: n.left != nil, required: true, node: n.left},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "OperatorToken", present: n.operatorToken != nil, required: true, node: n.operatorToken},
			{name: "Right", present: n.right != nil, required: true, node: n.right},
		},
	}
}

func (n *conditionalExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Condition", present: n.condition != nil, required: true, node: n.condition},
			{name: "QuestionToken", present: n.questionToken != nil, required: true, node: n.questionToken},
			{name: "WhenTrue", present: n.whenTrue != nil, required: true, node: n.whenTrue},
			{name: "ColonToken", present: n.colonToken != nil, required: true, node: n.colonToken},
			{name: "WhenFalse", present: n.whenFalse != nil, required: true, node: n.whenFalse},
		},
	}
}

func (n *templateExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Head", present: n.head != nil, required: true, node: n.head},
			{name: "TemplateSpans", present: true, required: true, raw: false, nodes: nodesOf(n.templateSpans)},
		},
	}
}

func (n *yieldExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "AsteriskToken", present: n.asteriskToken != nil, required: false, node: n.asteriskToken},
			{name: "Expression", present: n.expression != nil, required: false, node: n.expression},
		},
	}
}

func (n *spreadElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *classExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "HeritageClauses", present: n.heritageClauses != nil, required: false, raw: false, nodes: nodesOf(n.heritageClauses)},
			{name: "Members", present: true, required: true, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *omittedExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *expressionWithTypeArgumentsNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
		},
	}
}

func (n *asExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *nonNullExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *metaPropertyNode) targetEncoding() nodeEncoding {
	var keywordTokenIndex uint32
	switch SyntaxKind(n.keywordToken) {
	case SyntaxKindNewKeyword:
		keywordTokenIndex = 1
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: keywordTokenIndex << 24,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *syntheticExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataExtended,
		children: []childEncoding{
			{name: "TupleNameSource", present: n.tupleNameSource != nil, required: false, node: n.tupleNameSource},
		},
		extended: extendedUnsupported,
	}
}

func (n *satisfiesExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *templateSpanNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Literal", present: n.literal != nil, required: true, node: n.literal},
		},
	}
}

func (n *semicolonClassElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *blockNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.multiLine) << 24,
		children: []childEncoding{
			{name: "Statements", present: true, required: true, raw: false, nodes: nodesOf(n.statements)},
		},
	}
}

func (n *emptyStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *variableStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "DeclarationList", present: n.declarationList != nil, required: true, node: n.declarationList},
		},
	}
}

func (n *expressionStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *ifStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "ThenStatement", present: n.thenStatement != nil, required: true, node: n.thenStatement},
			{name: "ElseStatement", present: n.elseStatement != nil, required: false, node: n.elseStatement},
		},
	}
}

func (n *doStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *whileStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *forStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
			{name: "Condition", present: n.condition != nil, required: false, node: n.condition},
			{name: "Incrementor", present: n.incrementor != nil, required: false, node: n.incrementor},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *forInStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "AwaitModifier", present: n.awaitModifier != nil, required: false, node: n.awaitModifier},
			{name: "Initializer", present: n.initializer != nil, required: true, node: n.initializer},
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *forOfStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "AwaitModifier", present: n.awaitModifier != nil, required: false, node: n.awaitModifier},
			{name: "Initializer", present: n.initializer != nil, required: true, node: n.initializer},
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *continueStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Label", present: n.label != nil, required: false, node: n.label},
		},
	}
}

func (n *breakStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Label", present: n.label != nil, required: false, node: n.label},
		},
	}
}

func (n *returnStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: false, node: n.expression},
		},
	}
}

func (n *withStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *switchStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "CaseBlock", present: n.caseBlock != nil, required: true, node: n.caseBlock},
		},
	}
}

func (n *labeledStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Label", present: n.label != nil, required: true, node: n.label},
			{name: "Statement", present: n.statement != nil, required: true, node: n.statement},
		},
	}
}

func (n *throwStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *tryStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TryBlock", present: n.tryBlock != nil, required: true, node: n.tryBlock},
			{name: "CatchClause", present: n.catchClause != nil, required: false, node: n.catchClause},
			{name: "FinallyBlock", present: n.finallyBlock != nil, required: false, node: n.finallyBlock},
		},
	}
}

func (n *debuggerStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *variableDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "ExclamationToken", present: n.exclamationToken != nil, required: false, node: n.exclamationToken},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *variableDeclarationListNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Declarations", present: true, required: true, raw: false, nodes: nodesOf(n.declarations)},
		},
	}
}

func (n *functionDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "AsteriskToken", present: n.asteriskToken != nil, required: false, node: n.asteriskToken},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *classDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "HeritageClauses", present: n.heritageClauses != nil, required: false, raw: false, nodes: nodesOf(n.heritageClauses)},
			{name: "Members", present: true, required: true, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *interfaceDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "HeritageClauses", present: n.heritageClauses != nil, required: false, raw: false, nodes: nodesOf(n.heritageClauses)},
			{name: "Members", present: true, required: true, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *typeAliasDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *enumDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Members", present: true, required: true, raw: false, nodes: nodesOf(n.members)},
		},
	}
}

func (n *moduleDeclarationNode) targetEncoding() nodeEncoding {
	var keywordIndex uint32
	switch SyntaxKind(n.keyword) {
	case SyntaxKindNamespaceKeyword:
		keywordIndex = 1
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: keywordIndex << 24,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Body", present: n.body != nil, required: false, node: n.body},
		},
	}
}

func (n *moduleBlockNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Statements", present: true, required: true, raw: false, nodes: nodesOf(n.statements)},
		},
	}
}

func (n *caseBlockNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Clauses", present: true, required: true, raw: false, nodes: nodesOf(n.clauses)},
		},
	}
}

func (n *namespaceExportDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *importEqualsDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isTypeOnly) << 24,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "ModuleReference", present: n.moduleReference != nil, required: true, node: n.moduleReference},
		},
	}
}

func (n *importDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "ImportClause", present: n.importClause != nil, required: false, node: n.importClause},
			{name: "ModuleSpecifier", present: n.moduleSpecifier != nil, required: true, node: n.moduleSpecifier},
			{name: "Attributes", present: n.attributes != nil, required: false, node: n.attributes},
		},
	}
}

func (n *importClauseNode) targetEncoding() nodeEncoding {
	var phaseModifierIndex uint32
	switch SyntaxKind(n.phaseModifier) {
	case SyntaxKindTypeKeyword:
		phaseModifierIndex = 1
	case SyntaxKindDeferKeyword:
		phaseModifierIndex = 2
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: phaseModifierIndex << 24,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "NamedBindings", present: n.namedBindings != nil, required: false, node: n.namedBindings},
		},
	}
}

func (n *namespaceImportNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *namedImportsNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *importSpecifierNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isTypeOnly) << 24,
		children: []childEncoding{
			{name: "PropertyName", present: n.propertyName != nil, required: false, node: n.propertyName},
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *exportAssignmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isExportEquals) << 24,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *exportDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isTypeOnly) << 24,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "ExportClause", present: n.exportClause != nil, required: false, node: n.exportClause},
			{name: "ModuleSpecifier", present: n.moduleSpecifier != nil, required: false, node: n.moduleSpecifier},
			{name: "Attributes", present: n.attributes != nil, required: false, node: n.attributes},
		},
	}
}

func (n *namedExportsNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Elements", present: true, required: true, raw: false, nodes: nodesOf(n.elements)},
		},
	}
}

func (n *namespaceExportNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *exportSpecifierNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isTypeOnly) << 24,
		children: []childEncoding{
			{name: "PropertyName", present: n.propertyName != nil, required: false, node: n.propertyName},
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *missingDeclarationNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
		},
	}
}

func (n *externalModuleReferenceNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *jsxElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "OpeningElement", present: n.openingElement != nil, required: true, node: n.openingElement},
			{name: "Children", present: true, required: true, raw: false, nodes: nodesOf(n.children)},
			{name: "ClosingElement", present: n.closingElement != nil, required: true, node: n.closingElement},
		},
	}
}

func (n *jsxSelfClosingElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
			{name: "Attributes", present: n.attributes != nil, required: true, node: n.attributes},
		},
	}
}

func (n *jsxOpeningElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeArguments", present: n.typeArguments != nil, required: false, raw: false, nodes: nodesOf(n.typeArguments)},
			{name: "Attributes", present: n.attributes != nil, required: true, node: n.attributes},
		},
	}
}

func (n *jsxClosingElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
		},
	}
}

func (n *jsxFragmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "OpeningFragment", present: n.openingFragment != nil, required: true, node: n.openingFragment},
			{name: "Children", present: true, required: true, raw: false, nodes: nodesOf(n.children)},
			{name: "ClosingFragment", present: n.closingFragment != nil, required: true, node: n.closingFragment},
		},
	}
}

func (n *jsxOpeningFragmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *jsxClosingFragmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *jsxAttributeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *jsxAttributesNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Properties", present: true, required: true, raw: false, nodes: nodesOf(n.properties)},
		},
	}
}

func (n *jsxSpreadAttributeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *jsxExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "DotDotDotToken", present: n.dotDotDotToken != nil, required: false, node: n.dotDotDotToken},
			{name: "Expression", present: n.expression != nil, required: false, node: n.expression},
		},
	}
}

func (n *jsxNamespacedNameNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Namespace", present: n.namespace != nil, required: true, node: n.namespace},
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *caseClauseNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statements", present: true, required: true, raw: false, nodes: nodesOf(n.statements)},
		},
	}
}

func (n *defaultClauseNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "Statements", present: true, required: true, raw: false, nodes: nodesOf(n.statements)},
		},
	}
}

func (n *heritageClauseNode) targetEncoding() nodeEncoding {
	var tokenIndex uint32
	switch SyntaxKind(n.token) {
	case SyntaxKindImplementsKeyword:
		tokenIndex = 1
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: tokenIndex << 24,
		children: []childEncoding{
			{name: "Types", present: true, required: true, raw: false, nodes: nodesOf(n.types)},
		},
	}
}

func (n *catchClauseNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "VariableDeclaration", present: n.variableDeclaration != nil, required: false, node: n.variableDeclaration},
			{name: "Block", present: n.block != nil, required: true, node: n.block},
		},
	}
}

func (n *importAttributesNode) targetEncoding() nodeEncoding {
	var tokenIndex uint32
	switch SyntaxKind(n.token) {
	case SyntaxKindAssertKeyword:
		tokenIndex = 1
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: tokenIndex<<24 | boolBit(n.multiLine)<<25,
		children: []childEncoding{
			{name: "Attributes", present: true, required: true, raw: false, nodes: nodesOf(n.attributes)},
		},
	}
}

func (n *importAttributeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Value", present: n.value != nil, required: true, node: n.value},
		},
	}
}

func (n *propertyAssignmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "Initializer", present: n.initializer != nil, required: true, node: n.initializer},
		},
	}
}

func (n *shorthandPropertyAssignmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "modifiers", present: n.modifiers != nil, required: false, raw: false, nodes: nodesOf(n.modifiers)},
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "PostfixToken", present: n.postfixToken != nil, required: false, node: n.postfixToken},
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
			{name: "EqualsToken", present: n.equalsToken != nil, required: false, node: n.equalsToken},
			{name: "ObjectAssignmentInitializer", present: n.objectAssignmentInitializer != nil, required: false, node: n.objectAssignmentInitializer},
		},
	}
}

func (n *spreadAssignmentNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *enumMemberNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
			{name: "Initializer", present: n.initializer != nil, required: false, node: n.initializer},
		},
	}
}

func (n *sourceFileNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataExtended,
		children: []childEncoding{
			{name: "Statements", present: true, required: true, raw: false, nodes: nodesOf(n.statements)},
			{name: "EndOfFileToken", present: n.endOfFileToken != nil, required: true, node: n.endOfFileToken},
		},
		extended:   extendedSourceFile,
		sourceFile: cloneSourceFileData(n.sourceData),
	}
}

func (n *jsDocTypeExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *jsDocNameReferenceNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: true, node: n.name},
		},
	}
}

func (n *jsDocAllTypeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *jsDocNullableTypeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *jsDocNonNullableTypeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *jsDocOptionalTypeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *jsDocVariadicTypeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Type", present: n.typeNode != nil, required: true, node: n.typeNode},
		},
	}
}

func (n *jsDocNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Comment", present: true, required: true, raw: false, nodes: nodesOf(n.comment)},
			{name: "Tags", present: n.tags != nil, required: false, raw: false, nodes: nodesOf(n.tags)},
		},
	}
}

func (n *jsDocTextNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		text:     strings.Join(n.text, ""),
	}
}

func (n *jsDocTypeLiteralNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isArrayType) << 24,
		children: []childEncoding{
			{name: "JSDocPropertyTags", present: len(n.jSDocPropertyTags) != 0, required: false, raw: true, nodes: nodesOf(n.jSDocPropertyTags)},
		},
	}
}

func (n *jsDocSignatureNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TypeParameters", present: n.typeParameters != nil, required: false, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Parameters", present: true, required: true, raw: false, nodes: nodesOf(n.parameters)},
			{name: "Type", present: n.typeNode != nil, required: false, node: n.typeNode},
		},
	}
}

func (n *jsDocLinkNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: false, node: n.name},
		},
		text: strings.Join(n.text, ""),
	}
}

func (n *jsDocLinkCodeNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: false, node: n.name},
		},
		text: strings.Join(n.text, ""),
	}
}

func (n *jsDocLinkPlainNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataString,
		children: []childEncoding{
			{name: "name", present: n.name != nil, required: false, node: n.name},
		},
		text: strings.Join(n.text, ""),
	}
}

func (n *jsDocUnknownTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocAugmentsTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "ClassName", present: n.className != nil, required: true, node: n.className},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocImplementsTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "ClassName", present: n.className != nil, required: true, node: n.className},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocDeprecatedTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocPublicTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocPrivateTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocProtectedTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocReadonlyTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocOverrideTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocCallbackTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: true, node: n.typeExpression},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocOverloadTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: true, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocParameterTagNode) targetEncoding() nodeEncoding {
	children := []childEncoding{
		{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
		{name: "name", present: n.name != nil, required: true, node: n.name},
		{name: "TypeExpression", present: n.typeExpression != nil, required: false, node: n.typeExpression},
		{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
	}
	if !n.isNameFirst {
		children[1], children[2] = children[2], children[1]
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isBracketed)<<24 | boolBit(n.isNameFirst)<<25,
		children:   children,
	}
}

func (n *jsDocReturnTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: false, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocThisTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: true, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocTypeTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: true, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocTemplateTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "Constraint", present: n.constraint != nil, required: true, node: n.constraint},
			{name: "TypeParameters", present: true, required: true, raw: false, nodes: nodesOf(n.typeParameters)},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocTypedefTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: false, node: n.typeExpression},
			{name: "name", present: n.name != nil, required: false, node: n.name},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocSeeTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "NameExpression", present: n.nameExpression != nil, required: true, node: n.nameExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocPropertyTagNode) targetEncoding() nodeEncoding {
	children := []childEncoding{
		{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
		{name: "name", present: n.name != nil, required: true, node: n.name},
		{name: "TypeExpression", present: n.typeExpression != nil, required: false, node: n.typeExpression},
		{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
	}
	if !n.isNameFirst {
		children[1], children[2] = children[2], children[1]
	}
	return nodeEncoding{
		dataType:   nodeDataChildren,
		commonData: boolBit(n.isBracketed)<<24 | boolBit(n.isNameFirst)<<25,
		children:   children,
	}
}

func (n *jsDocThrowsTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: false, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocSatisfiesTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "TypeExpression", present: n.typeExpression != nil, required: true, node: n.typeExpression},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *jsDocImportTagNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "TagName", present: n.tagName != nil, required: true, node: n.tagName},
			{name: "ImportClause", present: n.importClause != nil, required: false, node: n.importClause},
			{name: "ModuleSpecifier", present: n.moduleSpecifier != nil, required: true, node: n.moduleSpecifier},
			{name: "Attributes", present: n.attributes != nil, required: false, node: n.attributes},
			{name: "Comment", present: n.comment != nil, required: false, raw: false, nodes: nodesOf(n.comment)},
		},
	}
}

func (n *syntaxListNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Children", present: len(n.children) != 0, required: false, raw: true, nodes: nodesOf(n.children)},
		},
	}
}

func (n *notEmittedStatementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}

func (n *partiallyEmittedExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
		},
	}
}

func (n *syntheticReferenceExpressionNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
		children: []childEncoding{
			{name: "Expression", present: n.expression != nil, required: true, node: n.expression},
			{name: "ThisArg", present: n.thisArg != nil, required: true, node: n.thisArg},
		},
	}
}

func (n *notEmittedTypeElementNode) targetEncoding() nodeEncoding {
	return nodeEncoding{
		dataType: nodeDataChildren,
	}
}
