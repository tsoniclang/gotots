// Code generated from schema/tsgo by go generate. DO NOT EDIT.

package tsgo

func (Factory) Token(kind TokenSyntaxKind) Token {
	return &tokenNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}
}

func (Factory) KeywordTypeNode(kind KeywordTypeSyntaxKind) KeywordTypeNode {
	return &keywordTypeNodeNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}
}

func (Factory) AssignmentOperatorToken(kind AssignmentOperator) AssignmentOperatorToken {
	return &assignmentOperatorTokenNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}
}

func (Factory) BinaryOperatorToken(kind BinaryOperator) BinaryOperatorToken {
	return &binaryOperatorTokenNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}
}

func (Factory) EndOfFile() EndOfFile {
	return &endOfFileNode{
		nodeCore: newNodeCore(SyntaxKindEndOfFile, NodeFlagsNone),
	}
}

func (Factory) NumericLiteral(text string, tokenFlags TokenFlags) NumericLiteral {
	return &numericLiteralNode{
		nodeCore:   newNodeCore(SyntaxKindNumericLiteral, NodeFlagsNone),
		text:       text,
		tokenFlags: tokenFlags,
	}
}

func (Factory) BigIntLiteral(text string, tokenFlags TokenFlags) BigIntLiteral {
	return &bigIntLiteralNode{
		nodeCore:   newNodeCore(SyntaxKindBigIntLiteral, NodeFlagsNone),
		text:       text,
		tokenFlags: tokenFlags,
	}
}

func (Factory) StringLiteral(text string, tokenFlags TokenFlags) StringLiteral {
	return &stringLiteralNode{
		nodeCore:   newNodeCore(SyntaxKindStringLiteral, NodeFlagsNone),
		text:       text,
		tokenFlags: tokenFlags,
	}
}

func (Factory) JsxText(text string, containsOnlyTriviaWhiteSpaces bool) JsxText {
	return &jsxTextNode{
		nodeCore:                      newNodeCore(SyntaxKindJsxText, NodeFlagsNone),
		text:                          text,
		containsOnlyTriviaWhiteSpaces: containsOnlyTriviaWhiteSpaces,
	}
}

func (Factory) RegularExpressionLiteral(text string, tokenFlags TokenFlags) RegularExpressionLiteral {
	return &regularExpressionLiteralNode{
		nodeCore:   newNodeCore(SyntaxKindRegularExpressionLiteral, NodeFlagsNone),
		text:       text,
		tokenFlags: tokenFlags,
	}
}

func (Factory) NoSubstitutionTemplateLiteral(text string, templateFlags TokenFlags) NoSubstitutionTemplateLiteral {
	return &noSubstitutionTemplateLiteralNode{
		nodeCore:      newNodeCore(SyntaxKindNoSubstitutionTemplateLiteral, NodeFlagsNone),
		text:          text,
		templateFlags: templateFlags,
	}
}

func (Factory) TemplateHead(text string, rawText string, templateFlags TokenFlags) TemplateHead {
	return &templateHeadNode{
		nodeCore:      newNodeCore(SyntaxKindTemplateHead, NodeFlagsNone),
		text:          text,
		rawText:       rawText,
		templateFlags: templateFlags,
	}
}

func (Factory) TemplateMiddle(text string, rawText string, templateFlags TokenFlags) TemplateMiddle {
	return &templateMiddleNode{
		nodeCore:      newNodeCore(SyntaxKindTemplateMiddle, NodeFlagsNone),
		text:          text,
		rawText:       rawText,
		templateFlags: templateFlags,
	}
}

func (Factory) TemplateTail(text string, rawText string, templateFlags TokenFlags) TemplateTail {
	return &templateTailNode{
		nodeCore:      newNodeCore(SyntaxKindTemplateTail, NodeFlagsNone),
		text:          text,
		rawText:       rawText,
		templateFlags: templateFlags,
	}
}

func (Factory) DotToken() DotToken {
	return &dotTokenNode{
		nodeCore: newNodeCore(SyntaxKindDotToken, NodeFlagsNone),
	}
}

func (Factory) DotDotDotToken() DotDotDotToken {
	return &dotDotDotTokenNode{
		nodeCore: newNodeCore(SyntaxKindDotDotDotToken, NodeFlagsNone),
	}
}

func (Factory) QuestionDotToken() QuestionDotToken {
	return &questionDotTokenNode{
		nodeCore: newNodeCore(SyntaxKindQuestionDotToken, NodeFlagsNone),
	}
}

func (Factory) EqualsGreaterThanToken() EqualsGreaterThanToken {
	return &equalsGreaterThanTokenNode{
		nodeCore: newNodeCore(SyntaxKindEqualsGreaterThanToken, NodeFlagsNone),
	}
}

func (Factory) PlusToken() PlusToken {
	return &plusTokenNode{
		nodeCore: newNodeCore(SyntaxKindPlusToken, NodeFlagsNone),
	}
}

func (Factory) MinusToken() MinusToken {
	return &minusTokenNode{
		nodeCore: newNodeCore(SyntaxKindMinusToken, NodeFlagsNone),
	}
}

func (Factory) AsteriskToken() AsteriskToken {
	return &asteriskTokenNode{
		nodeCore: newNodeCore(SyntaxKindAsteriskToken, NodeFlagsNone),
	}
}

func (Factory) ExclamationToken() ExclamationToken {
	return &exclamationTokenNode{
		nodeCore: newNodeCore(SyntaxKindExclamationToken, NodeFlagsNone),
	}
}

func (Factory) QuestionToken() QuestionToken {
	return &questionTokenNode{
		nodeCore: newNodeCore(SyntaxKindQuestionToken, NodeFlagsNone),
	}
}

func (Factory) ColonToken() ColonToken {
	return &colonTokenNode{
		nodeCore: newNodeCore(SyntaxKindColonToken, NodeFlagsNone),
	}
}

func (Factory) EqualsToken() EqualsToken {
	return &equalsTokenNode{
		nodeCore: newNodeCore(SyntaxKindEqualsToken, NodeFlagsNone),
	}
}

func (Factory) Identifier(text string) Identifier {
	return &identifierNode{
		nodeCore: newNodeCore(SyntaxKindIdentifier, NodeFlagsNone),
		text:     text,
	}
}

func (Factory) PrivateIdentifier(text string) PrivateIdentifier {
	return &privateIdentifierNode{
		nodeCore: newNodeCore(SyntaxKindPrivateIdentifier, NodeFlagsNone),
		text:     text,
	}
}

func (Factory) CaseKeyword() CaseKeyword {
	return &caseKeywordNode{
		nodeCore: newNodeCore(SyntaxKindCaseKeyword, NodeFlagsNone),
	}
}

func (Factory) ConstKeyword() ConstKeyword {
	return &constKeywordNode{
		nodeCore: newNodeCore(SyntaxKindConstKeyword, NodeFlagsNone),
	}
}

func (Factory) DefaultKeyword() DefaultKeyword {
	return &defaultKeywordNode{
		nodeCore: newNodeCore(SyntaxKindDefaultKeyword, NodeFlagsNone),
	}
}

func (Factory) ExportKeyword() ExportKeyword {
	return &exportKeywordNode{
		nodeCore: newNodeCore(SyntaxKindExportKeyword, NodeFlagsNone),
	}
}

func (Factory) FalseLiteral() FalseLiteral {
	return &falseLiteralNode{
		nodeCore: newNodeCore(SyntaxKindFalseKeyword, NodeFlagsNone),
	}
}

func (Factory) ImportExpression() ImportExpression {
	return &importExpressionNode{
		nodeCore: newNodeCore(SyntaxKindImportKeyword, NodeFlagsNone),
	}
}

func (Factory) InKeyword() InKeyword {
	return &inKeywordNode{
		nodeCore: newNodeCore(SyntaxKindInKeyword, NodeFlagsNone),
	}
}

func (Factory) NullLiteral() NullLiteral {
	return &nullLiteralNode{
		nodeCore: newNodeCore(SyntaxKindNullKeyword, NodeFlagsNone),
	}
}

func (Factory) SuperExpression() SuperExpression {
	return &superExpressionNode{
		nodeCore: newNodeCore(SyntaxKindSuperKeyword, NodeFlagsNone),
	}
}

func (Factory) ThisExpression() ThisExpression {
	return &thisExpressionNode{
		nodeCore: newNodeCore(SyntaxKindThisKeyword, NodeFlagsNone),
	}
}

func (Factory) TrueLiteral() TrueLiteral {
	return &trueLiteralNode{
		nodeCore: newNodeCore(SyntaxKindTrueKeyword, NodeFlagsNone),
	}
}

func (Factory) PrivateKeyword() PrivateKeyword {
	return &privateKeywordNode{
		nodeCore: newNodeCore(SyntaxKindPrivateKeyword, NodeFlagsNone),
	}
}

func (Factory) ProtectedKeyword() ProtectedKeyword {
	return &protectedKeywordNode{
		nodeCore: newNodeCore(SyntaxKindProtectedKeyword, NodeFlagsNone),
	}
}

func (Factory) PublicKeyword() PublicKeyword {
	return &publicKeywordNode{
		nodeCore: newNodeCore(SyntaxKindPublicKeyword, NodeFlagsNone),
	}
}

func (Factory) StaticKeyword() StaticKeyword {
	return &staticKeywordNode{
		nodeCore: newNodeCore(SyntaxKindStaticKeyword, NodeFlagsNone),
	}
}

func (Factory) AbstractKeyword() AbstractKeyword {
	return &abstractKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAbstractKeyword, NodeFlagsNone),
	}
}

func (Factory) AccessorKeyword() AccessorKeyword {
	return &accessorKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAccessorKeyword, NodeFlagsNone),
	}
}

func (Factory) AssertsKeyword() AssertsKeyword {
	return &assertsKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAssertsKeyword, NodeFlagsNone),
	}
}

func (Factory) AssertKeyword() AssertKeyword {
	return &assertKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAssertKeyword, NodeFlagsNone),
	}
}

func (Factory) AsyncKeyword() AsyncKeyword {
	return &asyncKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAsyncKeyword, NodeFlagsNone),
	}
}

func (Factory) AwaitKeyword() AwaitKeyword {
	return &awaitKeywordNode{
		nodeCore: newNodeCore(SyntaxKindAwaitKeyword, NodeFlagsNone),
	}
}

func (Factory) DeclareKeyword() DeclareKeyword {
	return &declareKeywordNode{
		nodeCore: newNodeCore(SyntaxKindDeclareKeyword, NodeFlagsNone),
	}
}

func (Factory) OutKeyword() OutKeyword {
	return &outKeywordNode{
		nodeCore: newNodeCore(SyntaxKindOutKeyword, NodeFlagsNone),
	}
}

func (Factory) ReadonlyKeyword() ReadonlyKeyword {
	return &readonlyKeywordNode{
		nodeCore: newNodeCore(SyntaxKindReadonlyKeyword, NodeFlagsNone),
	}
}

func (Factory) OverrideKeyword() OverrideKeyword {
	return &overrideKeywordNode{
		nodeCore: newNodeCore(SyntaxKindOverrideKeyword, NodeFlagsNone),
	}
}

func (Factory) QualifiedName(left EntityName, right Identifier) QualifiedName {
	return &qualifiedNameNode{
		nodeCore: newNodeCore(SyntaxKindQualifiedName, NodeFlagsNone),
		left:     left,
		right:    right,
	}
}

func (Factory) ComputedPropertyName(expression Expression) ComputedPropertyName {
	return &computedPropertyNameNode{
		nodeCore:   newNodeCore(SyntaxKindComputedPropertyName, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) TypeParameterDeclaration(modifiers []ModifierLike, name Identifier, constraint TypeNode, expression Expression, defaultType TypeNode) TypeParameterDeclaration {
	return &typeParameterDeclarationNode{
		nodeCore:    newNodeCore(SyntaxKindTypeParameter, NodeFlagsNone),
		modifiers:   cloneSlice(modifiers),
		name:        name,
		constraint:  constraint,
		expression:  expression,
		defaultType: defaultType,
	}
}

func (Factory) ParameterDeclaration(modifiers []ModifierLike, dotDotDotToken DotDotDotToken, name BindingName, questionToken QuestionToken, typeNode TypeNode, initializer Expression) ParameterDeclaration {
	return &parameterDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindParameter, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		dotDotDotToken: dotDotDotToken,
		name:           name,
		questionToken:  questionToken,
		typeNode:       typeNode,
		initializer:    initializer,
	}
}

func (Factory) Decorator(expression LeftHandSideExpression) Decorator {
	return &decoratorNode{
		nodeCore:   newNodeCore(SyntaxKindDecorator, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) PropertySignatureDeclaration(modifiers []ModifierLike, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeNode TypeNode, initializer Expression) PropertySignatureDeclaration {
	return &propertySignatureDeclarationNode{
		nodeCore:     newNodeCore(SyntaxKindPropertySignature, NodeFlagsNone),
		modifiers:    cloneSlice(modifiers),
		name:         name,
		postfixToken: postfixToken,
		typeNode:     typeNode,
		initializer:  initializer,
	}
}

func (Factory) PropertyDeclaration(modifiers []ModifierLike, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeNode TypeNode, initializer Expression) PropertyDeclaration {
	return &propertyDeclarationNode{
		nodeCore:     newNodeCore(SyntaxKindPropertyDeclaration, NodeFlagsNone),
		modifiers:    cloneSlice(modifiers),
		name:         name,
		postfixToken: postfixToken,
		typeNode:     typeNode,
		initializer:  initializer,
	}
}

func (Factory) MethodSignatureDeclaration(modifiers []ModifierLike, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) MethodSignatureDeclaration {
	return &methodSignatureDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindMethodSignature, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		name:           name,
		postfixToken:   postfixToken,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) MethodDeclaration(modifiers []ModifierLike, asteriskToken AsteriskToken, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) MethodDeclaration {
	return &methodDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindMethodDeclaration, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		asteriskToken:  asteriskToken,
		name:           name,
		postfixToken:   postfixToken,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) ClassStaticBlockDeclaration(modifiers []ModifierLike, body Block) ClassStaticBlockDeclaration {
	return &classStaticBlockDeclarationNode{
		nodeCore:  newNodeCore(SyntaxKindClassStaticBlockDeclaration, NodeFlagsNone),
		modifiers: cloneSlice(modifiers),
		body:      body,
	}
}

func (Factory) ConstructorDeclaration(modifiers []ModifierLike, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) ConstructorDeclaration {
	return &constructorDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindConstructor, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) GetAccessorDeclaration(modifiers []ModifierLike, name PropertyName, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) GetAccessorDeclaration {
	return &getAccessorDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindGetAccessor, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		name:           name,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) SetAccessorDeclaration(modifiers []ModifierLike, name PropertyName, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) SetAccessorDeclaration {
	return &setAccessorDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindSetAccessor, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		name:           name,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) CallSignatureDeclaration(typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) CallSignatureDeclaration {
	return &callSignatureDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindCallSignature, NodeFlagsNone),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) ConstructSignatureDeclaration(typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) ConstructSignatureDeclaration {
	return &constructSignatureDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindConstructSignature, NodeFlagsNone),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) IndexSignatureDeclaration(modifiers []ModifierLike, parameters []ParameterDeclaration, typeNode TypeNode) IndexSignatureDeclaration {
	return &indexSignatureDeclarationNode{
		nodeCore:   newNodeCore(SyntaxKindIndexSignature, NodeFlagsNone),
		modifiers:  cloneSlice(modifiers),
		parameters: cloneSlice(parameters),
		typeNode:   typeNode,
	}
}

func (Factory) TypePredicateNode(assertsModifier AssertsKeyword, parameterName TypePredicateParameterName, typeNode TypeNode) TypePredicateNode {
	return &typePredicateNodeNode{
		nodeCore:        newNodeCore(SyntaxKindTypePredicate, NodeFlagsNone),
		assertsModifier: assertsModifier,
		parameterName:   parameterName,
		typeNode:        typeNode,
	}
}

func (Factory) TypeReferenceNode(typeName EntityName, typeArguments []TypeNode) TypeReferenceNode {
	return &typeReferenceNodeNode{
		nodeCore:      newNodeCore(SyntaxKindTypeReference, NodeFlagsNone),
		typeName:      typeName,
		typeArguments: cloneSlice(typeArguments),
	}
}

func (Factory) FunctionTypeNode(typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) FunctionTypeNode {
	return &functionTypeNodeNode{
		nodeCore:       newNodeCore(SyntaxKindFunctionType, NodeFlagsNone),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) ConstructorTypeNode(modifiers []ModifierLike, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) ConstructorTypeNode {
	return &constructorTypeNodeNode{
		nodeCore:       newNodeCore(SyntaxKindConstructorType, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) TypeQueryNode(exprName EntityName, typeArguments []TypeNode) TypeQueryNode {
	return &typeQueryNodeNode{
		nodeCore:      newNodeCore(SyntaxKindTypeQuery, NodeFlagsNone),
		exprName:      exprName,
		typeArguments: cloneSlice(typeArguments),
	}
}

func (Factory) TypeLiteralNode(members []TypeElement) TypeLiteralNode {
	return &typeLiteralNodeNode{
		nodeCore: newNodeCore(SyntaxKindTypeLiteral, NodeFlagsNone),
		members:  cloneSlice(members),
	}
}

func (Factory) ArrayTypeNode(elementType TypeNode) ArrayTypeNode {
	return &arrayTypeNodeNode{
		nodeCore:    newNodeCore(SyntaxKindArrayType, NodeFlagsNone),
		elementType: elementType,
	}
}

func (Factory) TupleTypeNode(elements []TypeNode) TupleTypeNode {
	return &tupleTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindTupleType, NodeFlagsNone),
		elements: cloneSlice(elements),
	}
}

func (Factory) OptionalTypeNode(typeNode TypeNode) OptionalTypeNode {
	return &optionalTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindOptionalType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) RestTypeNode(typeNode TypeNode) RestTypeNode {
	return &restTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindRestType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) UnionTypeNode(types []TypeNode) UnionTypeNode {
	return &unionTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindUnionType, NodeFlagsNone),
		types:    cloneSlice(types),
	}
}

func (Factory) IntersectionTypeNode(types []TypeNode) IntersectionTypeNode {
	return &intersectionTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindIntersectionType, NodeFlagsNone),
		types:    cloneSlice(types),
	}
}

func (Factory) ConditionalTypeNode(checkType TypeNode, extendsType TypeNode, trueType TypeNode, falseType TypeNode) ConditionalTypeNode {
	return &conditionalTypeNodeNode{
		nodeCore:    newNodeCore(SyntaxKindConditionalType, NodeFlagsNone),
		checkType:   checkType,
		extendsType: extendsType,
		trueType:    trueType,
		falseType:   falseType,
	}
}

func (Factory) InferTypeNode(typeParameter TypeParameterDeclaration) InferTypeNode {
	return &inferTypeNodeNode{
		nodeCore:      newNodeCore(SyntaxKindInferType, NodeFlagsNone),
		typeParameter: typeParameter,
	}
}

func (Factory) ParenthesizedTypeNode(typeNode TypeNode) ParenthesizedTypeNode {
	return &parenthesizedTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindParenthesizedType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) ThisTypeNode() ThisTypeNode {
	return &thisTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindThisType, NodeFlagsNone),
	}
}

func (Factory) TypeOperatorNode(operator TypeOperatorNodeOperatorKind, typeNode TypeNode) TypeOperatorNode {
	return &typeOperatorNodeNode{
		nodeCore: newNodeCore(SyntaxKindTypeOperator, NodeFlagsNone),
		operator: operator,
		typeNode: typeNode,
	}
}

func (Factory) IndexedAccessTypeNode(objectType TypeNode, indexType TypeNode) IndexedAccessTypeNode {
	return &indexedAccessTypeNodeNode{
		nodeCore:   newNodeCore(SyntaxKindIndexedAccessType, NodeFlagsNone),
		objectType: objectType,
		indexType:  indexType,
	}
}

func (Factory) MappedTypeNode(readonlyToken MappedTypeNodeReadonlyToken, typeParameter TypeParameterDeclaration, nameType TypeNode, questionToken MappedTypeNodeQuestionToken, typeNode TypeNode, members []TypeElement) MappedTypeNode {
	return &mappedTypeNodeNode{
		nodeCore:      newNodeCore(SyntaxKindMappedType, NodeFlagsNone),
		readonlyToken: readonlyToken,
		typeParameter: typeParameter,
		nameType:      nameType,
		questionToken: questionToken,
		typeNode:      typeNode,
		members:       cloneSlice(members),
	}
}

func (Factory) LiteralTypeNode(literal Node) LiteralTypeNode {
	return &literalTypeNodeNode{
		nodeCore: newNodeCore(SyntaxKindLiteralType, NodeFlagsNone),
		literal:  literal,
	}
}

func (Factory) NamedTupleMember(dotDotDotToken DotDotDotToken, name Identifier, questionToken QuestionToken, typeNode TypeNode) NamedTupleMember {
	return &namedTupleMemberNode{
		nodeCore:       newNodeCore(SyntaxKindNamedTupleMember, NodeFlagsNone),
		dotDotDotToken: dotDotDotToken,
		name:           name,
		questionToken:  questionToken,
		typeNode:       typeNode,
	}
}

func (Factory) TemplateLiteralTypeNode(head TemplateHead, templateSpans []TemplateLiteralTypeSpan) TemplateLiteralTypeNode {
	return &templateLiteralTypeNodeNode{
		nodeCore:      newNodeCore(SyntaxKindTemplateLiteralType, NodeFlagsNone),
		head:          head,
		templateSpans: cloneSlice(templateSpans),
	}
}

func (Factory) TemplateLiteralTypeSpan(typeNode TypeNode, literal TemplateMiddleOrTail) TemplateLiteralTypeSpan {
	return &templateLiteralTypeSpanNode{
		nodeCore: newNodeCore(SyntaxKindTemplateLiteralTypeSpan, NodeFlagsNone),
		typeNode: typeNode,
		literal:  literal,
	}
}

func (Factory) ImportTypeNode(isTypeOf bool, argument TypeNode, attributes ImportAttributes, qualifier EntityName, typeArguments []TypeNode) ImportTypeNode {
	return &importTypeNodeNode{
		nodeCore:      newNodeCore(SyntaxKindImportType, NodeFlagsNone),
		isTypeOf:      isTypeOf,
		argument:      argument,
		attributes:    attributes,
		qualifier:     qualifier,
		typeArguments: cloneSlice(typeArguments),
	}
}

func (Factory) ObjectBindingPattern(elements []BindingElement) ObjectBindingPattern {
	return &objectBindingPatternNode{
		nodeCore: newNodeCore(SyntaxKindObjectBindingPattern, NodeFlagsNone),
		elements: cloneSlice(elements),
	}
}

func (Factory) ArrayBindingPattern(elements []BindingElement) ArrayBindingPattern {
	return &arrayBindingPatternNode{
		nodeCore: newNodeCore(SyntaxKindArrayBindingPattern, NodeFlagsNone),
		elements: cloneSlice(elements),
	}
}

func (Factory) BindingElement(dotDotDotToken DotDotDotToken, propertyName PropertyName, name BindingName, initializer Expression) BindingElement {
	return &bindingElementNode{
		nodeCore:       newNodeCore(SyntaxKindBindingElement, NodeFlagsNone),
		dotDotDotToken: dotDotDotToken,
		propertyName:   propertyName,
		name:           name,
		initializer:    initializer,
	}
}

func (Factory) ArrayLiteralExpression(elements []Expression, multiLine bool) ArrayLiteralExpression {
	return &arrayLiteralExpressionNode{
		nodeCore:  newNodeCore(SyntaxKindArrayLiteralExpression, NodeFlagsNone),
		elements:  cloneSlice(elements),
		multiLine: multiLine,
	}
}

func (Factory) ObjectLiteralExpression(properties []ObjectLiteralElementLike, multiLine bool) ObjectLiteralExpression {
	return &objectLiteralExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindObjectLiteralExpression, NodeFlagsNone),
		properties: cloneSlice(properties),
		multiLine:  multiLine,
	}
}

func (Factory) PropertyAccessExpression(expression Expression, questionDotToken QuestionDotToken, name MemberName, flags NodeFlags) PropertyAccessExpression {
	return &propertyAccessExpressionNode{
		nodeCore:         newNodeCore(SyntaxKindPropertyAccessExpression, flags),
		expression:       expression,
		questionDotToken: questionDotToken,
		name:             name,
	}
}

func (Factory) ElementAccessExpression(expression Expression, questionDotToken QuestionDotToken, argumentExpression Expression, flags NodeFlags) ElementAccessExpression {
	return &elementAccessExpressionNode{
		nodeCore:           newNodeCore(SyntaxKindElementAccessExpression, flags),
		expression:         expression,
		questionDotToken:   questionDotToken,
		argumentExpression: argumentExpression,
	}
}

func (Factory) CallExpression(expression Expression, questionDotToken QuestionDotToken, typeArguments []TypeNode, arguments []Expression, flags NodeFlags) CallExpression {
	return &callExpressionNode{
		nodeCore:         newNodeCore(SyntaxKindCallExpression, flags),
		expression:       expression,
		questionDotToken: questionDotToken,
		typeArguments:    cloneSlice(typeArguments),
		arguments:        cloneSlice(arguments),
	}
}

func (Factory) NewExpression(expression Expression, typeArguments []TypeNode, arguments []Expression) NewExpression {
	return &newExpressionNode{
		nodeCore:      newNodeCore(SyntaxKindNewExpression, NodeFlagsNone),
		expression:    expression,
		typeArguments: cloneSlice(typeArguments),
		arguments:     cloneSlice(arguments),
	}
}

func (Factory) TaggedTemplateExpression(tag Expression, questionDotToken QuestionDotToken, typeArguments []TypeNode, template TemplateLiteral, flags NodeFlags) TaggedTemplateExpression {
	return &taggedTemplateExpressionNode{
		nodeCore:         newNodeCore(SyntaxKindTaggedTemplateExpression, flags),
		tag:              tag,
		questionDotToken: questionDotToken,
		typeArguments:    cloneSlice(typeArguments),
		template:         template,
	}
}

func (Factory) TypeAssertion(typeNode TypeNode, expression Expression) TypeAssertion {
	return &typeAssertionNode{
		nodeCore:   newNodeCore(SyntaxKindTypeAssertionExpression, NodeFlagsNone),
		typeNode:   typeNode,
		expression: expression,
	}
}

func (Factory) ParenthesizedExpression(expression Expression) ParenthesizedExpression {
	return &parenthesizedExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindParenthesizedExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) FunctionExpression(modifiers []ModifierLike, asteriskToken AsteriskToken, name Identifier, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) FunctionExpression {
	return &functionExpressionNode{
		nodeCore:       newNodeCore(SyntaxKindFunctionExpression, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		asteriskToken:  asteriskToken,
		name:           name,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) ArrowFunction(modifiers []ModifierLike, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, equalsGreaterThanToken EqualsGreaterThanToken, body ConciseBody) ArrowFunction {
	return &arrowFunctionNode{
		nodeCore:               newNodeCore(SyntaxKindArrowFunction, NodeFlagsNone),
		modifiers:              cloneSlice(modifiers),
		typeParameters:         cloneSlice(typeParameters),
		parameters:             cloneSlice(parameters),
		typeNode:               typeNode,
		equalsGreaterThanToken: equalsGreaterThanToken,
		body:                   body,
	}
}

func (Factory) DeleteExpression(expression Expression) DeleteExpression {
	return &deleteExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindDeleteExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) TypeOfExpression(expression Expression) TypeOfExpression {
	return &typeOfExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindTypeOfExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) VoidExpression(expression Expression) VoidExpression {
	return &voidExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindVoidExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) AwaitExpression(expression Expression) AwaitExpression {
	return &awaitExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindAwaitExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) PrefixUnaryExpression(operator PrefixUnaryExpressionOperatorKind, operand Expression) PrefixUnaryExpression {
	return &prefixUnaryExpressionNode{
		nodeCore: newNodeCore(SyntaxKindPrefixUnaryExpression, NodeFlagsNone),
		operator: operator,
		operand:  operand,
	}
}

func (Factory) PostfixUnaryExpression(operand Expression, operator PostfixUnaryExpressionOperatorKind) PostfixUnaryExpression {
	return &postfixUnaryExpressionNode{
		nodeCore: newNodeCore(SyntaxKindPostfixUnaryExpression, NodeFlagsNone),
		operand:  operand,
		operator: operator,
	}
}

func (Factory) BinaryExpression(modifiers []ModifierLike, left Expression, typeNode TypeNode, operatorToken BinaryOperatorToken, right Expression) BinaryExpression {
	return &binaryExpressionNode{
		nodeCore:      newNodeCore(SyntaxKindBinaryExpression, NodeFlagsNone),
		modifiers:     cloneSlice(modifiers),
		left:          left,
		typeNode:      typeNode,
		operatorToken: operatorToken,
		right:         right,
	}
}

func (Factory) ConditionalExpression(condition Expression, questionToken QuestionToken, whenTrue Expression, colonToken ColonToken, whenFalse Expression) ConditionalExpression {
	return &conditionalExpressionNode{
		nodeCore:      newNodeCore(SyntaxKindConditionalExpression, NodeFlagsNone),
		condition:     condition,
		questionToken: questionToken,
		whenTrue:      whenTrue,
		colonToken:    colonToken,
		whenFalse:     whenFalse,
	}
}

func (Factory) TemplateExpression(head TemplateHead, templateSpans []TemplateSpan) TemplateExpression {
	return &templateExpressionNode{
		nodeCore:      newNodeCore(SyntaxKindTemplateExpression, NodeFlagsNone),
		head:          head,
		templateSpans: cloneSlice(templateSpans),
	}
}

func (Factory) YieldExpression(asteriskToken AsteriskToken, expression Expression) YieldExpression {
	return &yieldExpressionNode{
		nodeCore:      newNodeCore(SyntaxKindYieldExpression, NodeFlagsNone),
		asteriskToken: asteriskToken,
		expression:    expression,
	}
}

func (Factory) SpreadElement(expression Expression) SpreadElement {
	return &spreadElementNode{
		nodeCore:   newNodeCore(SyntaxKindSpreadElement, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) ClassExpression(modifiers []ModifierLike, name Identifier, typeParameters []TypeParameterDeclaration, heritageClauses []HeritageClause, members []ClassElement) ClassExpression {
	return &classExpressionNode{
		nodeCore:        newNodeCore(SyntaxKindClassExpression, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		name:            name,
		typeParameters:  cloneSlice(typeParameters),
		heritageClauses: cloneSlice(heritageClauses),
		members:         cloneSlice(members),
	}
}

func (Factory) OmittedExpression() OmittedExpression {
	return &omittedExpressionNode{
		nodeCore: newNodeCore(SyntaxKindOmittedExpression, NodeFlagsNone),
	}
}

func (Factory) ExpressionWithTypeArguments(expression Expression, typeArguments []TypeNode) ExpressionWithTypeArguments {
	return &expressionWithTypeArgumentsNode{
		nodeCore:      newNodeCore(SyntaxKindExpressionWithTypeArguments, NodeFlagsNone),
		expression:    expression,
		typeArguments: cloneSlice(typeArguments),
	}
}

func (Factory) AsExpression(expression Expression, typeNode TypeNode) AsExpression {
	return &asExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindAsExpression, NodeFlagsNone),
		expression: expression,
		typeNode:   typeNode,
	}
}

func (Factory) NonNullExpression(expression Expression, flags NodeFlags) NonNullExpression {
	return &nonNullExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindNonNullExpression, flags),
		expression: expression,
	}
}

func (Factory) MetaProperty(keywordToken MetaPropertyKeywordTokenKind, name Identifier) MetaProperty {
	return &metaPropertyNode{
		nodeCore:     newNodeCore(SyntaxKindMetaProperty, NodeFlagsNone),
		keywordToken: keywordToken,
		name:         name,
	}
}

func (Factory) SatisfiesExpression(expression Expression, typeNode TypeNode) SatisfiesExpression {
	return &satisfiesExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindSatisfiesExpression, NodeFlagsNone),
		expression: expression,
		typeNode:   typeNode,
	}
}

func (Factory) TemplateSpan(expression Expression, literal TemplateMiddleOrTail) TemplateSpan {
	return &templateSpanNode{
		nodeCore:   newNodeCore(SyntaxKindTemplateSpan, NodeFlagsNone),
		expression: expression,
		literal:    literal,
	}
}

func (Factory) SemicolonClassElement() SemicolonClassElement {
	return &semicolonClassElementNode{
		nodeCore: newNodeCore(SyntaxKindSemicolonClassElement, NodeFlagsNone),
	}
}

func (Factory) Block(statements []Statement, multiLine bool) Block {
	return &blockNode{
		nodeCore:   newNodeCore(SyntaxKindBlock, NodeFlagsNone),
		statements: cloneSlice(statements),
		multiLine:  multiLine,
	}
}

func (Factory) EmptyStatement() EmptyStatement {
	return &emptyStatementNode{
		nodeCore: newNodeCore(SyntaxKindEmptyStatement, NodeFlagsNone),
	}
}

func (Factory) VariableStatement(modifiers []ModifierLike, declarationList VariableDeclarationList) VariableStatement {
	return &variableStatementNode{
		nodeCore:        newNodeCore(SyntaxKindVariableStatement, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		declarationList: declarationList,
	}
}

func (Factory) ExpressionStatement(expression Expression) ExpressionStatement {
	return &expressionStatementNode{
		nodeCore:   newNodeCore(SyntaxKindExpressionStatement, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) IfStatement(expression Expression, thenStatement Statement, elseStatement Statement) IfStatement {
	return &ifStatementNode{
		nodeCore:      newNodeCore(SyntaxKindIfStatement, NodeFlagsNone),
		expression:    expression,
		thenStatement: thenStatement,
		elseStatement: elseStatement,
	}
}

func (Factory) DoStatement(statement Statement, expression Expression) DoStatement {
	return &doStatementNode{
		nodeCore:   newNodeCore(SyntaxKindDoStatement, NodeFlagsNone),
		statement:  statement,
		expression: expression,
	}
}

func (Factory) WhileStatement(expression Expression, statement Statement) WhileStatement {
	return &whileStatementNode{
		nodeCore:   newNodeCore(SyntaxKindWhileStatement, NodeFlagsNone),
		expression: expression,
		statement:  statement,
	}
}

func (Factory) ForStatement(initializer ForInitializer, condition Expression, incrementor Expression, statement Statement) ForStatement {
	return &forStatementNode{
		nodeCore:    newNodeCore(SyntaxKindForStatement, NodeFlagsNone),
		initializer: initializer,
		condition:   condition,
		incrementor: incrementor,
		statement:   statement,
	}
}

func (Factory) ForInStatement(awaitModifier AwaitKeyword, initializer ForInitializer, expression Expression, statement Statement) ForInStatement {
	return &forInStatementNode{
		nodeCore:      newNodeCore(SyntaxKindForInStatement, NodeFlagsNone),
		awaitModifier: awaitModifier,
		initializer:   initializer,
		expression:    expression,
		statement:     statement,
	}
}

func (Factory) ForOfStatement(awaitModifier AwaitKeyword, initializer ForInitializer, expression Expression, statement Statement) ForOfStatement {
	return &forOfStatementNode{
		nodeCore:      newNodeCore(SyntaxKindForOfStatement, NodeFlagsNone),
		awaitModifier: awaitModifier,
		initializer:   initializer,
		expression:    expression,
		statement:     statement,
	}
}

func (Factory) ContinueStatement(label Identifier) ContinueStatement {
	return &continueStatementNode{
		nodeCore: newNodeCore(SyntaxKindContinueStatement, NodeFlagsNone),
		label:    label,
	}
}

func (Factory) BreakStatement(label Identifier) BreakStatement {
	return &breakStatementNode{
		nodeCore: newNodeCore(SyntaxKindBreakStatement, NodeFlagsNone),
		label:    label,
	}
}

func (Factory) ReturnStatement(expression Expression) ReturnStatement {
	return &returnStatementNode{
		nodeCore:   newNodeCore(SyntaxKindReturnStatement, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) WithStatement(expression Expression, statement Statement) WithStatement {
	return &withStatementNode{
		nodeCore:   newNodeCore(SyntaxKindWithStatement, NodeFlagsNone),
		expression: expression,
		statement:  statement,
	}
}

func (Factory) SwitchStatement(expression Expression, caseBlock CaseBlock) SwitchStatement {
	return &switchStatementNode{
		nodeCore:   newNodeCore(SyntaxKindSwitchStatement, NodeFlagsNone),
		expression: expression,
		caseBlock:  caseBlock,
	}
}

func (Factory) LabeledStatement(label Identifier, statement Statement) LabeledStatement {
	return &labeledStatementNode{
		nodeCore:  newNodeCore(SyntaxKindLabeledStatement, NodeFlagsNone),
		label:     label,
		statement: statement,
	}
}

func (Factory) ThrowStatement(expression Expression) ThrowStatement {
	return &throwStatementNode{
		nodeCore:   newNodeCore(SyntaxKindThrowStatement, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) TryStatement(tryBlock Block, catchClause CatchClause, finallyBlock Block) TryStatement {
	return &tryStatementNode{
		nodeCore:     newNodeCore(SyntaxKindTryStatement, NodeFlagsNone),
		tryBlock:     tryBlock,
		catchClause:  catchClause,
		finallyBlock: finallyBlock,
	}
}

func (Factory) DebuggerStatement() DebuggerStatement {
	return &debuggerStatementNode{
		nodeCore: newNodeCore(SyntaxKindDebuggerStatement, NodeFlagsNone),
	}
}

func (Factory) VariableDeclaration(name BindingName, exclamationToken ExclamationToken, typeNode TypeNode, initializer Expression) VariableDeclaration {
	return &variableDeclarationNode{
		nodeCore:         newNodeCore(SyntaxKindVariableDeclaration, NodeFlagsNone),
		name:             name,
		exclamationToken: exclamationToken,
		typeNode:         typeNode,
		initializer:      initializer,
	}
}

func (Factory) VariableDeclarationList(declarations []VariableDeclaration, flags NodeFlags) VariableDeclarationList {
	return &variableDeclarationListNode{
		nodeCore:     newNodeCore(SyntaxKindVariableDeclarationList, flags),
		declarations: cloneSlice(declarations),
	}
}

func (Factory) FunctionDeclaration(modifiers []ModifierLike, asteriskToken AsteriskToken, name Identifier, typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode, body FunctionBody) FunctionDeclaration {
	return &functionDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindFunctionDeclaration, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		asteriskToken:  asteriskToken,
		name:           name,
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
		body:           body,
	}
}

func (Factory) ClassDeclaration(modifiers []ModifierLike, name Identifier, typeParameters []TypeParameterDeclaration, heritageClauses []HeritageClause, members []ClassElement) ClassDeclaration {
	return &classDeclarationNode{
		nodeCore:        newNodeCore(SyntaxKindClassDeclaration, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		name:            name,
		typeParameters:  cloneSlice(typeParameters),
		heritageClauses: cloneSlice(heritageClauses),
		members:         cloneSlice(members),
	}
}

func (Factory) InterfaceDeclaration(modifiers []ModifierLike, name Identifier, typeParameters []TypeParameterDeclaration, heritageClauses []HeritageClause, members []TypeElement) InterfaceDeclaration {
	return &interfaceDeclarationNode{
		nodeCore:        newNodeCore(SyntaxKindInterfaceDeclaration, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		name:            name,
		typeParameters:  cloneSlice(typeParameters),
		heritageClauses: cloneSlice(heritageClauses),
		members:         cloneSlice(members),
	}
}

func (Factory) TypeAliasDeclaration(modifiers []ModifierLike, name Identifier, typeParameters []TypeParameterDeclaration, typeNode TypeNode) TypeAliasDeclaration {
	return &typeAliasDeclarationNode{
		nodeCore:       newNodeCore(SyntaxKindTypeAliasDeclaration, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		name:           name,
		typeParameters: cloneSlice(typeParameters),
		typeNode:       typeNode,
	}
}

func (Factory) EnumDeclaration(modifiers []ModifierLike, name Identifier, members []EnumMember) EnumDeclaration {
	return &enumDeclarationNode{
		nodeCore:  newNodeCore(SyntaxKindEnumDeclaration, NodeFlagsNone),
		modifiers: cloneSlice(modifiers),
		name:      name,
		members:   cloneSlice(members),
	}
}

func (Factory) ModuleDeclaration(modifiers []ModifierLike, keyword ModuleDeclarationKeywordKind, name ModuleName, body ModuleBody) ModuleDeclaration {
	return &moduleDeclarationNode{
		nodeCore:  newNodeCore(SyntaxKindModuleDeclaration, NodeFlagsNone),
		modifiers: cloneSlice(modifiers),
		keyword:   keyword,
		name:      name,
		body:      body,
	}
}

func (Factory) ModuleBlock(statements []Statement) ModuleBlock {
	return &moduleBlockNode{
		nodeCore:   newNodeCore(SyntaxKindModuleBlock, NodeFlagsNone),
		statements: cloneSlice(statements),
	}
}

func (Factory) CaseBlock(clauses []CaseOrDefaultClause) CaseBlock {
	return &caseBlockNode{
		nodeCore: newNodeCore(SyntaxKindCaseBlock, NodeFlagsNone),
		clauses:  cloneSlice(clauses),
	}
}

func (Factory) NamespaceExportDeclaration(modifiers []ModifierLike, name Identifier) NamespaceExportDeclaration {
	return &namespaceExportDeclarationNode{
		nodeCore:  newNodeCore(SyntaxKindNamespaceExportDeclaration, NodeFlagsNone),
		modifiers: cloneSlice(modifiers),
		name:      name,
	}
}

func (Factory) ImportEqualsDeclaration(modifiers []ModifierLike, isTypeOnly bool, name Identifier, moduleReference ModuleReference) ImportEqualsDeclaration {
	return &importEqualsDeclarationNode{
		nodeCore:        newNodeCore(SyntaxKindImportEqualsDeclaration, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		isTypeOnly:      isTypeOnly,
		name:            name,
		moduleReference: moduleReference,
	}
}

func (Factory) ImportDeclaration(modifiers []ModifierLike, importClause ImportClause, moduleSpecifier Expression, attributes ImportAttributes) ImportDeclaration {
	return &importDeclarationNode{
		nodeCore:        newNodeCore(SyntaxKindImportDeclaration, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		importClause:    importClause,
		moduleSpecifier: moduleSpecifier,
		attributes:      attributes,
	}
}

func (Factory) ImportClause(phaseModifier ImportPhaseModifierSyntaxKind, name Identifier, namedBindings NamedImportBindings) ImportClause {
	return &importClauseNode{
		nodeCore:      newNodeCore(SyntaxKindImportClause, NodeFlagsNone),
		phaseModifier: phaseModifier,
		name:          name,
		namedBindings: namedBindings,
	}
}

func (Factory) NamespaceImport(name Identifier) NamespaceImport {
	return &namespaceImportNode{
		nodeCore: newNodeCore(SyntaxKindNamespaceImport, NodeFlagsNone),
		name:     name,
	}
}

func (Factory) NamedImports(elements []ImportSpecifier) NamedImports {
	return &namedImportsNode{
		nodeCore: newNodeCore(SyntaxKindNamedImports, NodeFlagsNone),
		elements: cloneSlice(elements),
	}
}

func (Factory) ImportSpecifier(isTypeOnly bool, propertyName ModuleExportName, name Identifier) ImportSpecifier {
	return &importSpecifierNode{
		nodeCore:     newNodeCore(SyntaxKindImportSpecifier, NodeFlagsNone),
		isTypeOnly:   isTypeOnly,
		propertyName: propertyName,
		name:         name,
	}
}

func (Factory) ExportAssignment(modifiers []ModifierLike, isExportEquals bool, typeNode TypeNode, expression Expression) ExportAssignment {
	return &exportAssignmentNode{
		nodeCore:       newNodeCore(SyntaxKindExportAssignment, NodeFlagsNone),
		modifiers:      cloneSlice(modifiers),
		isExportEquals: isExportEquals,
		typeNode:       typeNode,
		expression:     expression,
	}
}

func (Factory) ExportDeclaration(modifiers []ModifierLike, isTypeOnly bool, exportClause NamedExportBindings, moduleSpecifier Expression, attributes ImportAttributes) ExportDeclaration {
	return &exportDeclarationNode{
		nodeCore:        newNodeCore(SyntaxKindExportDeclaration, NodeFlagsNone),
		modifiers:       cloneSlice(modifiers),
		isTypeOnly:      isTypeOnly,
		exportClause:    exportClause,
		moduleSpecifier: moduleSpecifier,
		attributes:      attributes,
	}
}

func (Factory) NamedExports(elements []ExportSpecifier) NamedExports {
	return &namedExportsNode{
		nodeCore: newNodeCore(SyntaxKindNamedExports, NodeFlagsNone),
		elements: cloneSlice(elements),
	}
}

func (Factory) NamespaceExport(name ModuleExportName) NamespaceExport {
	return &namespaceExportNode{
		nodeCore: newNodeCore(SyntaxKindNamespaceExport, NodeFlagsNone),
		name:     name,
	}
}

func (Factory) ExportSpecifier(isTypeOnly bool, propertyName ModuleExportName, name ModuleExportName) ExportSpecifier {
	return &exportSpecifierNode{
		nodeCore:     newNodeCore(SyntaxKindExportSpecifier, NodeFlagsNone),
		isTypeOnly:   isTypeOnly,
		propertyName: propertyName,
		name:         name,
	}
}

func (Factory) MissingDeclaration(modifiers []ModifierLike) MissingDeclaration {
	return &missingDeclarationNode{
		nodeCore:  newNodeCore(SyntaxKindMissingDeclaration, NodeFlagsNone),
		modifiers: cloneSlice(modifiers),
	}
}

func (Factory) ExternalModuleReference(expression Expression) ExternalModuleReference {
	return &externalModuleReferenceNode{
		nodeCore:   newNodeCore(SyntaxKindExternalModuleReference, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) JsxElement(openingElement JsxOpeningElement, children []JsxChild, closingElement JsxClosingElement) JsxElement {
	return &jsxElementNode{
		nodeCore:       newNodeCore(SyntaxKindJsxElement, NodeFlagsNone),
		openingElement: openingElement,
		children:       cloneSlice(children),
		closingElement: closingElement,
	}
}

func (Factory) JsxSelfClosingElement(tagName JsxTagNameExpression, typeArguments []TypeNode, attributes JsxAttributes) JsxSelfClosingElement {
	return &jsxSelfClosingElementNode{
		nodeCore:      newNodeCore(SyntaxKindJsxSelfClosingElement, NodeFlagsNone),
		tagName:       tagName,
		typeArguments: cloneSlice(typeArguments),
		attributes:    attributes,
	}
}

func (Factory) JsxOpeningElement(tagName JsxTagNameExpression, typeArguments []TypeNode, attributes JsxAttributes) JsxOpeningElement {
	return &jsxOpeningElementNode{
		nodeCore:      newNodeCore(SyntaxKindJsxOpeningElement, NodeFlagsNone),
		tagName:       tagName,
		typeArguments: cloneSlice(typeArguments),
		attributes:    attributes,
	}
}

func (Factory) JsxClosingElement(tagName JsxTagNameExpression) JsxClosingElement {
	return &jsxClosingElementNode{
		nodeCore: newNodeCore(SyntaxKindJsxClosingElement, NodeFlagsNone),
		tagName:  tagName,
	}
}

func (Factory) JsxFragment(openingFragment JsxOpeningFragment, children []JsxChild, closingFragment JsxClosingFragment) JsxFragment {
	return &jsxFragmentNode{
		nodeCore:        newNodeCore(SyntaxKindJsxFragment, NodeFlagsNone),
		openingFragment: openingFragment,
		children:        cloneSlice(children),
		closingFragment: closingFragment,
	}
}

func (Factory) JsxOpeningFragment() JsxOpeningFragment {
	return &jsxOpeningFragmentNode{
		nodeCore: newNodeCore(SyntaxKindJsxOpeningFragment, NodeFlagsNone),
	}
}

func (Factory) JsxClosingFragment() JsxClosingFragment {
	return &jsxClosingFragmentNode{
		nodeCore: newNodeCore(SyntaxKindJsxClosingFragment, NodeFlagsNone),
	}
}

func (Factory) JsxAttribute(name JsxAttributeName, initializer JsxAttributeValue) JsxAttribute {
	return &jsxAttributeNode{
		nodeCore:    newNodeCore(SyntaxKindJsxAttribute, NodeFlagsNone),
		name:        name,
		initializer: initializer,
	}
}

func (Factory) JsxAttributes(properties []JsxAttributeLike) JsxAttributes {
	return &jsxAttributesNode{
		nodeCore:   newNodeCore(SyntaxKindJsxAttributes, NodeFlagsNone),
		properties: cloneSlice(properties),
	}
}

func (Factory) JsxSpreadAttribute(expression Expression) JsxSpreadAttribute {
	return &jsxSpreadAttributeNode{
		nodeCore:   newNodeCore(SyntaxKindJsxSpreadAttribute, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) JsxExpression(dotDotDotToken DotDotDotToken, expression Expression) JsxExpression {
	return &jsxExpressionNode{
		nodeCore:       newNodeCore(SyntaxKindJsxExpression, NodeFlagsNone),
		dotDotDotToken: dotDotDotToken,
		expression:     expression,
	}
}

func (Factory) JsxNamespacedName(namespace Identifier, name Identifier) JsxNamespacedName {
	return &jsxNamespacedNameNode{
		nodeCore:  newNodeCore(SyntaxKindJsxNamespacedName, NodeFlagsNone),
		namespace: namespace,
		name:      name,
	}
}

func (Factory) CaseClause(expression Expression, statements []Statement) CaseClause {
	return &caseClauseNode{
		nodeCore:   newNodeCore(SyntaxKindCaseClause, NodeFlagsNone),
		expression: expression,
		statements: cloneSlice(statements),
	}
}

func (Factory) DefaultClause(expression Expression, statements []Statement) DefaultClause {
	return &defaultClauseNode{
		nodeCore:   newNodeCore(SyntaxKindDefaultClause, NodeFlagsNone),
		expression: expression,
		statements: cloneSlice(statements),
	}
}

func (Factory) HeritageClause(token HeritageClauseTokenKind, types []ExpressionWithTypeArguments) HeritageClause {
	return &heritageClauseNode{
		nodeCore: newNodeCore(SyntaxKindHeritageClause, NodeFlagsNone),
		token:    token,
		types:    cloneSlice(types),
	}
}

func (Factory) CatchClause(variableDeclaration VariableDeclaration, block Block) CatchClause {
	return &catchClauseNode{
		nodeCore:            newNodeCore(SyntaxKindCatchClause, NodeFlagsNone),
		variableDeclaration: variableDeclaration,
		block:               block,
	}
}

func (Factory) ImportAttributes(token ImportAttributesTokenKind, attributes []ImportAttribute, multiLine bool) ImportAttributes {
	return &importAttributesNode{
		nodeCore:   newNodeCore(SyntaxKindImportAttributes, NodeFlagsNone),
		token:      token,
		attributes: cloneSlice(attributes),
		multiLine:  multiLine,
	}
}

func (Factory) ImportAttribute(name ImportAttributeName, value Expression) ImportAttribute {
	return &importAttributeNode{
		nodeCore: newNodeCore(SyntaxKindImportAttribute, NodeFlagsNone),
		name:     name,
		value:    value,
	}
}

func (Factory) PropertyAssignment(modifiers []ModifierLike, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeNode TypeNode, initializer Expression) PropertyAssignment {
	return &propertyAssignmentNode{
		nodeCore:     newNodeCore(SyntaxKindPropertyAssignment, NodeFlagsNone),
		modifiers:    cloneSlice(modifiers),
		name:         name,
		postfixToken: postfixToken,
		typeNode:     typeNode,
		initializer:  initializer,
	}
}

func (Factory) ShorthandPropertyAssignment(modifiers []ModifierLike, name PropertyName, postfixToken NamedMemberBasePostfixToken, typeNode TypeNode, equalsToken EqualsToken, objectAssignmentInitializer Expression) ShorthandPropertyAssignment {
	return &shorthandPropertyAssignmentNode{
		nodeCore:                    newNodeCore(SyntaxKindShorthandPropertyAssignment, NodeFlagsNone),
		modifiers:                   cloneSlice(modifiers),
		name:                        name,
		postfixToken:                postfixToken,
		typeNode:                    typeNode,
		equalsToken:                 equalsToken,
		objectAssignmentInitializer: objectAssignmentInitializer,
	}
}

func (Factory) SpreadAssignment(expression Expression) SpreadAssignment {
	return &spreadAssignmentNode{
		nodeCore:   newNodeCore(SyntaxKindSpreadAssignment, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) EnumMember(name PropertyName, initializer Expression) EnumMember {
	return &enumMemberNode{
		nodeCore:    newNodeCore(SyntaxKindEnumMember, NodeFlagsNone),
		name:        name,
		initializer: initializer,
	}
}

func (Factory) SourceFile(statements []Statement, endOfFileToken EndOfFile, sourceData SourceFileData) SourceFile {
	return &sourceFileNode{
		nodeCore:       newNodeCore(SyntaxKindSourceFile, NodeFlagsNone),
		statements:     cloneSlice(statements),
		endOfFileToken: endOfFileToken,
		sourceData:     cloneSourceFileData(sourceData),
	}
}

func (Factory) JSDocTypeExpression(typeNode TypeNode) JSDocTypeExpression {
	return &jsDocTypeExpressionNode{
		nodeCore: newNodeCore(SyntaxKindJSDocTypeExpression, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) JSDocNameReference(name EntityName) JSDocNameReference {
	return &jsDocNameReferenceNode{
		nodeCore: newNodeCore(SyntaxKindJSDocNameReference, NodeFlagsNone),
		name:     name,
	}
}

func (Factory) JSDocAllType() JSDocAllType {
	return &jsDocAllTypeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocAllType, NodeFlagsNone),
	}
}

func (Factory) JSDocNullableType(typeNode TypeNode) JSDocNullableType {
	return &jsDocNullableTypeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocNullableType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) JSDocNonNullableType(typeNode TypeNode) JSDocNonNullableType {
	return &jsDocNonNullableTypeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocNonNullableType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) JSDocOptionalType(typeNode TypeNode) JSDocOptionalType {
	return &jsDocOptionalTypeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocOptionalType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) JSDocVariadicType(typeNode TypeNode) JSDocVariadicType {
	return &jsDocVariadicTypeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocVariadicType, NodeFlagsNone),
		typeNode: typeNode,
	}
}

func (Factory) JSDoc(comment []JSDocComment, tags []JSDocTag) JSDoc {
	return &jsDocNode{
		nodeCore: newNodeCore(SyntaxKindJSDoc, NodeFlagsNone),
		comment:  cloneSlice(comment),
		tags:     cloneSlice(tags),
	}
}

func (Factory) JSDocText(text []string) JSDocText {
	return &jsDocTextNode{
		nodeCore: newNodeCore(SyntaxKindJSDocText, NodeFlagsNone),
		text:     cloneSlice(text),
	}
}

func (Factory) JSDocTypeLiteral(jSDocPropertyTags []JSDocTag, isArrayType bool) JSDocTypeLiteral {
	return &jsDocTypeLiteralNode{
		nodeCore:          newNodeCore(SyntaxKindJSDocTypeLiteral, NodeFlagsNone),
		jSDocPropertyTags: cloneSlice(jSDocPropertyTags),
		isArrayType:       isArrayType,
	}
}

func (Factory) JSDocSignature(typeParameters []TypeParameterDeclaration, parameters []ParameterDeclaration, typeNode TypeNode) JSDocSignature {
	return &jsDocSignatureNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocSignature, NodeFlagsNone),
		typeParameters: cloneSlice(typeParameters),
		parameters:     cloneSlice(parameters),
		typeNode:       typeNode,
	}
}

func (Factory) JSDocLink(name EntityName, text []string) JSDocLink {
	return &jsDocLinkNode{
		nodeCore: newNodeCore(SyntaxKindJSDocLink, NodeFlagsNone),
		name:     name,
		text:     cloneSlice(text),
	}
}

func (Factory) JSDocLinkCode(name EntityName, text []string) JSDocLinkCode {
	return &jsDocLinkCodeNode{
		nodeCore: newNodeCore(SyntaxKindJSDocLinkCode, NodeFlagsNone),
		name:     name,
		text:     cloneSlice(text),
	}
}

func (Factory) JSDocLinkPlain(name EntityName, text []string) JSDocLinkPlain {
	return &jsDocLinkPlainNode{
		nodeCore: newNodeCore(SyntaxKindJSDocLinkPlain, NodeFlagsNone),
		name:     name,
		text:     cloneSlice(text),
	}
}

func (Factory) JSDocUnknownTag(tagName Identifier, comment []JSDocComment) JSDocUnknownTag {
	return &jsDocUnknownTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocUnknownTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocAugmentsTag(tagName Identifier, className ExpressionWithTypeArguments, comment []JSDocComment) JSDocAugmentsTag {
	return &jsDocAugmentsTagNode{
		nodeCore:  newNodeCore(SyntaxKindJSDocAugmentsTag, NodeFlagsNone),
		tagName:   tagName,
		className: className,
		comment:   cloneSlice(comment),
	}
}

func (Factory) JSDocImplementsTag(tagName Identifier, className ExpressionWithTypeArguments, comment []JSDocComment) JSDocImplementsTag {
	return &jsDocImplementsTagNode{
		nodeCore:  newNodeCore(SyntaxKindJSDocImplementsTag, NodeFlagsNone),
		tagName:   tagName,
		className: className,
		comment:   cloneSlice(comment),
	}
}

func (Factory) JSDocDeprecatedTag(tagName Identifier, comment []JSDocComment) JSDocDeprecatedTag {
	return &jsDocDeprecatedTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocDeprecatedTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocPublicTag(tagName Identifier, comment []JSDocComment) JSDocPublicTag {
	return &jsDocPublicTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocPublicTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocPrivateTag(tagName Identifier, comment []JSDocComment) JSDocPrivateTag {
	return &jsDocPrivateTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocPrivateTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocProtectedTag(tagName Identifier, comment []JSDocComment) JSDocProtectedTag {
	return &jsDocProtectedTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocProtectedTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocReadonlyTag(tagName Identifier, comment []JSDocComment) JSDocReadonlyTag {
	return &jsDocReadonlyTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocReadonlyTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocOverrideTag(tagName Identifier, comment []JSDocComment) JSDocOverrideTag {
	return &jsDocOverrideTagNode{
		nodeCore: newNodeCore(SyntaxKindJSDocOverrideTag, NodeFlagsNone),
		tagName:  tagName,
		comment:  cloneSlice(comment),
	}
}

func (Factory) JSDocCallbackTag(tagName Identifier, typeExpression TypeNode, name JSDocFullName, comment []JSDocComment) JSDocCallbackTag {
	return &jsDocCallbackTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocCallbackTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		name:           name,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocOverloadTag(tagName Identifier, typeExpression TypeNode, comment []JSDocComment) JSDocOverloadTag {
	return &jsDocOverloadTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocOverloadTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocParameterTag(tagName Identifier, name EntityName, isBracketed bool, typeExpression TypeNode, isNameFirst bool, comment []JSDocComment) JSDocParameterTag {
	return &jsDocParameterTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocParameterTag, NodeFlagsNone),
		tagName:        tagName,
		name:           name,
		isBracketed:    isBracketed,
		typeExpression: typeExpression,
		isNameFirst:    isNameFirst,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocReturnTag(tagName Identifier, typeExpression TypeNode, comment []JSDocComment) JSDocReturnTag {
	return &jsDocReturnTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocReturnTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocThisTag(tagName Identifier, typeExpression TypeNode, comment []JSDocComment) JSDocThisTag {
	return &jsDocThisTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocThisTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocTypeTag(tagName Identifier, typeExpression Node, comment []JSDocComment) JSDocTypeTag {
	return &jsDocTypeTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocTypeTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocTemplateTag(tagName Identifier, constraint Node, typeParameters []TypeParameterDeclaration, comment []JSDocComment) JSDocTemplateTag {
	return &jsDocTemplateTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocTemplateTag, NodeFlagsNone),
		tagName:        tagName,
		constraint:     constraint,
		typeParameters: cloneSlice(typeParameters),
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocTypedefTag(tagName Identifier, typeExpression Node, name JSDocFullName, comment []JSDocComment) JSDocTypedefTag {
	return &jsDocTypedefTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocTypedefTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		name:           name,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocSeeTag(tagName Identifier, nameExpression TypeNode, comment []JSDocComment) JSDocSeeTag {
	return &jsDocSeeTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocSeeTag, NodeFlagsNone),
		tagName:        tagName,
		nameExpression: nameExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocPropertyTag(tagName Identifier, name EntityName, isBracketed bool, typeExpression TypeNode, isNameFirst bool, comment []JSDocComment) JSDocPropertyTag {
	return &jsDocPropertyTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocPropertyTag, NodeFlagsNone),
		tagName:        tagName,
		name:           name,
		isBracketed:    isBracketed,
		typeExpression: typeExpression,
		isNameFirst:    isNameFirst,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocThrowsTag(tagName Identifier, typeExpression TypeNode, comment []JSDocComment) JSDocThrowsTag {
	return &jsDocThrowsTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocThrowsTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocSatisfiesTag(tagName Identifier, typeExpression TypeNode, comment []JSDocComment) JSDocSatisfiesTag {
	return &jsDocSatisfiesTagNode{
		nodeCore:       newNodeCore(SyntaxKindJSDocSatisfiesTag, NodeFlagsNone),
		tagName:        tagName,
		typeExpression: typeExpression,
		comment:        cloneSlice(comment),
	}
}

func (Factory) JSDocImportTag(tagName Identifier, importClause ImportClause, moduleSpecifier Expression, attributes ImportAttributes, comment []JSDocComment) JSDocImportTag {
	return &jsDocImportTagNode{
		nodeCore:        newNodeCore(SyntaxKindJSDocImportTag, NodeFlagsNone),
		tagName:         tagName,
		importClause:    importClause,
		moduleSpecifier: moduleSpecifier,
		attributes:      attributes,
		comment:         cloneSlice(comment),
	}
}

func (Factory) SyntaxList(children []Node) SyntaxList {
	return &syntaxListNode{
		nodeCore: newNodeCore(SyntaxKindSyntaxList, NodeFlagsNone),
		children: cloneSlice(children),
	}
}

func (Factory) NotEmittedStatement() NotEmittedStatement {
	return &notEmittedStatementNode{
		nodeCore: newNodeCore(SyntaxKindNotEmittedStatement, NodeFlagsNone),
	}
}

func (Factory) PartiallyEmittedExpression(expression Expression) PartiallyEmittedExpression {
	return &partiallyEmittedExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindPartiallyEmittedExpression, NodeFlagsNone),
		expression: expression,
	}
}

func (Factory) SyntheticReferenceExpression(expression Expression, thisArg Expression) SyntheticReferenceExpression {
	return &syntheticReferenceExpressionNode{
		nodeCore:   newNodeCore(SyntaxKindSyntheticReferenceExpression, NodeFlagsNone),
		expression: expression,
		thisArg:    thisArg,
	}
}

func (Factory) NotEmittedTypeElement() NotEmittedTypeElement {
	return &notEmittedTypeElementNode{
		nodeCore: newNodeCore(SyntaxKindNotEmittedTypeElement, NodeFlagsNone),
	}
}
