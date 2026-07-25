// Code generated from schema/tsgo by go generate. DO NOT EDIT.

package tsgo

type SourceFileData struct {
	Text                    string
	FileName                Path
	Path                    Path
	LanguageVariant         LanguageVariant
	ScriptKind              ScriptKind
	IsDeclarationFile       bool
	ReferencedFiles         []FileReference
	TypeReferenceDirectives []FileReference
	LibReferenceDirectives  []FileReference
	Imports                 []Node
	ModuleAugmentations     []Node
	AmbientModuleNames      []string
	ExternalModuleIndicator Node
}

type HeritageClauseTokenKind SyntaxKind

const (
	HeritageClauseTokenKindExtendsKeyword    HeritageClauseTokenKind = 95
	HeritageClauseTokenKindImplementsKeyword HeritageClauseTokenKind = 118
)

type ImportAttributesTokenKind SyntaxKind

const (
	ImportAttributesTokenKindWithKeyword   ImportAttributesTokenKind = 117
	ImportAttributesTokenKindAssertKeyword ImportAttributesTokenKind = 131
)

type MetaPropertyKeywordTokenKind SyntaxKind

const (
	MetaPropertyKeywordTokenKindImportKeyword MetaPropertyKeywordTokenKind = 101
	MetaPropertyKeywordTokenKindNewKeyword    MetaPropertyKeywordTokenKind = 104
)

type ModuleDeclarationKeywordKind SyntaxKind

const (
	ModuleDeclarationKeywordKindModuleKeyword    ModuleDeclarationKeywordKind = 144
	ModuleDeclarationKeywordKindNamespaceKeyword ModuleDeclarationKeywordKind = 145
)

type PostfixUnaryExpressionOperatorKind SyntaxKind

const (
	PostfixUnaryExpressionOperatorKindPlusPlusToken   PostfixUnaryExpressionOperatorKind = 45
	PostfixUnaryExpressionOperatorKindMinusMinusToken PostfixUnaryExpressionOperatorKind = 46
)

type PrefixUnaryExpressionOperatorKind SyntaxKind

const (
	PrefixUnaryExpressionOperatorKindPlusToken        PrefixUnaryExpressionOperatorKind = 39
	PrefixUnaryExpressionOperatorKindMinusToken       PrefixUnaryExpressionOperatorKind = 40
	PrefixUnaryExpressionOperatorKindTildeToken       PrefixUnaryExpressionOperatorKind = 54
	PrefixUnaryExpressionOperatorKindExclamationToken PrefixUnaryExpressionOperatorKind = 53
	PrefixUnaryExpressionOperatorKindPlusPlusToken    PrefixUnaryExpressionOperatorKind = 45
	PrefixUnaryExpressionOperatorKindMinusMinusToken  PrefixUnaryExpressionOperatorKind = 46
)

type TypeOperatorNodeOperatorKind SyntaxKind

const (
	TypeOperatorNodeOperatorKindKeyOfKeyword    TypeOperatorNodeOperatorKind = 143
	TypeOperatorNodeOperatorKindReadonlyKeyword TypeOperatorNodeOperatorKind = 148
	TypeOperatorNodeOperatorKindUniqueKeyword   TypeOperatorNodeOperatorKind = 158
)

type BodyBase interface {
	Node
	isBodyBase()
}

type ClassElementBase interface {
	Node
	isClassElementBase()
}

type ClassLikeBase interface {
	DeclarationBase
	ModifiersBase
	isClassLikeBase()
}

type DeclarationBase interface {
	Node
	isDeclarationBase()
}

type ExpressionBase interface {
	NodeBase
	isExpressionBase()
}

type FunctionLikeBase interface {
	DeclarationBase
	isFunctionLikeBase()
}

type FunctionLikeWithBodyBase interface {
	FunctionLikeBase
	BodyBase
	isFunctionLikeWithBodyBase()
}

type IterationStatementBase interface {
	StatementBase
	isIterationStatementBase()
}

type JSDocCommentBase interface {
	NodeBase
	isJSDocCommentBase()
}

type JSDocTagBase interface {
	NodeBase
	isJSDocTagBase()
}

type JSDocTypeBase interface {
	TypeNodeBase
	isJSDocTypeBase()
}

type LeftHandSideExpressionBase interface {
	UpdateExpressionBase
	isLeftHandSideExpressionBase()
}

type LiteralExpressionBase interface {
	LiteralLikeNodeBase
	PrimaryExpressionBase
	isLiteralExpressionBase()
}

type LiteralLikeNodeBase interface {
	Node
	isLiteralLikeNodeBase()
}

type MemberExpressionBase interface {
	LeftHandSideExpressionBase
	isMemberExpressionBase()
}

type ModifiersBase interface {
	Node
	isModifiersBase()
}

type NamedMemberBase interface {
	DeclarationBase
	ModifiersBase
	isNamedMemberBase()
}

type NodeBase interface {
	Node
	isNodeBase()
}

type NodeWithTypeArgumentsBase interface {
	TypeNodeBase
	isNodeWithTypeArgumentsBase()
}

type ObjectLiteralElementBase interface {
	Node
	isObjectLiteralElementBase()
}

type PrimaryExpressionBase interface {
	MemberExpressionBase
	isPrimaryExpressionBase()
}

type StatementBase interface {
	NodeBase
	isStatementBase()
}

type TemplateLiteralLikeNodeBase interface {
	LiteralLikeNodeBase
	isTemplateLiteralLikeNodeBase()
}

type TypeElementBase interface {
	Node
	isTypeElementBase()
}

type TypeNodeBase interface {
	NodeBase
	isTypeNodeBase()
}

type UnaryExpressionBase interface {
	ExpressionBase
	isUnaryExpressionBase()
}

type UnionOrIntersectionTypeNodeBase interface {
	TypeNodeBase
	isUnionOrIntersectionTypeNodeBase()
}

type UpdateExpressionBase interface {
	UnaryExpressionBase
	isUpdateExpressionBase()
}

type AccessExpression interface {
	Node
	isAccessExpression()
}

type AccessorDeclaration interface {
	Node
	isAccessorDeclaration()
}

type AnyImportSyntax interface {
	Node
	isAnyImportSyntax()
}

type ArrayBindingElement interface {
	Node
	isArrayBindingElement()
}

type ArrayDestructuringAssignment interface {
	Node
	isArrayDestructuringAssignment()
}

type AssertionExpression interface {
	Node
	isAssertionExpression()
}

type BindingName interface {
	Node
	isBindingName()
}

type BlockOrExpression interface {
	Node
	isBlockOrExpression()
}

type BooleanLiteral interface {
	Node
	isBooleanLiteral()
}

type BreakOrContinueStatement interface {
	Node
	isBreakOrContinueStatement()
}

type CallLikeExpression interface {
	Node
	isCallLikeExpression()
}

type CallOrNewExpression interface {
	Node
	isCallOrNewExpression()
}

type ClassElement interface {
	ClassElementBase
}

type ClassLikeDeclaration interface {
	Node
	isClassLikeDeclaration()
}

type ConciseBody interface {
	Node
	isConciseBody()
}

type Declaration interface {
	DeclarationBase
}

type DeclarationName interface {
	Node
	isDeclarationName()
}

type DestructuringAssignment interface {
	Node
	isDestructuringAssignment()
}

type EntityName interface {
	Node
	isEntityName()
}

type Expression interface {
	ExpressionBase
}

type ForInitializer interface {
	Node
	isForInitializer()
}

type FunctionBody interface {
	Node
	isFunctionBody()
}

type FunctionLikeDeclaration interface {
	Node
	isFunctionLikeDeclaration()
}

type ImportAttributeName interface {
	Node
	isImportAttributeName()
}

type ImportClauseOrBindingPattern interface {
	Node
	isImportClauseOrBindingPattern()
}

type IncrementExpression interface {
	Node
	isIncrementExpression()
}

type JSDocComment interface {
	Node
	isJSDocComment()
}

type JSDocFullName interface {
	Node
	isJSDocFullName()
}

type JSDocTag interface {
	JSDocTagBase
}

type JsxAttributeLike interface {
	Node
	isJsxAttributeLike()
}

type JsxAttributeName interface {
	Node
	isJsxAttributeName()
}

type JsxAttributeValue interface {
	Node
	isJsxAttributeValue()
}

type JsxChild interface {
	Node
	isJsxChild()
}

type JsxOpeningLikeElement interface {
	Node
	isJsxOpeningLikeElement()
}

type JsxTagNameExpression interface {
	Node
	isJsxTagNameExpression()
}

type LeftHandSideExpression interface {
	LeftHandSideExpressionBase
}

type LiteralExpression interface {
	Node
	isLiteralExpression()
}

type LiteralLikeNode interface {
	Node
	isLiteralLikeNode()
}

type LiteralToken interface {
	Node
	isLiteralToken()
}

type MemberName interface {
	Node
	isMemberName()
}

type Modifier interface {
	Node
	isModifier()
}

type ModifierLike interface {
	Node
	isModifierLike()
}

type ModuleBody interface {
	Node
	isModuleBody()
}

type ModuleExportName interface {
	Node
	isModuleExportName()
}

type ModuleName interface {
	Node
	isModuleName()
}

type ModuleReference interface {
	Node
	isModuleReference()
}

type NamedExportBindings interface {
	Node
	isNamedExportBindings()
}

type NamedImportBindings interface {
	Node
	isNamedImportBindings()
}

type NamedImportsOrExports interface {
	Node
	isNamedImportsOrExports()
}

type NodeBody interface {
	Node
	isNodeBody()
}

type NumericOrStringLikeLiteral interface {
	Node
	isNumericOrStringLikeLiteral()
}

type ObjectDestructuringAssignment interface {
	Node
	isObjectDestructuringAssignment()
}

type ObjectLiteralElement interface {
	ObjectLiteralElementBase
}

type ObjectLiteralElementLike interface {
	Node
	isObjectLiteralElementLike()
}

type ObjectLiteralLikeNode interface {
	Node
	isObjectLiteralLikeNode()
}

type ObjectTypeDeclaration interface {
	Node
	isObjectTypeDeclaration()
}

type PropertyName interface {
	Node
	isPropertyName()
}

type PropertyNameLiteral interface {
	Node
	isPropertyNameLiteral()
}

type PseudoLiteralToken interface {
	Node
	isPseudoLiteralToken()
}

type SignatureDeclaration interface {
	Node
	isSignatureDeclaration()
}

type Statement interface {
	StatementBase
}

type StringLiteralLikeNode interface {
	Node
	isStringLiteralLikeNode()
}

type TemplateLiteral interface {
	Node
	isTemplateLiteral()
}

type TemplateLiteralLikeNode interface {
	Node
	isTemplateLiteralLikeNode()
}

type TemplateLiteralToken interface {
	Node
	isTemplateLiteralToken()
}

type TemplateMiddleOrTail interface {
	Node
	isTemplateMiddleOrTail()
}

type TypeElement interface {
	TypeElementBase
}

type TypeNode interface {
	TypeNodeBase
}

type TypePredicateParameterName interface {
	Node
	isTypePredicateParameterName()
}

type UnionOrIntersectionTypeNode interface {
	Node
	isUnionOrIntersectionTypeNode()
}

type VariableOrParameterDeclaration interface {
	Node
	isVariableOrParameterDeclaration()
}

type VariableOrPropertyDeclaration interface {
	Node
	isVariableOrPropertyDeclaration()
}

type MappedTypeNodeQuestionToken interface {
	Node
	isMappedTypeNodeQuestionToken()
}

type MappedTypeNodeReadonlyToken interface {
	Node
	isMappedTypeNodeReadonlyToken()
}

type NamedMemberBasePostfixToken interface {
	Node
	isNamedMemberBasePostfixToken()
}

type AssignmentOperatorToken interface {
	Token
	isAssignmentOperatorToken()
}

type BinaryOperatorToken interface {
	Token
	isBinaryOperatorToken()
}

type ArgumentList []Expression

type BindingElementList []BindingElement

type CaseClausesList []CaseOrDefaultClause

type ClassElementList []ClassElement

type ElementList []Expression

type EnumMemberList []EnumMember

type ExportSpecifierList []ExportSpecifier

type ExpressionWithTypeArgumentsList []ExpressionWithTypeArguments

type HeritageClauseList []HeritageClause

type ImportAttributeList []ImportAttribute

type ImportSpecifierList []ImportSpecifier

type JsxAttributeList []JsxAttributeLike

type JsxChildList []JsxChild

type ParameterList []ParameterDeclaration

type PropertyDefinitionList []ObjectLiteralElement

type StatementList []Statement

type TemplateLiteralTypeSpanList []TemplateLiteralTypeSpan

type TemplateSpanList []TemplateSpan

type TypeArgumentList []TypeNode

type TypeElementList []TypeElement

type TypeList []TypeNode

type TypeParameterList []TypeParameterDeclaration

type VariableDeclarationNodeList []VariableDeclaration

type BindingPattern interface {
	NodeBase
	isBindingPattern()
	Elements() []BindingElement
}

type CaseOrDefaultClause interface {
	NodeBase
	isCaseOrDefaultClause()
	Expression() Expression
	Statements() []Statement
}

type ForInOrOfStatement interface {
	StatementBase
	isForInOrOfStatement()
	AwaitModifier() AwaitKeyword
	Initializer() ForInitializer
	Expression() Expression
	Statement() Statement
}

type JSDocParameterOrPropertyTag interface {
	JSDocTagBase
	isJSDocParameterOrPropertyTag()
	TagName() Identifier
	Name() EntityName
	IsBracketed() bool
	TypeExpression() TypeNode
	IsNameFirst() bool
	Comment() []JSDocComment
}

type KeywordExpression interface {
	ExpressionBase
	isKeywordExpression()
}

type KeywordTypeNode interface {
	TypeNodeBase
	isKeywordTypeNode()
}

type Token interface {
	NodeBase
	isToken()
}

type assignmentOperatorTokenNode struct {
	nodeCore
}

func (*assignmentOperatorTokenNode) isNodeBase()                {}
func (*assignmentOperatorTokenNode) isToken()                   {}
func (*assignmentOperatorTokenNode) isAssignmentOperatorToken() {}

type binaryOperatorTokenNode struct {
	nodeCore
}

func (*binaryOperatorTokenNode) isNodeBase()            {}
func (*binaryOperatorTokenNode) isToken()               {}
func (*binaryOperatorTokenNode) isBinaryOperatorToken() {}

type keywordTypeNodeNode struct {
	nodeCore
}

func (*keywordTypeNodeNode) isNodeBase()        {}
func (*keywordTypeNodeNode) isTypeNodeBase()    {}
func (*keywordTypeNodeNode) isKeywordTypeNode() {}

type tokenNode struct {
	nodeCore
}

func (*tokenNode) isNodeBase() {}
func (*tokenNode) isToken()    {}

type EndOfFile interface {
	Token
	isEndOfFile()
}

type endOfFileNode struct {
	nodeCore
}

func (*endOfFileNode) isNodeBase()  {}
func (*endOfFileNode) isToken()     {}
func (*endOfFileNode) isEndOfFile() {}

type NumericLiteral interface {
	LiteralExpressionBase
	isNumericLiteral()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isIncrementExpression()
	isLiteralExpression()
	isLiteralLikeNode()
	isLiteralToken()
	isNodeBody()
	isNumericOrStringLikeLiteral()
	isPropertyName()
	isPropertyNameLiteral()
	Text() string
	TokenFlags() TokenFlags
}

type numericLiteralNode struct {
	nodeCore
	text       string
	tokenFlags TokenFlags
}

func (*numericLiteralNode) isExpressionBase()             {}
func (*numericLiteralNode) isLeftHandSideExpressionBase() {}
func (*numericLiteralNode) isLiteralExpressionBase()      {}
func (*numericLiteralNode) isLiteralLikeNodeBase()        {}
func (*numericLiteralNode) isMemberExpressionBase()       {}
func (*numericLiteralNode) isNodeBase()                   {}
func (*numericLiteralNode) isPrimaryExpressionBase()      {}
func (*numericLiteralNode) isUnaryExpressionBase()        {}
func (*numericLiteralNode) isUpdateExpressionBase()       {}
func (*numericLiteralNode) isNumericLiteral()             {}
func (*numericLiteralNode) isBlockOrExpression()          {}
func (*numericLiteralNode) isConciseBody()                {}
func (*numericLiteralNode) isDeclarationName()            {}
func (*numericLiteralNode) isForInitializer()             {}
func (*numericLiteralNode) isIncrementExpression()        {}
func (*numericLiteralNode) isLiteralExpression()          {}
func (*numericLiteralNode) isLiteralLikeNode()            {}
func (*numericLiteralNode) isLiteralToken()               {}
func (*numericLiteralNode) isNodeBody()                   {}
func (*numericLiteralNode) isNumericOrStringLikeLiteral() {}
func (*numericLiteralNode) isPropertyName()               {}
func (*numericLiteralNode) isPropertyNameLiteral()        {}

func (n *numericLiteralNode) Text() string {
	return n.text
}

func (n *numericLiteralNode) TokenFlags() TokenFlags {
	return n.tokenFlags
}

type BigIntLiteral interface {
	LiteralExpressionBase
	isBigIntLiteral()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isIncrementExpression()
	isLiteralExpression()
	isLiteralLikeNode()
	isLiteralToken()
	isNodeBody()
	isPropertyName()
	Text() string
	TokenFlags() TokenFlags
}

type bigIntLiteralNode struct {
	nodeCore
	text       string
	tokenFlags TokenFlags
}

func (*bigIntLiteralNode) isExpressionBase()             {}
func (*bigIntLiteralNode) isLeftHandSideExpressionBase() {}
func (*bigIntLiteralNode) isLiteralExpressionBase()      {}
func (*bigIntLiteralNode) isLiteralLikeNodeBase()        {}
func (*bigIntLiteralNode) isMemberExpressionBase()       {}
func (*bigIntLiteralNode) isNodeBase()                   {}
func (*bigIntLiteralNode) isPrimaryExpressionBase()      {}
func (*bigIntLiteralNode) isUnaryExpressionBase()        {}
func (*bigIntLiteralNode) isUpdateExpressionBase()       {}
func (*bigIntLiteralNode) isBigIntLiteral()              {}
func (*bigIntLiteralNode) isBlockOrExpression()          {}
func (*bigIntLiteralNode) isConciseBody()                {}
func (*bigIntLiteralNode) isDeclarationName()            {}
func (*bigIntLiteralNode) isForInitializer()             {}
func (*bigIntLiteralNode) isIncrementExpression()        {}
func (*bigIntLiteralNode) isLiteralExpression()          {}
func (*bigIntLiteralNode) isLiteralLikeNode()            {}
func (*bigIntLiteralNode) isLiteralToken()               {}
func (*bigIntLiteralNode) isNodeBody()                   {}
func (*bigIntLiteralNode) isPropertyName()               {}

func (n *bigIntLiteralNode) Text() string {
	return n.text
}

func (n *bigIntLiteralNode) TokenFlags() TokenFlags {
	return n.tokenFlags
}

type StringLiteral interface {
	LiteralExpressionBase
	isStringLiteral()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isImportAttributeName()
	isIncrementExpression()
	isJsxAttributeValue()
	isLiteralExpression()
	isLiteralLikeNode()
	isLiteralToken()
	isModuleExportName()
	isModuleName()
	isNodeBody()
	isNumericOrStringLikeLiteral()
	isPropertyName()
	isPropertyNameLiteral()
	isStringLiteralLikeNode()
	Text() string
	TokenFlags() TokenFlags
}

type stringLiteralNode struct {
	nodeCore
	text       string
	tokenFlags TokenFlags
}

func (*stringLiteralNode) isExpressionBase()             {}
func (*stringLiteralNode) isLeftHandSideExpressionBase() {}
func (*stringLiteralNode) isLiteralExpressionBase()      {}
func (*stringLiteralNode) isLiteralLikeNodeBase()        {}
func (*stringLiteralNode) isMemberExpressionBase()       {}
func (*stringLiteralNode) isNodeBase()                   {}
func (*stringLiteralNode) isPrimaryExpressionBase()      {}
func (*stringLiteralNode) isUnaryExpressionBase()        {}
func (*stringLiteralNode) isUpdateExpressionBase()       {}
func (*stringLiteralNode) isStringLiteral()              {}
func (*stringLiteralNode) isBlockOrExpression()          {}
func (*stringLiteralNode) isConciseBody()                {}
func (*stringLiteralNode) isDeclarationName()            {}
func (*stringLiteralNode) isForInitializer()             {}
func (*stringLiteralNode) isImportAttributeName()        {}
func (*stringLiteralNode) isIncrementExpression()        {}
func (*stringLiteralNode) isJsxAttributeValue()          {}
func (*stringLiteralNode) isLiteralExpression()          {}
func (*stringLiteralNode) isLiteralLikeNode()            {}
func (*stringLiteralNode) isLiteralToken()               {}
func (*stringLiteralNode) isModuleExportName()           {}
func (*stringLiteralNode) isModuleName()                 {}
func (*stringLiteralNode) isNodeBody()                   {}
func (*stringLiteralNode) isNumericOrStringLikeLiteral() {}
func (*stringLiteralNode) isPropertyName()               {}
func (*stringLiteralNode) isPropertyNameLiteral()        {}
func (*stringLiteralNode) isStringLiteralLikeNode()      {}

func (n *stringLiteralNode) Text() string {
	return n.text
}

func (n *stringLiteralNode) TokenFlags() TokenFlags {
	return n.tokenFlags
}

type JsxText interface {
	ExpressionBase
	LiteralLikeNodeBase
	isJsxText()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxChild()
	isLiteralLikeNode()
	isLiteralToken()
	isNodeBody()
	Text() string
	ContainsOnlyTriviaWhiteSpaces() bool
}

type jsxTextNode struct {
	nodeCore
	text                          string
	containsOnlyTriviaWhiteSpaces bool
}

func (*jsxTextNode) isExpressionBase()      {}
func (*jsxTextNode) isLiteralLikeNodeBase() {}
func (*jsxTextNode) isNodeBase()            {}
func (*jsxTextNode) isJsxText()             {}
func (*jsxTextNode) isBlockOrExpression()   {}
func (*jsxTextNode) isConciseBody()         {}
func (*jsxTextNode) isForInitializer()      {}
func (*jsxTextNode) isJsxChild()            {}
func (*jsxTextNode) isLiteralLikeNode()     {}
func (*jsxTextNode) isLiteralToken()        {}
func (*jsxTextNode) isNodeBody()            {}

func (n *jsxTextNode) Text() string {
	return n.text
}

func (n *jsxTextNode) ContainsOnlyTriviaWhiteSpaces() bool {
	return n.containsOnlyTriviaWhiteSpaces
}

type RegularExpressionLiteral interface {
	LiteralExpressionBase
	isRegularExpressionLiteral()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isLiteralExpression()
	isLiteralLikeNode()
	isLiteralToken()
	isNodeBody()
	Text() string
	TokenFlags() TokenFlags
}

type regularExpressionLiteralNode struct {
	nodeCore
	text       string
	tokenFlags TokenFlags
}

func (*regularExpressionLiteralNode) isExpressionBase()             {}
func (*regularExpressionLiteralNode) isLeftHandSideExpressionBase() {}
func (*regularExpressionLiteralNode) isLiteralExpressionBase()      {}
func (*regularExpressionLiteralNode) isLiteralLikeNodeBase()        {}
func (*regularExpressionLiteralNode) isMemberExpressionBase()       {}
func (*regularExpressionLiteralNode) isNodeBase()                   {}
func (*regularExpressionLiteralNode) isPrimaryExpressionBase()      {}
func (*regularExpressionLiteralNode) isUnaryExpressionBase()        {}
func (*regularExpressionLiteralNode) isUpdateExpressionBase()       {}
func (*regularExpressionLiteralNode) isRegularExpressionLiteral()   {}
func (*regularExpressionLiteralNode) isBlockOrExpression()          {}
func (*regularExpressionLiteralNode) isConciseBody()                {}
func (*regularExpressionLiteralNode) isForInitializer()             {}
func (*regularExpressionLiteralNode) isIncrementExpression()        {}
func (*regularExpressionLiteralNode) isLiteralExpression()          {}
func (*regularExpressionLiteralNode) isLiteralLikeNode()            {}
func (*regularExpressionLiteralNode) isLiteralToken()               {}
func (*regularExpressionLiteralNode) isNodeBody()                   {}

func (n *regularExpressionLiteralNode) Text() string {
	return n.text
}

func (n *regularExpressionLiteralNode) TokenFlags() TokenFlags {
	return n.tokenFlags
}

type NoSubstitutionTemplateLiteral interface {
	ExpressionBase
	TemplateLiteralLikeNodeBase
	DeclarationBase
	isNoSubstitutionTemplateLiteral()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isLiteralExpression()
	isLiteralToken()
	isNodeBody()
	isNumericOrStringLikeLiteral()
	isPropertyName()
	isStringLiteralLikeNode()
	isTemplateLiteral()
	isTemplateLiteralToken()
	Text() string
	TemplateFlags() TokenFlags
}

type noSubstitutionTemplateLiteralNode struct {
	nodeCore
	text          string
	templateFlags TokenFlags
}

func (*noSubstitutionTemplateLiteralNode) isDeclarationBase()               {}
func (*noSubstitutionTemplateLiteralNode) isExpressionBase()                {}
func (*noSubstitutionTemplateLiteralNode) isLiteralLikeNodeBase()           {}
func (*noSubstitutionTemplateLiteralNode) isNodeBase()                      {}
func (*noSubstitutionTemplateLiteralNode) isTemplateLiteralLikeNodeBase()   {}
func (*noSubstitutionTemplateLiteralNode) isNoSubstitutionTemplateLiteral() {}
func (*noSubstitutionTemplateLiteralNode) isBlockOrExpression()             {}
func (*noSubstitutionTemplateLiteralNode) isConciseBody()                   {}
func (*noSubstitutionTemplateLiteralNode) isDeclarationName()               {}
func (*noSubstitutionTemplateLiteralNode) isForInitializer()                {}
func (*noSubstitutionTemplateLiteralNode) isLiteralExpression()             {}
func (*noSubstitutionTemplateLiteralNode) isLiteralToken()                  {}
func (*noSubstitutionTemplateLiteralNode) isNodeBody()                      {}
func (*noSubstitutionTemplateLiteralNode) isNumericOrStringLikeLiteral()    {}
func (*noSubstitutionTemplateLiteralNode) isPropertyName()                  {}
func (*noSubstitutionTemplateLiteralNode) isStringLiteralLikeNode()         {}
func (*noSubstitutionTemplateLiteralNode) isTemplateLiteral()               {}
func (*noSubstitutionTemplateLiteralNode) isTemplateLiteralToken()          {}

func (n *noSubstitutionTemplateLiteralNode) Text() string {
	return n.text
}

func (n *noSubstitutionTemplateLiteralNode) TemplateFlags() TokenFlags {
	return n.templateFlags
}

type TemplateHead interface {
	NodeBase
	TemplateLiteralLikeNodeBase
	isTemplateHead()
	isLiteralLikeNode()
	isPseudoLiteralToken()
	isTemplateLiteralLikeNode()
	isTemplateLiteralToken()
	Text() string
	RawText() string
	TemplateFlags() TokenFlags
}

type templateHeadNode struct {
	nodeCore
	text          string
	rawText       string
	templateFlags TokenFlags
}

func (*templateHeadNode) isLiteralLikeNodeBase()         {}
func (*templateHeadNode) isNodeBase()                    {}
func (*templateHeadNode) isTemplateLiteralLikeNodeBase() {}
func (*templateHeadNode) isTemplateHead()                {}
func (*templateHeadNode) isLiteralLikeNode()             {}
func (*templateHeadNode) isPseudoLiteralToken()          {}
func (*templateHeadNode) isTemplateLiteralLikeNode()     {}
func (*templateHeadNode) isTemplateLiteralToken()        {}

func (n *templateHeadNode) Text() string {
	return n.text
}

func (n *templateHeadNode) RawText() string {
	return n.rawText
}

func (n *templateHeadNode) TemplateFlags() TokenFlags {
	return n.templateFlags
}

type TemplateMiddle interface {
	NodeBase
	TemplateLiteralLikeNodeBase
	isTemplateMiddle()
	isLiteralLikeNode()
	isPseudoLiteralToken()
	isTemplateLiteralLikeNode()
	isTemplateLiteralToken()
	isTemplateMiddleOrTail()
	Text() string
	RawText() string
	TemplateFlags() TokenFlags
}

type templateMiddleNode struct {
	nodeCore
	text          string
	rawText       string
	templateFlags TokenFlags
}

func (*templateMiddleNode) isLiteralLikeNodeBase()         {}
func (*templateMiddleNode) isNodeBase()                    {}
func (*templateMiddleNode) isTemplateLiteralLikeNodeBase() {}
func (*templateMiddleNode) isTemplateMiddle()              {}
func (*templateMiddleNode) isLiteralLikeNode()             {}
func (*templateMiddleNode) isPseudoLiteralToken()          {}
func (*templateMiddleNode) isTemplateLiteralLikeNode()     {}
func (*templateMiddleNode) isTemplateLiteralToken()        {}
func (*templateMiddleNode) isTemplateMiddleOrTail()        {}

func (n *templateMiddleNode) Text() string {
	return n.text
}

func (n *templateMiddleNode) RawText() string {
	return n.rawText
}

func (n *templateMiddleNode) TemplateFlags() TokenFlags {
	return n.templateFlags
}

type TemplateTail interface {
	NodeBase
	TemplateLiteralLikeNodeBase
	isTemplateTail()
	isLiteralLikeNode()
	isPseudoLiteralToken()
	isTemplateLiteralLikeNode()
	isTemplateLiteralToken()
	isTemplateMiddleOrTail()
	Text() string
	RawText() string
	TemplateFlags() TokenFlags
}

type templateTailNode struct {
	nodeCore
	text          string
	rawText       string
	templateFlags TokenFlags
}

func (*templateTailNode) isLiteralLikeNodeBase()         {}
func (*templateTailNode) isNodeBase()                    {}
func (*templateTailNode) isTemplateLiteralLikeNodeBase() {}
func (*templateTailNode) isTemplateTail()                {}
func (*templateTailNode) isLiteralLikeNode()             {}
func (*templateTailNode) isPseudoLiteralToken()          {}
func (*templateTailNode) isTemplateLiteralLikeNode()     {}
func (*templateTailNode) isTemplateLiteralToken()        {}
func (*templateTailNode) isTemplateMiddleOrTail()        {}

func (n *templateTailNode) Text() string {
	return n.text
}

func (n *templateTailNode) RawText() string {
	return n.rawText
}

func (n *templateTailNode) TemplateFlags() TokenFlags {
	return n.templateFlags
}

type DotToken interface {
	Token
	isDotToken()
}

type dotTokenNode struct {
	nodeCore
}

func (*dotTokenNode) isNodeBase() {}
func (*dotTokenNode) isToken()    {}
func (*dotTokenNode) isDotToken() {}

type DotDotDotToken interface {
	Token
	isDotDotDotToken()
}

type dotDotDotTokenNode struct {
	nodeCore
}

func (*dotDotDotTokenNode) isNodeBase()       {}
func (*dotDotDotTokenNode) isToken()          {}
func (*dotDotDotTokenNode) isDotDotDotToken() {}

type QuestionDotToken interface {
	Token
	isQuestionDotToken()
}

type questionDotTokenNode struct {
	nodeCore
}

func (*questionDotTokenNode) isNodeBase()         {}
func (*questionDotTokenNode) isToken()            {}
func (*questionDotTokenNode) isQuestionDotToken() {}

type EqualsGreaterThanToken interface {
	Token
	isEqualsGreaterThanToken()
}

type equalsGreaterThanTokenNode struct {
	nodeCore
}

func (*equalsGreaterThanTokenNode) isNodeBase()               {}
func (*equalsGreaterThanTokenNode) isToken()                  {}
func (*equalsGreaterThanTokenNode) isEqualsGreaterThanToken() {}

type PlusToken interface {
	Token
	isPlusToken()
	isBinaryOperatorToken()
	isMappedTypeNodeQuestionToken()
	isMappedTypeNodeReadonlyToken()
}

type plusTokenNode struct {
	nodeCore
}

func (*plusTokenNode) isNodeBase()                    {}
func (*plusTokenNode) isToken()                       {}
func (*plusTokenNode) isPlusToken()                   {}
func (*plusTokenNode) isBinaryOperatorToken()         {}
func (*plusTokenNode) isMappedTypeNodeQuestionToken() {}
func (*plusTokenNode) isMappedTypeNodeReadonlyToken() {}

type MinusToken interface {
	Token
	isMinusToken()
	isBinaryOperatorToken()
	isMappedTypeNodeQuestionToken()
	isMappedTypeNodeReadonlyToken()
}

type minusTokenNode struct {
	nodeCore
}

func (*minusTokenNode) isNodeBase()                    {}
func (*minusTokenNode) isToken()                       {}
func (*minusTokenNode) isMinusToken()                  {}
func (*minusTokenNode) isBinaryOperatorToken()         {}
func (*minusTokenNode) isMappedTypeNodeQuestionToken() {}
func (*minusTokenNode) isMappedTypeNodeReadonlyToken() {}

type AsteriskToken interface {
	Token
	isAsteriskToken()
	isBinaryOperatorToken()
}

type asteriskTokenNode struct {
	nodeCore
}

func (*asteriskTokenNode) isNodeBase()            {}
func (*asteriskTokenNode) isToken()               {}
func (*asteriskTokenNode) isAsteriskToken()       {}
func (*asteriskTokenNode) isBinaryOperatorToken() {}

type ExclamationToken interface {
	Token
	isExclamationToken()
	isNamedMemberBasePostfixToken()
}

type exclamationTokenNode struct {
	nodeCore
}

func (*exclamationTokenNode) isNodeBase()                    {}
func (*exclamationTokenNode) isToken()                       {}
func (*exclamationTokenNode) isExclamationToken()            {}
func (*exclamationTokenNode) isNamedMemberBasePostfixToken() {}

type QuestionToken interface {
	Token
	isQuestionToken()
	isMappedTypeNodeQuestionToken()
	isNamedMemberBasePostfixToken()
}

type questionTokenNode struct {
	nodeCore
}

func (*questionTokenNode) isNodeBase()                    {}
func (*questionTokenNode) isToken()                       {}
func (*questionTokenNode) isQuestionToken()               {}
func (*questionTokenNode) isMappedTypeNodeQuestionToken() {}
func (*questionTokenNode) isNamedMemberBasePostfixToken() {}

type ColonToken interface {
	Token
	isColonToken()
}

type colonTokenNode struct {
	nodeCore
}

func (*colonTokenNode) isNodeBase()   {}
func (*colonTokenNode) isToken()      {}
func (*colonTokenNode) isColonToken() {}

type EqualsToken interface {
	Token
	isEqualsToken()
	isAssignmentOperatorToken()
	isBinaryOperatorToken()
}

type equalsTokenNode struct {
	nodeCore
}

func (*equalsTokenNode) isNodeBase()                {}
func (*equalsTokenNode) isToken()                   {}
func (*equalsTokenNode) isEqualsToken()             {}
func (*equalsTokenNode) isAssignmentOperatorToken() {}
func (*equalsTokenNode) isBinaryOperatorToken()     {}

type Identifier interface {
	PrimaryExpressionBase
	isIdentifier()
	isBindingName()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isEntityName()
	isForInitializer()
	isImportAttributeName()
	isIncrementExpression()
	isJSDocFullName()
	isJsxAttributeName()
	isJsxTagNameExpression()
	isMemberName()
	isModuleExportName()
	isModuleName()
	isModuleReference()
	isNodeBody()
	isPropertyName()
	isPropertyNameLiteral()
	isTypePredicateParameterName()
	Text() string
}

type identifierNode struct {
	nodeCore
	text string
}

func (*identifierNode) isExpressionBase()             {}
func (*identifierNode) isLeftHandSideExpressionBase() {}
func (*identifierNode) isMemberExpressionBase()       {}
func (*identifierNode) isNodeBase()                   {}
func (*identifierNode) isPrimaryExpressionBase()      {}
func (*identifierNode) isUnaryExpressionBase()        {}
func (*identifierNode) isUpdateExpressionBase()       {}
func (*identifierNode) isIdentifier()                 {}
func (*identifierNode) isBindingName()                {}
func (*identifierNode) isBlockOrExpression()          {}
func (*identifierNode) isConciseBody()                {}
func (*identifierNode) isDeclarationName()            {}
func (*identifierNode) isEntityName()                 {}
func (*identifierNode) isForInitializer()             {}
func (*identifierNode) isImportAttributeName()        {}
func (*identifierNode) isIncrementExpression()        {}
func (*identifierNode) isJSDocFullName()              {}
func (*identifierNode) isJsxAttributeName()           {}
func (*identifierNode) isJsxTagNameExpression()       {}
func (*identifierNode) isMemberName()                 {}
func (*identifierNode) isModuleExportName()           {}
func (*identifierNode) isModuleName()                 {}
func (*identifierNode) isModuleReference()            {}
func (*identifierNode) isNodeBody()                   {}
func (*identifierNode) isPropertyName()               {}
func (*identifierNode) isPropertyNameLiteral()        {}
func (*identifierNode) isTypePredicateParameterName() {}

func (n *identifierNode) Text() string {
	return n.text
}

type PrivateIdentifier interface {
	PrimaryExpressionBase
	isPrivateIdentifier()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isIncrementExpression()
	isMemberName()
	isNodeBody()
	isPropertyName()
	Text() string
}

type privateIdentifierNode struct {
	nodeCore
	text string
}

func (*privateIdentifierNode) isExpressionBase()             {}
func (*privateIdentifierNode) isLeftHandSideExpressionBase() {}
func (*privateIdentifierNode) isMemberExpressionBase()       {}
func (*privateIdentifierNode) isNodeBase()                   {}
func (*privateIdentifierNode) isPrimaryExpressionBase()      {}
func (*privateIdentifierNode) isUnaryExpressionBase()        {}
func (*privateIdentifierNode) isUpdateExpressionBase()       {}
func (*privateIdentifierNode) isPrivateIdentifier()          {}
func (*privateIdentifierNode) isBlockOrExpression()          {}
func (*privateIdentifierNode) isConciseBody()                {}
func (*privateIdentifierNode) isDeclarationName()            {}
func (*privateIdentifierNode) isForInitializer()             {}
func (*privateIdentifierNode) isIncrementExpression()        {}
func (*privateIdentifierNode) isMemberName()                 {}
func (*privateIdentifierNode) isNodeBody()                   {}
func (*privateIdentifierNode) isPropertyName()               {}

func (n *privateIdentifierNode) Text() string {
	return n.text
}

type CaseKeyword interface {
	Token
	isCaseKeyword()
}

type caseKeywordNode struct {
	nodeCore
}

func (*caseKeywordNode) isNodeBase()    {}
func (*caseKeywordNode) isToken()       {}
func (*caseKeywordNode) isCaseKeyword() {}

type ConstKeyword interface {
	Token
	isConstKeyword()
	isModifier()
	isModifierLike()
}

type constKeywordNode struct {
	nodeCore
}

func (*constKeywordNode) isNodeBase()     {}
func (*constKeywordNode) isToken()        {}
func (*constKeywordNode) isConstKeyword() {}
func (*constKeywordNode) isModifier()     {}
func (*constKeywordNode) isModifierLike() {}

type DefaultKeyword interface {
	Token
	isDefaultKeyword()
	isModifier()
	isModifierLike()
}

type defaultKeywordNode struct {
	nodeCore
}

func (*defaultKeywordNode) isNodeBase()       {}
func (*defaultKeywordNode) isToken()          {}
func (*defaultKeywordNode) isDefaultKeyword() {}
func (*defaultKeywordNode) isModifier()       {}
func (*defaultKeywordNode) isModifierLike()   {}

type ExportKeyword interface {
	Token
	isExportKeyword()
	isModifier()
	isModifierLike()
}

type exportKeywordNode struct {
	nodeCore
}

func (*exportKeywordNode) isNodeBase()      {}
func (*exportKeywordNode) isToken()         {}
func (*exportKeywordNode) isExportKeyword() {}
func (*exportKeywordNode) isModifier()      {}
func (*exportKeywordNode) isModifierLike()  {}

type FalseLiteral interface {
	KeywordExpression
	isFalseLiteral()
	isBlockOrExpression()
	isBooleanLiteral()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type falseLiteralNode struct {
	nodeCore
}

func (*falseLiteralNode) isExpressionBase()       {}
func (*falseLiteralNode) isNodeBase()             {}
func (*falseLiteralNode) isKeywordExpression()    {}
func (*falseLiteralNode) isFalseLiteral()         {}
func (*falseLiteralNode) isBlockOrExpression()    {}
func (*falseLiteralNode) isBooleanLiteral()       {}
func (*falseLiteralNode) isConciseBody()          {}
func (*falseLiteralNode) isForInitializer()       {}
func (*falseLiteralNode) isJsxTagNameExpression() {}
func (*falseLiteralNode) isNodeBody()             {}

type ImportExpression interface {
	KeywordExpression
	isImportExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type importExpressionNode struct {
	nodeCore
}

func (*importExpressionNode) isExpressionBase()       {}
func (*importExpressionNode) isNodeBase()             {}
func (*importExpressionNode) isKeywordExpression()    {}
func (*importExpressionNode) isImportExpression()     {}
func (*importExpressionNode) isBlockOrExpression()    {}
func (*importExpressionNode) isConciseBody()          {}
func (*importExpressionNode) isForInitializer()       {}
func (*importExpressionNode) isJsxTagNameExpression() {}
func (*importExpressionNode) isNodeBody()             {}

type InKeyword interface {
	Token
	isInKeyword()
	isBinaryOperatorToken()
	isModifier()
	isModifierLike()
}

type inKeywordNode struct {
	nodeCore
}

func (*inKeywordNode) isNodeBase()            {}
func (*inKeywordNode) isToken()               {}
func (*inKeywordNode) isInKeyword()           {}
func (*inKeywordNode) isBinaryOperatorToken() {}
func (*inKeywordNode) isModifier()            {}
func (*inKeywordNode) isModifierLike()        {}

type NullLiteral interface {
	KeywordExpression
	isNullLiteral()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type nullLiteralNode struct {
	nodeCore
}

func (*nullLiteralNode) isExpressionBase()       {}
func (*nullLiteralNode) isNodeBase()             {}
func (*nullLiteralNode) isKeywordExpression()    {}
func (*nullLiteralNode) isNullLiteral()          {}
func (*nullLiteralNode) isBlockOrExpression()    {}
func (*nullLiteralNode) isConciseBody()          {}
func (*nullLiteralNode) isForInitializer()       {}
func (*nullLiteralNode) isJsxTagNameExpression() {}
func (*nullLiteralNode) isNodeBody()             {}

type SuperExpression interface {
	KeywordExpression
	isSuperExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type superExpressionNode struct {
	nodeCore
}

func (*superExpressionNode) isExpressionBase()       {}
func (*superExpressionNode) isNodeBase()             {}
func (*superExpressionNode) isKeywordExpression()    {}
func (*superExpressionNode) isSuperExpression()      {}
func (*superExpressionNode) isBlockOrExpression()    {}
func (*superExpressionNode) isConciseBody()          {}
func (*superExpressionNode) isForInitializer()       {}
func (*superExpressionNode) isJsxTagNameExpression() {}
func (*superExpressionNode) isNodeBody()             {}

type ThisExpression interface {
	KeywordExpression
	isThisExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type thisExpressionNode struct {
	nodeCore
}

func (*thisExpressionNode) isExpressionBase()       {}
func (*thisExpressionNode) isNodeBase()             {}
func (*thisExpressionNode) isKeywordExpression()    {}
func (*thisExpressionNode) isThisExpression()       {}
func (*thisExpressionNode) isBlockOrExpression()    {}
func (*thisExpressionNode) isConciseBody()          {}
func (*thisExpressionNode) isForInitializer()       {}
func (*thisExpressionNode) isJsxTagNameExpression() {}
func (*thisExpressionNode) isNodeBody()             {}

type TrueLiteral interface {
	KeywordExpression
	isTrueLiteral()
	isBlockOrExpression()
	isBooleanLiteral()
	isConciseBody()
	isForInitializer()
	isJsxTagNameExpression()
	isNodeBody()
}

type trueLiteralNode struct {
	nodeCore
}

func (*trueLiteralNode) isExpressionBase()       {}
func (*trueLiteralNode) isNodeBase()             {}
func (*trueLiteralNode) isKeywordExpression()    {}
func (*trueLiteralNode) isTrueLiteral()          {}
func (*trueLiteralNode) isBlockOrExpression()    {}
func (*trueLiteralNode) isBooleanLiteral()       {}
func (*trueLiteralNode) isConciseBody()          {}
func (*trueLiteralNode) isForInitializer()       {}
func (*trueLiteralNode) isJsxTagNameExpression() {}
func (*trueLiteralNode) isNodeBody()             {}

type PrivateKeyword interface {
	Token
	isPrivateKeyword()
	isModifier()
	isModifierLike()
}

type privateKeywordNode struct {
	nodeCore
}

func (*privateKeywordNode) isNodeBase()       {}
func (*privateKeywordNode) isToken()          {}
func (*privateKeywordNode) isPrivateKeyword() {}
func (*privateKeywordNode) isModifier()       {}
func (*privateKeywordNode) isModifierLike()   {}

type ProtectedKeyword interface {
	Token
	isProtectedKeyword()
	isModifier()
	isModifierLike()
}

type protectedKeywordNode struct {
	nodeCore
}

func (*protectedKeywordNode) isNodeBase()         {}
func (*protectedKeywordNode) isToken()            {}
func (*protectedKeywordNode) isProtectedKeyword() {}
func (*protectedKeywordNode) isModifier()         {}
func (*protectedKeywordNode) isModifierLike()     {}

type PublicKeyword interface {
	Token
	isPublicKeyword()
	isModifier()
	isModifierLike()
}

type publicKeywordNode struct {
	nodeCore
}

func (*publicKeywordNode) isNodeBase()      {}
func (*publicKeywordNode) isToken()         {}
func (*publicKeywordNode) isPublicKeyword() {}
func (*publicKeywordNode) isModifier()      {}
func (*publicKeywordNode) isModifierLike()  {}

type StaticKeyword interface {
	Token
	isStaticKeyword()
	isModifier()
	isModifierLike()
}

type staticKeywordNode struct {
	nodeCore
}

func (*staticKeywordNode) isNodeBase()      {}
func (*staticKeywordNode) isToken()         {}
func (*staticKeywordNode) isStaticKeyword() {}
func (*staticKeywordNode) isModifier()      {}
func (*staticKeywordNode) isModifierLike()  {}

type AbstractKeyword interface {
	Token
	isAbstractKeyword()
	isModifier()
	isModifierLike()
}

type abstractKeywordNode struct {
	nodeCore
}

func (*abstractKeywordNode) isNodeBase()        {}
func (*abstractKeywordNode) isToken()           {}
func (*abstractKeywordNode) isAbstractKeyword() {}
func (*abstractKeywordNode) isModifier()        {}
func (*abstractKeywordNode) isModifierLike()    {}

type AccessorKeyword interface {
	Token
	isAccessorKeyword()
	isModifier()
	isModifierLike()
}

type accessorKeywordNode struct {
	nodeCore
}

func (*accessorKeywordNode) isNodeBase()        {}
func (*accessorKeywordNode) isToken()           {}
func (*accessorKeywordNode) isAccessorKeyword() {}
func (*accessorKeywordNode) isModifier()        {}
func (*accessorKeywordNode) isModifierLike()    {}

type AssertsKeyword interface {
	Token
	isAssertsKeyword()
}

type assertsKeywordNode struct {
	nodeCore
}

func (*assertsKeywordNode) isNodeBase()       {}
func (*assertsKeywordNode) isToken()          {}
func (*assertsKeywordNode) isAssertsKeyword() {}

type AssertKeyword interface {
	Token
	isAssertKeyword()
}

type assertKeywordNode struct {
	nodeCore
}

func (*assertKeywordNode) isNodeBase()      {}
func (*assertKeywordNode) isToken()         {}
func (*assertKeywordNode) isAssertKeyword() {}

type AsyncKeyword interface {
	Token
	isAsyncKeyword()
	isModifier()
	isModifierLike()
}

type asyncKeywordNode struct {
	nodeCore
}

func (*asyncKeywordNode) isNodeBase()     {}
func (*asyncKeywordNode) isToken()        {}
func (*asyncKeywordNode) isAsyncKeyword() {}
func (*asyncKeywordNode) isModifier()     {}
func (*asyncKeywordNode) isModifierLike() {}

type AwaitKeyword interface {
	Token
	isAwaitKeyword()
}

type awaitKeywordNode struct {
	nodeCore
}

func (*awaitKeywordNode) isNodeBase()     {}
func (*awaitKeywordNode) isToken()        {}
func (*awaitKeywordNode) isAwaitKeyword() {}

type DeclareKeyword interface {
	Token
	isDeclareKeyword()
	isModifier()
	isModifierLike()
}

type declareKeywordNode struct {
	nodeCore
}

func (*declareKeywordNode) isNodeBase()       {}
func (*declareKeywordNode) isToken()          {}
func (*declareKeywordNode) isDeclareKeyword() {}
func (*declareKeywordNode) isModifier()       {}
func (*declareKeywordNode) isModifierLike()   {}

type OutKeyword interface {
	Token
	isOutKeyword()
	isModifier()
	isModifierLike()
}

type outKeywordNode struct {
	nodeCore
}

func (*outKeywordNode) isNodeBase()     {}
func (*outKeywordNode) isToken()        {}
func (*outKeywordNode) isOutKeyword()   {}
func (*outKeywordNode) isModifier()     {}
func (*outKeywordNode) isModifierLike() {}

type ReadonlyKeyword interface {
	Token
	isReadonlyKeyword()
	isMappedTypeNodeReadonlyToken()
	isModifier()
	isModifierLike()
}

type readonlyKeywordNode struct {
	nodeCore
}

func (*readonlyKeywordNode) isNodeBase()                    {}
func (*readonlyKeywordNode) isToken()                       {}
func (*readonlyKeywordNode) isReadonlyKeyword()             {}
func (*readonlyKeywordNode) isMappedTypeNodeReadonlyToken() {}
func (*readonlyKeywordNode) isModifier()                    {}
func (*readonlyKeywordNode) isModifierLike()                {}

type OverrideKeyword interface {
	Token
	isOverrideKeyword()
	isModifier()
	isModifierLike()
}

type overrideKeywordNode struct {
	nodeCore
}

func (*overrideKeywordNode) isNodeBase()        {}
func (*overrideKeywordNode) isToken()           {}
func (*overrideKeywordNode) isOverrideKeyword() {}
func (*overrideKeywordNode) isModifier()        {}
func (*overrideKeywordNode) isModifierLike()    {}

type QualifiedName interface {
	NodeBase
	isQualifiedName()
	isEntityName()
	isModuleReference()
	Left() EntityName
	Right() Identifier
}

type qualifiedNameNode struct {
	nodeCore
	left  EntityName
	right Identifier
}

func (*qualifiedNameNode) isNodeBase()        {}
func (*qualifiedNameNode) isQualifiedName()   {}
func (*qualifiedNameNode) isEntityName()      {}
func (*qualifiedNameNode) isModuleReference() {}

func (n *qualifiedNameNode) Left() EntityName {
	return n.left
}

func (n *qualifiedNameNode) Right() Identifier {
	return n.right
}

type ComputedPropertyName interface {
	NodeBase
	isComputedPropertyName()
	isDeclarationName()
	isPropertyName()
	Expression() Expression
}

type computedPropertyNameNode struct {
	nodeCore
	expression Expression
}

func (*computedPropertyNameNode) isNodeBase()             {}
func (*computedPropertyNameNode) isComputedPropertyName() {}
func (*computedPropertyNameNode) isDeclarationName()      {}
func (*computedPropertyNameNode) isPropertyName()         {}

func (n *computedPropertyNameNode) Expression() Expression {
	return n.expression
}

type TypeParameterDeclaration interface {
	NodeBase
	DeclarationBase
	ModifiersBase
	isTypeParameterDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	Constraint() TypeNode
	Expression() Expression
	DefaultType() TypeNode
}

type typeParameterDeclarationNode struct {
	nodeCore
	modifiers   []ModifierLike
	name        Identifier
	constraint  TypeNode
	expression  Expression
	defaultType TypeNode
}

func (*typeParameterDeclarationNode) isDeclarationBase()          {}
func (*typeParameterDeclarationNode) isModifiersBase()            {}
func (*typeParameterDeclarationNode) isNodeBase()                 {}
func (*typeParameterDeclarationNode) isTypeParameterDeclaration() {}

func (n *typeParameterDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *typeParameterDeclarationNode) Name() Identifier {
	return n.name
}

func (n *typeParameterDeclarationNode) Constraint() TypeNode {
	return n.constraint
}

func (n *typeParameterDeclarationNode) Expression() Expression {
	return n.expression
}

func (n *typeParameterDeclarationNode) DefaultType() TypeNode {
	return n.defaultType
}

type ParameterDeclaration interface {
	NodeBase
	DeclarationBase
	ModifiersBase
	isParameterDeclaration()
	isVariableOrParameterDeclaration()
	Modifiers() []ModifierLike
	DotDotDotToken() DotDotDotToken
	Name() BindingName
	QuestionToken() QuestionToken
	Type() TypeNode
	Initializer() Expression
}

type parameterDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	dotDotDotToken DotDotDotToken
	name           BindingName
	questionToken  QuestionToken
	typeNode       TypeNode
	initializer    Expression
}

func (*parameterDeclarationNode) isDeclarationBase()                {}
func (*parameterDeclarationNode) isModifiersBase()                  {}
func (*parameterDeclarationNode) isNodeBase()                       {}
func (*parameterDeclarationNode) isParameterDeclaration()           {}
func (*parameterDeclarationNode) isVariableOrParameterDeclaration() {}

func (n *parameterDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *parameterDeclarationNode) DotDotDotToken() DotDotDotToken {
	return n.dotDotDotToken
}

func (n *parameterDeclarationNode) Name() BindingName {
	return n.name
}

func (n *parameterDeclarationNode) QuestionToken() QuestionToken {
	return n.questionToken
}

func (n *parameterDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *parameterDeclarationNode) Initializer() Expression {
	return n.initializer
}

type Decorator interface {
	NodeBase
	isDecorator()
	isCallLikeExpression()
	isModifierLike()
	Expression() LeftHandSideExpression
}

type decoratorNode struct {
	nodeCore
	expression LeftHandSideExpression
}

func (*decoratorNode) isNodeBase()           {}
func (*decoratorNode) isDecorator()          {}
func (*decoratorNode) isCallLikeExpression() {}
func (*decoratorNode) isModifierLike()       {}

func (n *decoratorNode) Expression() LeftHandSideExpression {
	return n.expression
}

type PropertySignatureDeclaration interface {
	NodeBase
	NamedMemberBase
	TypeElementBase
	isPropertySignatureDeclaration()
	Modifiers() []ModifierLike
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	Type() TypeNode
	Initializer() Expression
}

type propertySignatureDeclarationNode struct {
	nodeCore
	modifiers    []ModifierLike
	name         PropertyName
	postfixToken NamedMemberBasePostfixToken
	typeNode     TypeNode
	initializer  Expression
}

func (*propertySignatureDeclarationNode) isDeclarationBase()              {}
func (*propertySignatureDeclarationNode) isModifiersBase()                {}
func (*propertySignatureDeclarationNode) isNamedMemberBase()              {}
func (*propertySignatureDeclarationNode) isNodeBase()                     {}
func (*propertySignatureDeclarationNode) isTypeElementBase()              {}
func (*propertySignatureDeclarationNode) isPropertySignatureDeclaration() {}

func (n *propertySignatureDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *propertySignatureDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *propertySignatureDeclarationNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *propertySignatureDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *propertySignatureDeclarationNode) Initializer() Expression {
	return n.initializer
}

type PropertyDeclaration interface {
	NodeBase
	NamedMemberBase
	ClassElementBase
	isPropertyDeclaration()
	isVariableOrPropertyDeclaration()
	Modifiers() []ModifierLike
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	Type() TypeNode
	Initializer() Expression
}

type propertyDeclarationNode struct {
	nodeCore
	modifiers    []ModifierLike
	name         PropertyName
	postfixToken NamedMemberBasePostfixToken
	typeNode     TypeNode
	initializer  Expression
}

func (*propertyDeclarationNode) isClassElementBase()              {}
func (*propertyDeclarationNode) isDeclarationBase()               {}
func (*propertyDeclarationNode) isModifiersBase()                 {}
func (*propertyDeclarationNode) isNamedMemberBase()               {}
func (*propertyDeclarationNode) isNodeBase()                      {}
func (*propertyDeclarationNode) isPropertyDeclaration()           {}
func (*propertyDeclarationNode) isVariableOrPropertyDeclaration() {}

func (n *propertyDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *propertyDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *propertyDeclarationNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *propertyDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *propertyDeclarationNode) Initializer() Expression {
	return n.initializer
}

type MethodSignatureDeclaration interface {
	NodeBase
	NamedMemberBase
	FunctionLikeBase
	TypeElementBase
	isMethodSignatureDeclaration()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type methodSignatureDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	name           PropertyName
	postfixToken   NamedMemberBasePostfixToken
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*methodSignatureDeclarationNode) isDeclarationBase()            {}
func (*methodSignatureDeclarationNode) isFunctionLikeBase()           {}
func (*methodSignatureDeclarationNode) isModifiersBase()              {}
func (*methodSignatureDeclarationNode) isNamedMemberBase()            {}
func (*methodSignatureDeclarationNode) isNodeBase()                   {}
func (*methodSignatureDeclarationNode) isTypeElementBase()            {}
func (*methodSignatureDeclarationNode) isMethodSignatureDeclaration() {}
func (*methodSignatureDeclarationNode) isSignatureDeclaration()       {}

func (n *methodSignatureDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *methodSignatureDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *methodSignatureDeclarationNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *methodSignatureDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *methodSignatureDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *methodSignatureDeclarationNode) Type() TypeNode {
	return n.typeNode
}

type MethodDeclaration interface {
	NodeBase
	NamedMemberBase
	FunctionLikeWithBodyBase
	ClassElementBase
	ObjectLiteralElementBase
	isMethodDeclaration()
	isFunctionLikeDeclaration()
	isObjectLiteralElementLike()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	AsteriskToken() AsteriskToken
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type methodDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	asteriskToken  AsteriskToken
	name           PropertyName
	postfixToken   NamedMemberBasePostfixToken
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*methodDeclarationNode) isBodyBase()                 {}
func (*methodDeclarationNode) isClassElementBase()         {}
func (*methodDeclarationNode) isDeclarationBase()          {}
func (*methodDeclarationNode) isFunctionLikeBase()         {}
func (*methodDeclarationNode) isFunctionLikeWithBodyBase() {}
func (*methodDeclarationNode) isModifiersBase()            {}
func (*methodDeclarationNode) isNamedMemberBase()          {}
func (*methodDeclarationNode) isNodeBase()                 {}
func (*methodDeclarationNode) isObjectLiteralElementBase() {}
func (*methodDeclarationNode) isMethodDeclaration()        {}
func (*methodDeclarationNode) isFunctionLikeDeclaration()  {}
func (*methodDeclarationNode) isObjectLiteralElementLike() {}
func (*methodDeclarationNode) isSignatureDeclaration()     {}

func (n *methodDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *methodDeclarationNode) AsteriskToken() AsteriskToken {
	return n.asteriskToken
}

func (n *methodDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *methodDeclarationNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *methodDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *methodDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *methodDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *methodDeclarationNode) Body() FunctionBody {
	return n.body
}

type ClassStaticBlockDeclaration interface {
	NodeBase
	DeclarationBase
	ModifiersBase
	ClassElementBase
	isClassStaticBlockDeclaration()
	Modifiers() []ModifierLike
	Body() Block
}

type classStaticBlockDeclarationNode struct {
	nodeCore
	modifiers []ModifierLike
	body      Block
}

func (*classStaticBlockDeclarationNode) isClassElementBase()            {}
func (*classStaticBlockDeclarationNode) isDeclarationBase()             {}
func (*classStaticBlockDeclarationNode) isModifiersBase()               {}
func (*classStaticBlockDeclarationNode) isNodeBase()                    {}
func (*classStaticBlockDeclarationNode) isClassStaticBlockDeclaration() {}

func (n *classStaticBlockDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *classStaticBlockDeclarationNode) Body() Block {
	return n.body
}

type ConstructorDeclaration interface {
	NodeBase
	DeclarationBase
	ModifiersBase
	FunctionLikeWithBodyBase
	ClassElementBase
	isConstructorDeclaration()
	isFunctionLikeDeclaration()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type constructorDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*constructorDeclarationNode) isBodyBase()                 {}
func (*constructorDeclarationNode) isClassElementBase()         {}
func (*constructorDeclarationNode) isDeclarationBase()          {}
func (*constructorDeclarationNode) isFunctionLikeBase()         {}
func (*constructorDeclarationNode) isFunctionLikeWithBodyBase() {}
func (*constructorDeclarationNode) isModifiersBase()            {}
func (*constructorDeclarationNode) isNodeBase()                 {}
func (*constructorDeclarationNode) isConstructorDeclaration()   {}
func (*constructorDeclarationNode) isFunctionLikeDeclaration()  {}
func (*constructorDeclarationNode) isSignatureDeclaration()     {}

func (n *constructorDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *constructorDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *constructorDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *constructorDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *constructorDeclarationNode) Body() FunctionBody {
	return n.body
}

type GetAccessorDeclaration interface {
	NamedMemberBase
	FunctionLikeWithBodyBase
	TypeElementBase
	ClassElementBase
	ObjectLiteralElementBase
	NodeBase
	isGetAccessorDeclaration()
	isAccessorDeclaration()
	isFunctionLikeDeclaration()
	isObjectLiteralElementLike()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	Name() PropertyName
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type getAccessorDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	name           PropertyName
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*getAccessorDeclarationNode) isBodyBase()                 {}
func (*getAccessorDeclarationNode) isClassElementBase()         {}
func (*getAccessorDeclarationNode) isDeclarationBase()          {}
func (*getAccessorDeclarationNode) isFunctionLikeBase()         {}
func (*getAccessorDeclarationNode) isFunctionLikeWithBodyBase() {}
func (*getAccessorDeclarationNode) isModifiersBase()            {}
func (*getAccessorDeclarationNode) isNamedMemberBase()          {}
func (*getAccessorDeclarationNode) isNodeBase()                 {}
func (*getAccessorDeclarationNode) isObjectLiteralElementBase() {}
func (*getAccessorDeclarationNode) isTypeElementBase()          {}
func (*getAccessorDeclarationNode) isGetAccessorDeclaration()   {}
func (*getAccessorDeclarationNode) isAccessorDeclaration()      {}
func (*getAccessorDeclarationNode) isFunctionLikeDeclaration()  {}
func (*getAccessorDeclarationNode) isObjectLiteralElementLike() {}
func (*getAccessorDeclarationNode) isSignatureDeclaration()     {}

func (n *getAccessorDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *getAccessorDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *getAccessorDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *getAccessorDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *getAccessorDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *getAccessorDeclarationNode) Body() FunctionBody {
	return n.body
}

type SetAccessorDeclaration interface {
	NamedMemberBase
	FunctionLikeWithBodyBase
	TypeElementBase
	ClassElementBase
	ObjectLiteralElementBase
	NodeBase
	isSetAccessorDeclaration()
	isAccessorDeclaration()
	isFunctionLikeDeclaration()
	isObjectLiteralElementLike()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	Name() PropertyName
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type setAccessorDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	name           PropertyName
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*setAccessorDeclarationNode) isBodyBase()                 {}
func (*setAccessorDeclarationNode) isClassElementBase()         {}
func (*setAccessorDeclarationNode) isDeclarationBase()          {}
func (*setAccessorDeclarationNode) isFunctionLikeBase()         {}
func (*setAccessorDeclarationNode) isFunctionLikeWithBodyBase() {}
func (*setAccessorDeclarationNode) isModifiersBase()            {}
func (*setAccessorDeclarationNode) isNamedMemberBase()          {}
func (*setAccessorDeclarationNode) isNodeBase()                 {}
func (*setAccessorDeclarationNode) isObjectLiteralElementBase() {}
func (*setAccessorDeclarationNode) isTypeElementBase()          {}
func (*setAccessorDeclarationNode) isSetAccessorDeclaration()   {}
func (*setAccessorDeclarationNode) isAccessorDeclaration()      {}
func (*setAccessorDeclarationNode) isFunctionLikeDeclaration()  {}
func (*setAccessorDeclarationNode) isObjectLiteralElementLike() {}
func (*setAccessorDeclarationNode) isSignatureDeclaration()     {}

func (n *setAccessorDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *setAccessorDeclarationNode) Name() PropertyName {
	return n.name
}

func (n *setAccessorDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *setAccessorDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *setAccessorDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *setAccessorDeclarationNode) Body() FunctionBody {
	return n.body
}

type CallSignatureDeclaration interface {
	NodeBase
	DeclarationBase
	FunctionLikeBase
	TypeElementBase
	isCallSignatureDeclaration()
	isSignatureDeclaration()
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type callSignatureDeclarationNode struct {
	nodeCore
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*callSignatureDeclarationNode) isDeclarationBase()          {}
func (*callSignatureDeclarationNode) isFunctionLikeBase()         {}
func (*callSignatureDeclarationNode) isNodeBase()                 {}
func (*callSignatureDeclarationNode) isTypeElementBase()          {}
func (*callSignatureDeclarationNode) isCallSignatureDeclaration() {}
func (*callSignatureDeclarationNode) isSignatureDeclaration()     {}

func (n *callSignatureDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *callSignatureDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *callSignatureDeclarationNode) Type() TypeNode {
	return n.typeNode
}

type ConstructSignatureDeclaration interface {
	NodeBase
	DeclarationBase
	FunctionLikeBase
	TypeElementBase
	isConstructSignatureDeclaration()
	isSignatureDeclaration()
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type constructSignatureDeclarationNode struct {
	nodeCore
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*constructSignatureDeclarationNode) isDeclarationBase()               {}
func (*constructSignatureDeclarationNode) isFunctionLikeBase()              {}
func (*constructSignatureDeclarationNode) isNodeBase()                      {}
func (*constructSignatureDeclarationNode) isTypeElementBase()               {}
func (*constructSignatureDeclarationNode) isConstructSignatureDeclaration() {}
func (*constructSignatureDeclarationNode) isSignatureDeclaration()          {}

func (n *constructSignatureDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *constructSignatureDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *constructSignatureDeclarationNode) Type() TypeNode {
	return n.typeNode
}

type IndexSignatureDeclaration interface {
	NodeBase
	DeclarationBase
	ModifiersBase
	FunctionLikeBase
	TypeElementBase
	ClassElementBase
	isIndexSignatureDeclaration()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type indexSignatureDeclarationNode struct {
	nodeCore
	modifiers  []ModifierLike
	parameters []ParameterDeclaration
	typeNode   TypeNode
}

func (*indexSignatureDeclarationNode) isClassElementBase()          {}
func (*indexSignatureDeclarationNode) isDeclarationBase()           {}
func (*indexSignatureDeclarationNode) isFunctionLikeBase()          {}
func (*indexSignatureDeclarationNode) isModifiersBase()             {}
func (*indexSignatureDeclarationNode) isNodeBase()                  {}
func (*indexSignatureDeclarationNode) isTypeElementBase()           {}
func (*indexSignatureDeclarationNode) isIndexSignatureDeclaration() {}
func (*indexSignatureDeclarationNode) isSignatureDeclaration()      {}

func (n *indexSignatureDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *indexSignatureDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *indexSignatureDeclarationNode) Type() TypeNode {
	return n.typeNode
}

type TypePredicateNode interface {
	TypeNodeBase
	isTypePredicateNode()
	AssertsModifier() AssertsKeyword
	ParameterName() TypePredicateParameterName
	Type() TypeNode
}

type typePredicateNodeNode struct {
	nodeCore
	assertsModifier AssertsKeyword
	parameterName   TypePredicateParameterName
	typeNode        TypeNode
}

func (*typePredicateNodeNode) isNodeBase()          {}
func (*typePredicateNodeNode) isTypeNodeBase()      {}
func (*typePredicateNodeNode) isTypePredicateNode() {}

func (n *typePredicateNodeNode) AssertsModifier() AssertsKeyword {
	return n.assertsModifier
}

func (n *typePredicateNodeNode) ParameterName() TypePredicateParameterName {
	return n.parameterName
}

func (n *typePredicateNodeNode) Type() TypeNode {
	return n.typeNode
}

type TypeReferenceNode interface {
	NodeWithTypeArgumentsBase
	isTypeReferenceNode()
	TypeName() EntityName
	TypeArguments() []TypeNode
}

type typeReferenceNodeNode struct {
	nodeCore
	typeName      EntityName
	typeArguments []TypeNode
}

func (*typeReferenceNodeNode) isNodeBase()                  {}
func (*typeReferenceNodeNode) isNodeWithTypeArgumentsBase() {}
func (*typeReferenceNodeNode) isTypeNodeBase()              {}
func (*typeReferenceNodeNode) isTypeReferenceNode()         {}

func (n *typeReferenceNodeNode) TypeName() EntityName {
	return n.typeName
}

func (n *typeReferenceNodeNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

type FunctionTypeNode interface {
	TypeNodeBase
	ModifiersBase
	FunctionLikeBase
	isFunctionTypeNode()
	isSignatureDeclaration()
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type functionTypeNodeNode struct {
	nodeCore
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*functionTypeNodeNode) isDeclarationBase()      {}
func (*functionTypeNodeNode) isFunctionLikeBase()     {}
func (*functionTypeNodeNode) isModifiersBase()        {}
func (*functionTypeNodeNode) isNodeBase()             {}
func (*functionTypeNodeNode) isTypeNodeBase()         {}
func (*functionTypeNodeNode) isFunctionTypeNode()     {}
func (*functionTypeNodeNode) isSignatureDeclaration() {}

func (n *functionTypeNodeNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *functionTypeNodeNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *functionTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

type ConstructorTypeNode interface {
	TypeNodeBase
	ModifiersBase
	FunctionLikeBase
	isConstructorTypeNode()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type constructorTypeNodeNode struct {
	nodeCore
	modifiers      []ModifierLike
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*constructorTypeNodeNode) isDeclarationBase()      {}
func (*constructorTypeNodeNode) isFunctionLikeBase()     {}
func (*constructorTypeNodeNode) isModifiersBase()        {}
func (*constructorTypeNodeNode) isNodeBase()             {}
func (*constructorTypeNodeNode) isTypeNodeBase()         {}
func (*constructorTypeNodeNode) isConstructorTypeNode()  {}
func (*constructorTypeNodeNode) isSignatureDeclaration() {}

func (n *constructorTypeNodeNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *constructorTypeNodeNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *constructorTypeNodeNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *constructorTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

type TypeQueryNode interface {
	NodeWithTypeArgumentsBase
	isTypeQueryNode()
	ExprName() EntityName
	TypeArguments() []TypeNode
}

type typeQueryNodeNode struct {
	nodeCore
	exprName      EntityName
	typeArguments []TypeNode
}

func (*typeQueryNodeNode) isNodeBase()                  {}
func (*typeQueryNodeNode) isNodeWithTypeArgumentsBase() {}
func (*typeQueryNodeNode) isTypeNodeBase()              {}
func (*typeQueryNodeNode) isTypeQueryNode()             {}

func (n *typeQueryNodeNode) ExprName() EntityName {
	return n.exprName
}

func (n *typeQueryNodeNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

type TypeLiteralNode interface {
	TypeNodeBase
	DeclarationBase
	isTypeLiteralNode()
	isObjectTypeDeclaration()
	Members() []TypeElement
}

type typeLiteralNodeNode struct {
	nodeCore
	members []TypeElement
}

func (*typeLiteralNodeNode) isDeclarationBase()       {}
func (*typeLiteralNodeNode) isNodeBase()              {}
func (*typeLiteralNodeNode) isTypeNodeBase()          {}
func (*typeLiteralNodeNode) isTypeLiteralNode()       {}
func (*typeLiteralNodeNode) isObjectTypeDeclaration() {}

func (n *typeLiteralNodeNode) Members() []TypeElement {
	return cloneSlice(n.members)
}

type ArrayTypeNode interface {
	TypeNodeBase
	isArrayTypeNode()
	ElementType() TypeNode
}

type arrayTypeNodeNode struct {
	nodeCore
	elementType TypeNode
}

func (*arrayTypeNodeNode) isNodeBase()      {}
func (*arrayTypeNodeNode) isTypeNodeBase()  {}
func (*arrayTypeNodeNode) isArrayTypeNode() {}

func (n *arrayTypeNodeNode) ElementType() TypeNode {
	return n.elementType
}

type TupleTypeNode interface {
	TypeNodeBase
	isTupleTypeNode()
	Elements() []TypeNode
}

type tupleTypeNodeNode struct {
	nodeCore
	elements []TypeNode
}

func (*tupleTypeNodeNode) isNodeBase()      {}
func (*tupleTypeNodeNode) isTypeNodeBase()  {}
func (*tupleTypeNodeNode) isTupleTypeNode() {}

func (n *tupleTypeNodeNode) Elements() []TypeNode {
	return cloneSlice(n.elements)
}

type OptionalTypeNode interface {
	TypeNodeBase
	isOptionalTypeNode()
	Type() TypeNode
}

type optionalTypeNodeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*optionalTypeNodeNode) isNodeBase()         {}
func (*optionalTypeNodeNode) isTypeNodeBase()     {}
func (*optionalTypeNodeNode) isOptionalTypeNode() {}

func (n *optionalTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

type RestTypeNode interface {
	TypeNodeBase
	isRestTypeNode()
	Type() TypeNode
}

type restTypeNodeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*restTypeNodeNode) isNodeBase()     {}
func (*restTypeNodeNode) isTypeNodeBase() {}
func (*restTypeNodeNode) isRestTypeNode() {}

func (n *restTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

type UnionTypeNode interface {
	TypeNodeBase
	UnionOrIntersectionTypeNodeBase
	isUnionTypeNode()
	isUnionOrIntersectionTypeNode()
	Types() []TypeNode
}

type unionTypeNodeNode struct {
	nodeCore
	types []TypeNode
}

func (*unionTypeNodeNode) isNodeBase()                        {}
func (*unionTypeNodeNode) isTypeNodeBase()                    {}
func (*unionTypeNodeNode) isUnionOrIntersectionTypeNodeBase() {}
func (*unionTypeNodeNode) isUnionTypeNode()                   {}
func (*unionTypeNodeNode) isUnionOrIntersectionTypeNode()     {}

func (n *unionTypeNodeNode) Types() []TypeNode {
	return cloneSlice(n.types)
}

type IntersectionTypeNode interface {
	TypeNodeBase
	UnionOrIntersectionTypeNodeBase
	isIntersectionTypeNode()
	isUnionOrIntersectionTypeNode()
	Types() []TypeNode
}

type intersectionTypeNodeNode struct {
	nodeCore
	types []TypeNode
}

func (*intersectionTypeNodeNode) isNodeBase()                        {}
func (*intersectionTypeNodeNode) isTypeNodeBase()                    {}
func (*intersectionTypeNodeNode) isUnionOrIntersectionTypeNodeBase() {}
func (*intersectionTypeNodeNode) isIntersectionTypeNode()            {}
func (*intersectionTypeNodeNode) isUnionOrIntersectionTypeNode()     {}

func (n *intersectionTypeNodeNode) Types() []TypeNode {
	return cloneSlice(n.types)
}

type ConditionalTypeNode interface {
	TypeNodeBase
	isConditionalTypeNode()
	CheckType() TypeNode
	ExtendsType() TypeNode
	TrueType() TypeNode
	FalseType() TypeNode
}

type conditionalTypeNodeNode struct {
	nodeCore
	checkType   TypeNode
	extendsType TypeNode
	trueType    TypeNode
	falseType   TypeNode
}

func (*conditionalTypeNodeNode) isNodeBase()            {}
func (*conditionalTypeNodeNode) isTypeNodeBase()        {}
func (*conditionalTypeNodeNode) isConditionalTypeNode() {}

func (n *conditionalTypeNodeNode) CheckType() TypeNode {
	return n.checkType
}

func (n *conditionalTypeNodeNode) ExtendsType() TypeNode {
	return n.extendsType
}

func (n *conditionalTypeNodeNode) TrueType() TypeNode {
	return n.trueType
}

func (n *conditionalTypeNodeNode) FalseType() TypeNode {
	return n.falseType
}

type InferTypeNode interface {
	TypeNodeBase
	isInferTypeNode()
	TypeParameter() TypeParameterDeclaration
}

type inferTypeNodeNode struct {
	nodeCore
	typeParameter TypeParameterDeclaration
}

func (*inferTypeNodeNode) isNodeBase()      {}
func (*inferTypeNodeNode) isTypeNodeBase()  {}
func (*inferTypeNodeNode) isInferTypeNode() {}

func (n *inferTypeNodeNode) TypeParameter() TypeParameterDeclaration {
	return n.typeParameter
}

type ParenthesizedTypeNode interface {
	TypeNodeBase
	isParenthesizedTypeNode()
	Type() TypeNode
}

type parenthesizedTypeNodeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*parenthesizedTypeNodeNode) isNodeBase()              {}
func (*parenthesizedTypeNodeNode) isTypeNodeBase()          {}
func (*parenthesizedTypeNodeNode) isParenthesizedTypeNode() {}

func (n *parenthesizedTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

type ThisTypeNode interface {
	TypeNodeBase
	isThisTypeNode()
	isTypePredicateParameterName()
}

type thisTypeNodeNode struct {
	nodeCore
}

func (*thisTypeNodeNode) isNodeBase()                   {}
func (*thisTypeNodeNode) isTypeNodeBase()               {}
func (*thisTypeNodeNode) isThisTypeNode()               {}
func (*thisTypeNodeNode) isTypePredicateParameterName() {}

type TypeOperatorNode interface {
	TypeNodeBase
	isTypeOperatorNode()
	Operator() TypeOperatorNodeOperatorKind
	Type() TypeNode
}

type typeOperatorNodeNode struct {
	nodeCore
	operator TypeOperatorNodeOperatorKind
	typeNode TypeNode
}

func (*typeOperatorNodeNode) isNodeBase()         {}
func (*typeOperatorNodeNode) isTypeNodeBase()     {}
func (*typeOperatorNodeNode) isTypeOperatorNode() {}

func (n *typeOperatorNodeNode) Operator() TypeOperatorNodeOperatorKind {
	return n.operator
}

func (n *typeOperatorNodeNode) Type() TypeNode {
	return n.typeNode
}

type IndexedAccessTypeNode interface {
	TypeNodeBase
	isIndexedAccessTypeNode()
	ObjectType() TypeNode
	IndexType() TypeNode
}

type indexedAccessTypeNodeNode struct {
	nodeCore
	objectType TypeNode
	indexType  TypeNode
}

func (*indexedAccessTypeNodeNode) isNodeBase()              {}
func (*indexedAccessTypeNodeNode) isTypeNodeBase()          {}
func (*indexedAccessTypeNodeNode) isIndexedAccessTypeNode() {}

func (n *indexedAccessTypeNodeNode) ObjectType() TypeNode {
	return n.objectType
}

func (n *indexedAccessTypeNodeNode) IndexType() TypeNode {
	return n.indexType
}

type MappedTypeNode interface {
	TypeNodeBase
	DeclarationBase
	isMappedTypeNode()
	ReadonlyToken() MappedTypeNodeReadonlyToken
	TypeParameter() TypeParameterDeclaration
	NameType() TypeNode
	QuestionToken() MappedTypeNodeQuestionToken
	Type() TypeNode
	Members() []TypeElement
}

type mappedTypeNodeNode struct {
	nodeCore
	readonlyToken MappedTypeNodeReadonlyToken
	typeParameter TypeParameterDeclaration
	nameType      TypeNode
	questionToken MappedTypeNodeQuestionToken
	typeNode      TypeNode
	members       []TypeElement
}

func (*mappedTypeNodeNode) isDeclarationBase() {}
func (*mappedTypeNodeNode) isNodeBase()        {}
func (*mappedTypeNodeNode) isTypeNodeBase()    {}
func (*mappedTypeNodeNode) isMappedTypeNode()  {}

func (n *mappedTypeNodeNode) ReadonlyToken() MappedTypeNodeReadonlyToken {
	return n.readonlyToken
}

func (n *mappedTypeNodeNode) TypeParameter() TypeParameterDeclaration {
	return n.typeParameter
}

func (n *mappedTypeNodeNode) NameType() TypeNode {
	return n.nameType
}

func (n *mappedTypeNodeNode) QuestionToken() MappedTypeNodeQuestionToken {
	return n.questionToken
}

func (n *mappedTypeNodeNode) Type() TypeNode {
	return n.typeNode
}

func (n *mappedTypeNodeNode) Members() []TypeElement {
	return cloneSlice(n.members)
}

type LiteralTypeNode interface {
	TypeNodeBase
	isLiteralTypeNode()
	Literal() Node
}

type literalTypeNodeNode struct {
	nodeCore
	literal Node
}

func (*literalTypeNodeNode) isNodeBase()        {}
func (*literalTypeNodeNode) isTypeNodeBase()    {}
func (*literalTypeNodeNode) isLiteralTypeNode() {}

func (n *literalTypeNodeNode) Literal() Node {
	return n.literal
}

type NamedTupleMember interface {
	TypeNodeBase
	DeclarationBase
	isNamedTupleMember()
	DotDotDotToken() DotDotDotToken
	Name() Identifier
	QuestionToken() QuestionToken
	Type() TypeNode
}

type namedTupleMemberNode struct {
	nodeCore
	dotDotDotToken DotDotDotToken
	name           Identifier
	questionToken  QuestionToken
	typeNode       TypeNode
}

func (*namedTupleMemberNode) isDeclarationBase()  {}
func (*namedTupleMemberNode) isNodeBase()         {}
func (*namedTupleMemberNode) isTypeNodeBase()     {}
func (*namedTupleMemberNode) isNamedTupleMember() {}

func (n *namedTupleMemberNode) DotDotDotToken() DotDotDotToken {
	return n.dotDotDotToken
}

func (n *namedTupleMemberNode) Name() Identifier {
	return n.name
}

func (n *namedTupleMemberNode) QuestionToken() QuestionToken {
	return n.questionToken
}

func (n *namedTupleMemberNode) Type() TypeNode {
	return n.typeNode
}

type TemplateLiteralTypeNode interface {
	TypeNodeBase
	isTemplateLiteralTypeNode()
	Head() TemplateHead
	TemplateSpans() []TemplateLiteralTypeSpan
}

type templateLiteralTypeNodeNode struct {
	nodeCore
	head          TemplateHead
	templateSpans []TemplateLiteralTypeSpan
}

func (*templateLiteralTypeNodeNode) isNodeBase()                {}
func (*templateLiteralTypeNodeNode) isTypeNodeBase()            {}
func (*templateLiteralTypeNodeNode) isTemplateLiteralTypeNode() {}

func (n *templateLiteralTypeNodeNode) Head() TemplateHead {
	return n.head
}

func (n *templateLiteralTypeNodeNode) TemplateSpans() []TemplateLiteralTypeSpan {
	return cloneSlice(n.templateSpans)
}

type TemplateLiteralTypeSpan interface {
	TypeNodeBase
	isTemplateLiteralTypeSpan()
	Type() TypeNode
	Literal() TemplateMiddleOrTail
}

type templateLiteralTypeSpanNode struct {
	nodeCore
	typeNode TypeNode
	literal  TemplateMiddleOrTail
}

func (*templateLiteralTypeSpanNode) isNodeBase()                {}
func (*templateLiteralTypeSpanNode) isTypeNodeBase()            {}
func (*templateLiteralTypeSpanNode) isTemplateLiteralTypeSpan() {}

func (n *templateLiteralTypeSpanNode) Type() TypeNode {
	return n.typeNode
}

func (n *templateLiteralTypeSpanNode) Literal() TemplateMiddleOrTail {
	return n.literal
}

type ImportTypeNode interface {
	NodeWithTypeArgumentsBase
	isImportTypeNode()
	IsTypeOf() bool
	Argument() TypeNode
	Attributes() ImportAttributes
	Qualifier() EntityName
	TypeArguments() []TypeNode
}

type importTypeNodeNode struct {
	nodeCore
	isTypeOf      bool
	argument      TypeNode
	attributes    ImportAttributes
	qualifier     EntityName
	typeArguments []TypeNode
}

func (*importTypeNodeNode) isNodeBase()                  {}
func (*importTypeNodeNode) isNodeWithTypeArgumentsBase() {}
func (*importTypeNodeNode) isTypeNodeBase()              {}
func (*importTypeNodeNode) isImportTypeNode()            {}

func (n *importTypeNodeNode) IsTypeOf() bool {
	return n.isTypeOf
}

func (n *importTypeNodeNode) Argument() TypeNode {
	return n.argument
}

func (n *importTypeNodeNode) Attributes() ImportAttributes {
	return n.attributes
}

func (n *importTypeNodeNode) Qualifier() EntityName {
	return n.qualifier
}

func (n *importTypeNodeNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

type ObjectBindingPattern interface {
	BindingPattern
	isObjectBindingPattern()
	isBindingName()
	isDeclarationName()
	isImportClauseOrBindingPattern()
	isObjectLiteralLikeNode()
	Elements() []BindingElement
}

type objectBindingPatternNode struct {
	nodeCore
	elements []BindingElement
}

func (*objectBindingPatternNode) isNodeBase()                     {}
func (*objectBindingPatternNode) isBindingPattern()               {}
func (*objectBindingPatternNode) isObjectBindingPattern()         {}
func (*objectBindingPatternNode) isBindingName()                  {}
func (*objectBindingPatternNode) isDeclarationName()              {}
func (*objectBindingPatternNode) isImportClauseOrBindingPattern() {}
func (*objectBindingPatternNode) isObjectLiteralLikeNode()        {}

func (n *objectBindingPatternNode) Elements() []BindingElement {
	return cloneSlice(n.elements)
}

type ArrayBindingPattern interface {
	BindingPattern
	isArrayBindingPattern()
	isBindingName()
	isDeclarationName()
	isImportClauseOrBindingPattern()
	Elements() []BindingElement
}

type arrayBindingPatternNode struct {
	nodeCore
	elements []BindingElement
}

func (*arrayBindingPatternNode) isNodeBase()                     {}
func (*arrayBindingPatternNode) isBindingPattern()               {}
func (*arrayBindingPatternNode) isArrayBindingPattern()          {}
func (*arrayBindingPatternNode) isBindingName()                  {}
func (*arrayBindingPatternNode) isDeclarationName()              {}
func (*arrayBindingPatternNode) isImportClauseOrBindingPattern() {}

func (n *arrayBindingPatternNode) Elements() []BindingElement {
	return cloneSlice(n.elements)
}

type BindingElement interface {
	NodeBase
	DeclarationBase
	isBindingElement()
	isArrayBindingElement()
	DotDotDotToken() DotDotDotToken
	PropertyName() PropertyName
	Name() BindingName
	Initializer() Expression
}

type bindingElementNode struct {
	nodeCore
	dotDotDotToken DotDotDotToken
	propertyName   PropertyName
	name           BindingName
	initializer    Expression
}

func (*bindingElementNode) isDeclarationBase()     {}
func (*bindingElementNode) isNodeBase()            {}
func (*bindingElementNode) isBindingElement()      {}
func (*bindingElementNode) isArrayBindingElement() {}

func (n *bindingElementNode) DotDotDotToken() DotDotDotToken {
	return n.dotDotDotToken
}

func (n *bindingElementNode) PropertyName() PropertyName {
	return n.propertyName
}

func (n *bindingElementNode) Name() BindingName {
	return n.name
}

func (n *bindingElementNode) Initializer() Expression {
	return n.initializer
}

type ArrayLiteralExpression interface {
	PrimaryExpressionBase
	isArrayLiteralExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Elements() []Expression
	MultiLine() bool
}

type arrayLiteralExpressionNode struct {
	nodeCore
	elements  []Expression
	multiLine bool
}

func (*arrayLiteralExpressionNode) isExpressionBase()             {}
func (*arrayLiteralExpressionNode) isLeftHandSideExpressionBase() {}
func (*arrayLiteralExpressionNode) isMemberExpressionBase()       {}
func (*arrayLiteralExpressionNode) isNodeBase()                   {}
func (*arrayLiteralExpressionNode) isPrimaryExpressionBase()      {}
func (*arrayLiteralExpressionNode) isUnaryExpressionBase()        {}
func (*arrayLiteralExpressionNode) isUpdateExpressionBase()       {}
func (*arrayLiteralExpressionNode) isArrayLiteralExpression()     {}
func (*arrayLiteralExpressionNode) isBlockOrExpression()          {}
func (*arrayLiteralExpressionNode) isConciseBody()                {}
func (*arrayLiteralExpressionNode) isForInitializer()             {}
func (*arrayLiteralExpressionNode) isIncrementExpression()        {}
func (*arrayLiteralExpressionNode) isNodeBody()                   {}

func (n *arrayLiteralExpressionNode) Elements() []Expression {
	return cloneSlice(n.elements)
}

func (n *arrayLiteralExpressionNode) MultiLine() bool {
	return n.multiLine
}

type ObjectLiteralExpression interface {
	PrimaryExpressionBase
	DeclarationBase
	isObjectLiteralExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	isObjectLiteralLikeNode()
	Properties() []ObjectLiteralElementLike
	MultiLine() bool
}

type objectLiteralExpressionNode struct {
	nodeCore
	properties []ObjectLiteralElementLike
	multiLine  bool
}

func (*objectLiteralExpressionNode) isDeclarationBase()            {}
func (*objectLiteralExpressionNode) isExpressionBase()             {}
func (*objectLiteralExpressionNode) isLeftHandSideExpressionBase() {}
func (*objectLiteralExpressionNode) isMemberExpressionBase()       {}
func (*objectLiteralExpressionNode) isNodeBase()                   {}
func (*objectLiteralExpressionNode) isPrimaryExpressionBase()      {}
func (*objectLiteralExpressionNode) isUnaryExpressionBase()        {}
func (*objectLiteralExpressionNode) isUpdateExpressionBase()       {}
func (*objectLiteralExpressionNode) isObjectLiteralExpression()    {}
func (*objectLiteralExpressionNode) isBlockOrExpression()          {}
func (*objectLiteralExpressionNode) isConciseBody()                {}
func (*objectLiteralExpressionNode) isForInitializer()             {}
func (*objectLiteralExpressionNode) isIncrementExpression()        {}
func (*objectLiteralExpressionNode) isNodeBody()                   {}
func (*objectLiteralExpressionNode) isObjectLiteralLikeNode()      {}

func (n *objectLiteralExpressionNode) Properties() []ObjectLiteralElementLike {
	return cloneSlice(n.properties)
}

func (n *objectLiteralExpressionNode) MultiLine() bool {
	return n.multiLine
}

type PropertyAccessExpression interface {
	MemberExpressionBase
	isPropertyAccessExpression()
	isAccessExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isJsxTagNameExpression()
	isNodeBody()
	Expression() Expression
	QuestionDotToken() QuestionDotToken
	Name() MemberName
}

type propertyAccessExpressionNode struct {
	nodeCore
	expression       Expression
	questionDotToken QuestionDotToken
	name             MemberName
}

func (*propertyAccessExpressionNode) isExpressionBase()             {}
func (*propertyAccessExpressionNode) isLeftHandSideExpressionBase() {}
func (*propertyAccessExpressionNode) isMemberExpressionBase()       {}
func (*propertyAccessExpressionNode) isNodeBase()                   {}
func (*propertyAccessExpressionNode) isUnaryExpressionBase()        {}
func (*propertyAccessExpressionNode) isUpdateExpressionBase()       {}
func (*propertyAccessExpressionNode) isPropertyAccessExpression()   {}
func (*propertyAccessExpressionNode) isAccessExpression()           {}
func (*propertyAccessExpressionNode) isBlockOrExpression()          {}
func (*propertyAccessExpressionNode) isConciseBody()                {}
func (*propertyAccessExpressionNode) isForInitializer()             {}
func (*propertyAccessExpressionNode) isIncrementExpression()        {}
func (*propertyAccessExpressionNode) isJsxTagNameExpression()       {}
func (*propertyAccessExpressionNode) isNodeBody()                   {}

func (n *propertyAccessExpressionNode) Expression() Expression {
	return n.expression
}

func (n *propertyAccessExpressionNode) QuestionDotToken() QuestionDotToken {
	return n.questionDotToken
}

func (n *propertyAccessExpressionNode) Name() MemberName {
	return n.name
}

type ElementAccessExpression interface {
	MemberExpressionBase
	isElementAccessExpression()
	isAccessExpression()
	isBlockOrExpression()
	isConciseBody()
	isDeclarationName()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
	QuestionDotToken() QuestionDotToken
	ArgumentExpression() Expression
}

type elementAccessExpressionNode struct {
	nodeCore
	expression         Expression
	questionDotToken   QuestionDotToken
	argumentExpression Expression
}

func (*elementAccessExpressionNode) isExpressionBase()             {}
func (*elementAccessExpressionNode) isLeftHandSideExpressionBase() {}
func (*elementAccessExpressionNode) isMemberExpressionBase()       {}
func (*elementAccessExpressionNode) isNodeBase()                   {}
func (*elementAccessExpressionNode) isUnaryExpressionBase()        {}
func (*elementAccessExpressionNode) isUpdateExpressionBase()       {}
func (*elementAccessExpressionNode) isElementAccessExpression()    {}
func (*elementAccessExpressionNode) isAccessExpression()           {}
func (*elementAccessExpressionNode) isBlockOrExpression()          {}
func (*elementAccessExpressionNode) isConciseBody()                {}
func (*elementAccessExpressionNode) isDeclarationName()            {}
func (*elementAccessExpressionNode) isForInitializer()             {}
func (*elementAccessExpressionNode) isIncrementExpression()        {}
func (*elementAccessExpressionNode) isNodeBody()                   {}

func (n *elementAccessExpressionNode) Expression() Expression {
	return n.expression
}

func (n *elementAccessExpressionNode) QuestionDotToken() QuestionDotToken {
	return n.questionDotToken
}

func (n *elementAccessExpressionNode) ArgumentExpression() Expression {
	return n.argumentExpression
}

type CallExpression interface {
	LeftHandSideExpressionBase
	DeclarationBase
	isCallExpression()
	isBlockOrExpression()
	isCallLikeExpression()
	isCallOrNewExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
	QuestionDotToken() QuestionDotToken
	TypeArguments() []TypeNode
	Arguments() []Expression
}

type callExpressionNode struct {
	nodeCore
	expression       Expression
	questionDotToken QuestionDotToken
	typeArguments    []TypeNode
	arguments        []Expression
}

func (*callExpressionNode) isDeclarationBase()            {}
func (*callExpressionNode) isExpressionBase()             {}
func (*callExpressionNode) isLeftHandSideExpressionBase() {}
func (*callExpressionNode) isNodeBase()                   {}
func (*callExpressionNode) isUnaryExpressionBase()        {}
func (*callExpressionNode) isUpdateExpressionBase()       {}
func (*callExpressionNode) isCallExpression()             {}
func (*callExpressionNode) isBlockOrExpression()          {}
func (*callExpressionNode) isCallLikeExpression()         {}
func (*callExpressionNode) isCallOrNewExpression()        {}
func (*callExpressionNode) isConciseBody()                {}
func (*callExpressionNode) isForInitializer()             {}
func (*callExpressionNode) isIncrementExpression()        {}
func (*callExpressionNode) isNodeBody()                   {}

func (n *callExpressionNode) Expression() Expression {
	return n.expression
}

func (n *callExpressionNode) QuestionDotToken() QuestionDotToken {
	return n.questionDotToken
}

func (n *callExpressionNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

func (n *callExpressionNode) Arguments() []Expression {
	return cloneSlice(n.arguments)
}

type NewExpression interface {
	PrimaryExpressionBase
	isNewExpression()
	isBlockOrExpression()
	isCallLikeExpression()
	isCallOrNewExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
	TypeArguments() []TypeNode
	Arguments() []Expression
}

type newExpressionNode struct {
	nodeCore
	expression    Expression
	typeArguments []TypeNode
	arguments     []Expression
}

func (*newExpressionNode) isExpressionBase()             {}
func (*newExpressionNode) isLeftHandSideExpressionBase() {}
func (*newExpressionNode) isMemberExpressionBase()       {}
func (*newExpressionNode) isNodeBase()                   {}
func (*newExpressionNode) isPrimaryExpressionBase()      {}
func (*newExpressionNode) isUnaryExpressionBase()        {}
func (*newExpressionNode) isUpdateExpressionBase()       {}
func (*newExpressionNode) isNewExpression()              {}
func (*newExpressionNode) isBlockOrExpression()          {}
func (*newExpressionNode) isCallLikeExpression()         {}
func (*newExpressionNode) isCallOrNewExpression()        {}
func (*newExpressionNode) isConciseBody()                {}
func (*newExpressionNode) isForInitializer()             {}
func (*newExpressionNode) isIncrementExpression()        {}
func (*newExpressionNode) isNodeBody()                   {}

func (n *newExpressionNode) Expression() Expression {
	return n.expression
}

func (n *newExpressionNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

func (n *newExpressionNode) Arguments() []Expression {
	return cloneSlice(n.arguments)
}

type TaggedTemplateExpression interface {
	MemberExpressionBase
	isTaggedTemplateExpression()
	isBlockOrExpression()
	isCallLikeExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Tag() Expression
	QuestionDotToken() QuestionDotToken
	TypeArguments() []TypeNode
	Template() TemplateLiteral
}

type taggedTemplateExpressionNode struct {
	nodeCore
	tag              Expression
	questionDotToken QuestionDotToken
	typeArguments    []TypeNode
	template         TemplateLiteral
}

func (*taggedTemplateExpressionNode) isExpressionBase()             {}
func (*taggedTemplateExpressionNode) isLeftHandSideExpressionBase() {}
func (*taggedTemplateExpressionNode) isMemberExpressionBase()       {}
func (*taggedTemplateExpressionNode) isNodeBase()                   {}
func (*taggedTemplateExpressionNode) isUnaryExpressionBase()        {}
func (*taggedTemplateExpressionNode) isUpdateExpressionBase()       {}
func (*taggedTemplateExpressionNode) isTaggedTemplateExpression()   {}
func (*taggedTemplateExpressionNode) isBlockOrExpression()          {}
func (*taggedTemplateExpressionNode) isCallLikeExpression()         {}
func (*taggedTemplateExpressionNode) isConciseBody()                {}
func (*taggedTemplateExpressionNode) isForInitializer()             {}
func (*taggedTemplateExpressionNode) isIncrementExpression()        {}
func (*taggedTemplateExpressionNode) isNodeBody()                   {}

func (n *taggedTemplateExpressionNode) Tag() Expression {
	return n.tag
}

func (n *taggedTemplateExpressionNode) QuestionDotToken() QuestionDotToken {
	return n.questionDotToken
}

func (n *taggedTemplateExpressionNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

func (n *taggedTemplateExpressionNode) Template() TemplateLiteral {
	return n.template
}

type TypeAssertion interface {
	UnaryExpressionBase
	isTypeAssertion()
	isAssertionExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Type() TypeNode
	Expression() Expression
}

type typeAssertionNode struct {
	nodeCore
	typeNode   TypeNode
	expression Expression
}

func (*typeAssertionNode) isExpressionBase()      {}
func (*typeAssertionNode) isNodeBase()            {}
func (*typeAssertionNode) isUnaryExpressionBase() {}
func (*typeAssertionNode) isTypeAssertion()       {}
func (*typeAssertionNode) isAssertionExpression() {}
func (*typeAssertionNode) isBlockOrExpression()   {}
func (*typeAssertionNode) isConciseBody()         {}
func (*typeAssertionNode) isForInitializer()      {}
func (*typeAssertionNode) isNodeBody()            {}

func (n *typeAssertionNode) Type() TypeNode {
	return n.typeNode
}

func (n *typeAssertionNode) Expression() Expression {
	return n.expression
}

type ParenthesizedExpression interface {
	PrimaryExpressionBase
	isParenthesizedExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
}

type parenthesizedExpressionNode struct {
	nodeCore
	expression Expression
}

func (*parenthesizedExpressionNode) isExpressionBase()             {}
func (*parenthesizedExpressionNode) isLeftHandSideExpressionBase() {}
func (*parenthesizedExpressionNode) isMemberExpressionBase()       {}
func (*parenthesizedExpressionNode) isNodeBase()                   {}
func (*parenthesizedExpressionNode) isPrimaryExpressionBase()      {}
func (*parenthesizedExpressionNode) isUnaryExpressionBase()        {}
func (*parenthesizedExpressionNode) isUpdateExpressionBase()       {}
func (*parenthesizedExpressionNode) isParenthesizedExpression()    {}
func (*parenthesizedExpressionNode) isBlockOrExpression()          {}
func (*parenthesizedExpressionNode) isConciseBody()                {}
func (*parenthesizedExpressionNode) isForInitializer()             {}
func (*parenthesizedExpressionNode) isIncrementExpression()        {}
func (*parenthesizedExpressionNode) isNodeBody()                   {}

func (n *parenthesizedExpressionNode) Expression() Expression {
	return n.expression
}

type FunctionExpression interface {
	PrimaryExpressionBase
	DeclarationBase
	ModifiersBase
	FunctionLikeWithBodyBase
	isFunctionExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isFunctionLikeDeclaration()
	isIncrementExpression()
	isNodeBody()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	AsteriskToken() AsteriskToken
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type functionExpressionNode struct {
	nodeCore
	modifiers      []ModifierLike
	asteriskToken  AsteriskToken
	name           Identifier
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*functionExpressionNode) isBodyBase()                   {}
func (*functionExpressionNode) isDeclarationBase()            {}
func (*functionExpressionNode) isExpressionBase()             {}
func (*functionExpressionNode) isFunctionLikeBase()           {}
func (*functionExpressionNode) isFunctionLikeWithBodyBase()   {}
func (*functionExpressionNode) isLeftHandSideExpressionBase() {}
func (*functionExpressionNode) isMemberExpressionBase()       {}
func (*functionExpressionNode) isModifiersBase()              {}
func (*functionExpressionNode) isNodeBase()                   {}
func (*functionExpressionNode) isPrimaryExpressionBase()      {}
func (*functionExpressionNode) isUnaryExpressionBase()        {}
func (*functionExpressionNode) isUpdateExpressionBase()       {}
func (*functionExpressionNode) isFunctionExpression()         {}
func (*functionExpressionNode) isBlockOrExpression()          {}
func (*functionExpressionNode) isConciseBody()                {}
func (*functionExpressionNode) isForInitializer()             {}
func (*functionExpressionNode) isFunctionLikeDeclaration()    {}
func (*functionExpressionNode) isIncrementExpression()        {}
func (*functionExpressionNode) isNodeBody()                   {}
func (*functionExpressionNode) isSignatureDeclaration()       {}

func (n *functionExpressionNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *functionExpressionNode) AsteriskToken() AsteriskToken {
	return n.asteriskToken
}

func (n *functionExpressionNode) Name() Identifier {
	return n.name
}

func (n *functionExpressionNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *functionExpressionNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *functionExpressionNode) Type() TypeNode {
	return n.typeNode
}

func (n *functionExpressionNode) Body() FunctionBody {
	return n.body
}

type ArrowFunction interface {
	ExpressionBase
	DeclarationBase
	ModifiersBase
	FunctionLikeWithBodyBase
	isArrowFunction()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isFunctionLikeDeclaration()
	isNodeBody()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	EqualsGreaterThanToken() EqualsGreaterThanToken
	Body() ConciseBody
}

type arrowFunctionNode struct {
	nodeCore
	modifiers              []ModifierLike
	typeParameters         []TypeParameterDeclaration
	parameters             []ParameterDeclaration
	typeNode               TypeNode
	equalsGreaterThanToken EqualsGreaterThanToken
	body                   ConciseBody
}

func (*arrowFunctionNode) isBodyBase()                 {}
func (*arrowFunctionNode) isDeclarationBase()          {}
func (*arrowFunctionNode) isExpressionBase()           {}
func (*arrowFunctionNode) isFunctionLikeBase()         {}
func (*arrowFunctionNode) isFunctionLikeWithBodyBase() {}
func (*arrowFunctionNode) isModifiersBase()            {}
func (*arrowFunctionNode) isNodeBase()                 {}
func (*arrowFunctionNode) isArrowFunction()            {}
func (*arrowFunctionNode) isBlockOrExpression()        {}
func (*arrowFunctionNode) isConciseBody()              {}
func (*arrowFunctionNode) isForInitializer()           {}
func (*arrowFunctionNode) isFunctionLikeDeclaration()  {}
func (*arrowFunctionNode) isNodeBody()                 {}
func (*arrowFunctionNode) isSignatureDeclaration()     {}

func (n *arrowFunctionNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *arrowFunctionNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *arrowFunctionNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *arrowFunctionNode) Type() TypeNode {
	return n.typeNode
}

func (n *arrowFunctionNode) EqualsGreaterThanToken() EqualsGreaterThanToken {
	return n.equalsGreaterThanToken
}

func (n *arrowFunctionNode) Body() ConciseBody {
	return n.body
}

type DeleteExpression interface {
	UnaryExpressionBase
	isDeleteExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
}

type deleteExpressionNode struct {
	nodeCore
	expression Expression
}

func (*deleteExpressionNode) isExpressionBase()      {}
func (*deleteExpressionNode) isNodeBase()            {}
func (*deleteExpressionNode) isUnaryExpressionBase() {}
func (*deleteExpressionNode) isDeleteExpression()    {}
func (*deleteExpressionNode) isBlockOrExpression()   {}
func (*deleteExpressionNode) isConciseBody()         {}
func (*deleteExpressionNode) isForInitializer()      {}
func (*deleteExpressionNode) isNodeBody()            {}

func (n *deleteExpressionNode) Expression() Expression {
	return n.expression
}

type TypeOfExpression interface {
	UnaryExpressionBase
	isTypeOfExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
}

type typeOfExpressionNode struct {
	nodeCore
	expression Expression
}

func (*typeOfExpressionNode) isExpressionBase()      {}
func (*typeOfExpressionNode) isNodeBase()            {}
func (*typeOfExpressionNode) isUnaryExpressionBase() {}
func (*typeOfExpressionNode) isTypeOfExpression()    {}
func (*typeOfExpressionNode) isBlockOrExpression()   {}
func (*typeOfExpressionNode) isConciseBody()         {}
func (*typeOfExpressionNode) isForInitializer()      {}
func (*typeOfExpressionNode) isNodeBody()            {}

func (n *typeOfExpressionNode) Expression() Expression {
	return n.expression
}

type VoidExpression interface {
	UnaryExpressionBase
	isVoidExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
}

type voidExpressionNode struct {
	nodeCore
	expression Expression
}

func (*voidExpressionNode) isExpressionBase()      {}
func (*voidExpressionNode) isNodeBase()            {}
func (*voidExpressionNode) isUnaryExpressionBase() {}
func (*voidExpressionNode) isVoidExpression()      {}
func (*voidExpressionNode) isBlockOrExpression()   {}
func (*voidExpressionNode) isConciseBody()         {}
func (*voidExpressionNode) isForInitializer()      {}
func (*voidExpressionNode) isNodeBody()            {}

func (n *voidExpressionNode) Expression() Expression {
	return n.expression
}

type AwaitExpression interface {
	UnaryExpressionBase
	isAwaitExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
}

type awaitExpressionNode struct {
	nodeCore
	expression Expression
}

func (*awaitExpressionNode) isExpressionBase()      {}
func (*awaitExpressionNode) isNodeBase()            {}
func (*awaitExpressionNode) isUnaryExpressionBase() {}
func (*awaitExpressionNode) isAwaitExpression()     {}
func (*awaitExpressionNode) isBlockOrExpression()   {}
func (*awaitExpressionNode) isConciseBody()         {}
func (*awaitExpressionNode) isForInitializer()      {}
func (*awaitExpressionNode) isNodeBody()            {}

func (n *awaitExpressionNode) Expression() Expression {
	return n.expression
}

type PrefixUnaryExpression interface {
	UpdateExpressionBase
	isPrefixUnaryExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Operator() PrefixUnaryExpressionOperatorKind
	Operand() Expression
}

type prefixUnaryExpressionNode struct {
	nodeCore
	operator PrefixUnaryExpressionOperatorKind
	operand  Expression
}

func (*prefixUnaryExpressionNode) isExpressionBase()        {}
func (*prefixUnaryExpressionNode) isNodeBase()              {}
func (*prefixUnaryExpressionNode) isUnaryExpressionBase()   {}
func (*prefixUnaryExpressionNode) isUpdateExpressionBase()  {}
func (*prefixUnaryExpressionNode) isPrefixUnaryExpression() {}
func (*prefixUnaryExpressionNode) isBlockOrExpression()     {}
func (*prefixUnaryExpressionNode) isConciseBody()           {}
func (*prefixUnaryExpressionNode) isForInitializer()        {}
func (*prefixUnaryExpressionNode) isIncrementExpression()   {}
func (*prefixUnaryExpressionNode) isNodeBody()              {}

func (n *prefixUnaryExpressionNode) Operator() PrefixUnaryExpressionOperatorKind {
	return n.operator
}

func (n *prefixUnaryExpressionNode) Operand() Expression {
	return n.operand
}

type PostfixUnaryExpression interface {
	UpdateExpressionBase
	isPostfixUnaryExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Operand() Expression
	Operator() PostfixUnaryExpressionOperatorKind
}

type postfixUnaryExpressionNode struct {
	nodeCore
	operand  Expression
	operator PostfixUnaryExpressionOperatorKind
}

func (*postfixUnaryExpressionNode) isExpressionBase()         {}
func (*postfixUnaryExpressionNode) isNodeBase()               {}
func (*postfixUnaryExpressionNode) isUnaryExpressionBase()    {}
func (*postfixUnaryExpressionNode) isUpdateExpressionBase()   {}
func (*postfixUnaryExpressionNode) isPostfixUnaryExpression() {}
func (*postfixUnaryExpressionNode) isBlockOrExpression()      {}
func (*postfixUnaryExpressionNode) isConciseBody()            {}
func (*postfixUnaryExpressionNode) isForInitializer()         {}
func (*postfixUnaryExpressionNode) isIncrementExpression()    {}
func (*postfixUnaryExpressionNode) isNodeBody()               {}

func (n *postfixUnaryExpressionNode) Operand() Expression {
	return n.operand
}

func (n *postfixUnaryExpressionNode) Operator() PostfixUnaryExpressionOperatorKind {
	return n.operator
}

type BinaryExpression interface {
	ExpressionBase
	DeclarationBase
	ModifiersBase
	isBinaryExpression()
	isArrayDestructuringAssignment()
	isBlockOrExpression()
	isCallLikeExpression()
	isConciseBody()
	isDestructuringAssignment()
	isForInitializer()
	isNodeBody()
	isObjectDestructuringAssignment()
	Modifiers() []ModifierLike
	Left() Expression
	Type() TypeNode
	OperatorToken() BinaryOperatorToken
	Right() Expression
}

type binaryExpressionNode struct {
	nodeCore
	modifiers     []ModifierLike
	left          Expression
	typeNode      TypeNode
	operatorToken BinaryOperatorToken
	right         Expression
}

func (*binaryExpressionNode) isDeclarationBase()               {}
func (*binaryExpressionNode) isExpressionBase()                {}
func (*binaryExpressionNode) isModifiersBase()                 {}
func (*binaryExpressionNode) isNodeBase()                      {}
func (*binaryExpressionNode) isBinaryExpression()              {}
func (*binaryExpressionNode) isArrayDestructuringAssignment()  {}
func (*binaryExpressionNode) isBlockOrExpression()             {}
func (*binaryExpressionNode) isCallLikeExpression()            {}
func (*binaryExpressionNode) isConciseBody()                   {}
func (*binaryExpressionNode) isDestructuringAssignment()       {}
func (*binaryExpressionNode) isForInitializer()                {}
func (*binaryExpressionNode) isNodeBody()                      {}
func (*binaryExpressionNode) isObjectDestructuringAssignment() {}

func (n *binaryExpressionNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *binaryExpressionNode) Left() Expression {
	return n.left
}

func (n *binaryExpressionNode) Type() TypeNode {
	return n.typeNode
}

func (n *binaryExpressionNode) OperatorToken() BinaryOperatorToken {
	return n.operatorToken
}

func (n *binaryExpressionNode) Right() Expression {
	return n.right
}

type ConditionalExpression interface {
	ExpressionBase
	isConditionalExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Condition() Expression
	QuestionToken() QuestionToken
	WhenTrue() Expression
	ColonToken() ColonToken
	WhenFalse() Expression
}

type conditionalExpressionNode struct {
	nodeCore
	condition     Expression
	questionToken QuestionToken
	whenTrue      Expression
	colonToken    ColonToken
	whenFalse     Expression
}

func (*conditionalExpressionNode) isExpressionBase()        {}
func (*conditionalExpressionNode) isNodeBase()              {}
func (*conditionalExpressionNode) isConditionalExpression() {}
func (*conditionalExpressionNode) isBlockOrExpression()     {}
func (*conditionalExpressionNode) isConciseBody()           {}
func (*conditionalExpressionNode) isForInitializer()        {}
func (*conditionalExpressionNode) isNodeBody()              {}

func (n *conditionalExpressionNode) Condition() Expression {
	return n.condition
}

func (n *conditionalExpressionNode) QuestionToken() QuestionToken {
	return n.questionToken
}

func (n *conditionalExpressionNode) WhenTrue() Expression {
	return n.whenTrue
}

func (n *conditionalExpressionNode) ColonToken() ColonToken {
	return n.colonToken
}

func (n *conditionalExpressionNode) WhenFalse() Expression {
	return n.whenFalse
}

type TemplateExpression interface {
	PrimaryExpressionBase
	isTemplateExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	isTemplateLiteral()
	Head() TemplateHead
	TemplateSpans() []TemplateSpan
}

type templateExpressionNode struct {
	nodeCore
	head          TemplateHead
	templateSpans []TemplateSpan
}

func (*templateExpressionNode) isExpressionBase()             {}
func (*templateExpressionNode) isLeftHandSideExpressionBase() {}
func (*templateExpressionNode) isMemberExpressionBase()       {}
func (*templateExpressionNode) isNodeBase()                   {}
func (*templateExpressionNode) isPrimaryExpressionBase()      {}
func (*templateExpressionNode) isUnaryExpressionBase()        {}
func (*templateExpressionNode) isUpdateExpressionBase()       {}
func (*templateExpressionNode) isTemplateExpression()         {}
func (*templateExpressionNode) isBlockOrExpression()          {}
func (*templateExpressionNode) isConciseBody()                {}
func (*templateExpressionNode) isForInitializer()             {}
func (*templateExpressionNode) isIncrementExpression()        {}
func (*templateExpressionNode) isNodeBody()                   {}
func (*templateExpressionNode) isTemplateLiteral()            {}

func (n *templateExpressionNode) Head() TemplateHead {
	return n.head
}

func (n *templateExpressionNode) TemplateSpans() []TemplateSpan {
	return cloneSlice(n.templateSpans)
}

type YieldExpression interface {
	ExpressionBase
	isYieldExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	AsteriskToken() AsteriskToken
	Expression() Expression
}

type yieldExpressionNode struct {
	nodeCore
	asteriskToken AsteriskToken
	expression    Expression
}

func (*yieldExpressionNode) isExpressionBase()    {}
func (*yieldExpressionNode) isNodeBase()          {}
func (*yieldExpressionNode) isYieldExpression()   {}
func (*yieldExpressionNode) isBlockOrExpression() {}
func (*yieldExpressionNode) isConciseBody()       {}
func (*yieldExpressionNode) isForInitializer()    {}
func (*yieldExpressionNode) isNodeBody()          {}

func (n *yieldExpressionNode) AsteriskToken() AsteriskToken {
	return n.asteriskToken
}

func (n *yieldExpressionNode) Expression() Expression {
	return n.expression
}

type SpreadElement interface {
	ExpressionBase
	isSpreadElement()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
}

type spreadElementNode struct {
	nodeCore
	expression Expression
}

func (*spreadElementNode) isExpressionBase()    {}
func (*spreadElementNode) isNodeBase()          {}
func (*spreadElementNode) isSpreadElement()     {}
func (*spreadElementNode) isBlockOrExpression() {}
func (*spreadElementNode) isConciseBody()       {}
func (*spreadElementNode) isForInitializer()    {}
func (*spreadElementNode) isNodeBody()          {}

func (n *spreadElementNode) Expression() Expression {
	return n.expression
}

type ClassExpression interface {
	PrimaryExpressionBase
	ClassLikeBase
	isClassExpression()
	isBlockOrExpression()
	isClassLikeDeclaration()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	isObjectTypeDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	HeritageClauses() []HeritageClause
	Members() []ClassElement
}

type classExpressionNode struct {
	nodeCore
	modifiers       []ModifierLike
	name            Identifier
	typeParameters  []TypeParameterDeclaration
	heritageClauses []HeritageClause
	members         []ClassElement
}

func (*classExpressionNode) isClassLikeBase()              {}
func (*classExpressionNode) isDeclarationBase()            {}
func (*classExpressionNode) isExpressionBase()             {}
func (*classExpressionNode) isLeftHandSideExpressionBase() {}
func (*classExpressionNode) isMemberExpressionBase()       {}
func (*classExpressionNode) isModifiersBase()              {}
func (*classExpressionNode) isNodeBase()                   {}
func (*classExpressionNode) isPrimaryExpressionBase()      {}
func (*classExpressionNode) isUnaryExpressionBase()        {}
func (*classExpressionNode) isUpdateExpressionBase()       {}
func (*classExpressionNode) isClassExpression()            {}
func (*classExpressionNode) isBlockOrExpression()          {}
func (*classExpressionNode) isClassLikeDeclaration()       {}
func (*classExpressionNode) isConciseBody()                {}
func (*classExpressionNode) isForInitializer()             {}
func (*classExpressionNode) isIncrementExpression()        {}
func (*classExpressionNode) isNodeBody()                   {}
func (*classExpressionNode) isObjectTypeDeclaration()      {}

func (n *classExpressionNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *classExpressionNode) Name() Identifier {
	return n.name
}

func (n *classExpressionNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *classExpressionNode) HeritageClauses() []HeritageClause {
	return cloneSlice(n.heritageClauses)
}

func (n *classExpressionNode) Members() []ClassElement {
	return cloneSlice(n.members)
}

type OmittedExpression interface {
	ExpressionBase
	isOmittedExpression()
	isArrayBindingElement()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
}

type omittedExpressionNode struct {
	nodeCore
}

func (*omittedExpressionNode) isExpressionBase()      {}
func (*omittedExpressionNode) isNodeBase()            {}
func (*omittedExpressionNode) isOmittedExpression()   {}
func (*omittedExpressionNode) isArrayBindingElement() {}
func (*omittedExpressionNode) isBlockOrExpression()   {}
func (*omittedExpressionNode) isConciseBody()         {}
func (*omittedExpressionNode) isForInitializer()      {}
func (*omittedExpressionNode) isNodeBody()            {}

type ExpressionWithTypeArguments interface {
	MemberExpressionBase
	isExpressionWithTypeArguments()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
	TypeArguments() []TypeNode
}

type expressionWithTypeArgumentsNode struct {
	nodeCore
	expression    Expression
	typeArguments []TypeNode
}

func (*expressionWithTypeArgumentsNode) isExpressionBase()              {}
func (*expressionWithTypeArgumentsNode) isLeftHandSideExpressionBase()  {}
func (*expressionWithTypeArgumentsNode) isMemberExpressionBase()        {}
func (*expressionWithTypeArgumentsNode) isNodeBase()                    {}
func (*expressionWithTypeArgumentsNode) isUnaryExpressionBase()         {}
func (*expressionWithTypeArgumentsNode) isUpdateExpressionBase()        {}
func (*expressionWithTypeArgumentsNode) isExpressionWithTypeArguments() {}
func (*expressionWithTypeArgumentsNode) isBlockOrExpression()           {}
func (*expressionWithTypeArgumentsNode) isConciseBody()                 {}
func (*expressionWithTypeArgumentsNode) isForInitializer()              {}
func (*expressionWithTypeArgumentsNode) isIncrementExpression()         {}
func (*expressionWithTypeArgumentsNode) isNodeBody()                    {}

func (n *expressionWithTypeArgumentsNode) Expression() Expression {
	return n.expression
}

func (n *expressionWithTypeArgumentsNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

type AsExpression interface {
	ExpressionBase
	isAsExpression()
	isAssertionExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
	Type() TypeNode
}

type asExpressionNode struct {
	nodeCore
	expression Expression
	typeNode   TypeNode
}

func (*asExpressionNode) isExpressionBase()      {}
func (*asExpressionNode) isNodeBase()            {}
func (*asExpressionNode) isAsExpression()        {}
func (*asExpressionNode) isAssertionExpression() {}
func (*asExpressionNode) isBlockOrExpression()   {}
func (*asExpressionNode) isConciseBody()         {}
func (*asExpressionNode) isForInitializer()      {}
func (*asExpressionNode) isNodeBody()            {}

func (n *asExpressionNode) Expression() Expression {
	return n.expression
}

func (n *asExpressionNode) Type() TypeNode {
	return n.typeNode
}

type NonNullExpression interface {
	LeftHandSideExpressionBase
	isNonNullExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
}

type nonNullExpressionNode struct {
	nodeCore
	expression Expression
}

func (*nonNullExpressionNode) isExpressionBase()             {}
func (*nonNullExpressionNode) isLeftHandSideExpressionBase() {}
func (*nonNullExpressionNode) isNodeBase()                   {}
func (*nonNullExpressionNode) isUnaryExpressionBase()        {}
func (*nonNullExpressionNode) isUpdateExpressionBase()       {}
func (*nonNullExpressionNode) isNonNullExpression()          {}
func (*nonNullExpressionNode) isBlockOrExpression()          {}
func (*nonNullExpressionNode) isConciseBody()                {}
func (*nonNullExpressionNode) isForInitializer()             {}
func (*nonNullExpressionNode) isIncrementExpression()        {}
func (*nonNullExpressionNode) isNodeBody()                   {}

func (n *nonNullExpressionNode) Expression() Expression {
	return n.expression
}

type MetaProperty interface {
	PrimaryExpressionBase
	isMetaProperty()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	KeywordToken() MetaPropertyKeywordTokenKind
	Name() Identifier
}

type metaPropertyNode struct {
	nodeCore
	keywordToken MetaPropertyKeywordTokenKind
	name         Identifier
}

func (*metaPropertyNode) isExpressionBase()             {}
func (*metaPropertyNode) isLeftHandSideExpressionBase() {}
func (*metaPropertyNode) isMemberExpressionBase()       {}
func (*metaPropertyNode) isNodeBase()                   {}
func (*metaPropertyNode) isPrimaryExpressionBase()      {}
func (*metaPropertyNode) isUnaryExpressionBase()        {}
func (*metaPropertyNode) isUpdateExpressionBase()       {}
func (*metaPropertyNode) isMetaProperty()               {}
func (*metaPropertyNode) isBlockOrExpression()          {}
func (*metaPropertyNode) isConciseBody()                {}
func (*metaPropertyNode) isForInitializer()             {}
func (*metaPropertyNode) isIncrementExpression()        {}
func (*metaPropertyNode) isNodeBody()                   {}

func (n *metaPropertyNode) KeywordToken() MetaPropertyKeywordTokenKind {
	return n.keywordToken
}

func (n *metaPropertyNode) Name() Identifier {
	return n.name
}

type SyntheticExpression interface {
	ExpressionBase
	isSyntheticExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	IsSpread() bool
	TupleNameSource() Node
}

type syntheticExpressionNode struct {
	nodeCore
	isSpread        bool
	tupleNameSource Node
}

func (*syntheticExpressionNode) isExpressionBase()      {}
func (*syntheticExpressionNode) isNodeBase()            {}
func (*syntheticExpressionNode) isSyntheticExpression() {}
func (*syntheticExpressionNode) isBlockOrExpression()   {}
func (*syntheticExpressionNode) isConciseBody()         {}
func (*syntheticExpressionNode) isForInitializer()      {}
func (*syntheticExpressionNode) isNodeBody()            {}

func (n *syntheticExpressionNode) IsSpread() bool {
	return n.isSpread
}

func (n *syntheticExpressionNode) TupleNameSource() Node {
	return n.tupleNameSource
}

type SatisfiesExpression interface {
	ExpressionBase
	isSatisfiesExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
	Type() TypeNode
}

type satisfiesExpressionNode struct {
	nodeCore
	expression Expression
	typeNode   TypeNode
}

func (*satisfiesExpressionNode) isExpressionBase()      {}
func (*satisfiesExpressionNode) isNodeBase()            {}
func (*satisfiesExpressionNode) isSatisfiesExpression() {}
func (*satisfiesExpressionNode) isBlockOrExpression()   {}
func (*satisfiesExpressionNode) isConciseBody()         {}
func (*satisfiesExpressionNode) isForInitializer()      {}
func (*satisfiesExpressionNode) isNodeBody()            {}

func (n *satisfiesExpressionNode) Expression() Expression {
	return n.expression
}

func (n *satisfiesExpressionNode) Type() TypeNode {
	return n.typeNode
}

type TemplateSpan interface {
	NodeBase
	isTemplateSpan()
	Expression() Expression
	Literal() TemplateMiddleOrTail
}

type templateSpanNode struct {
	nodeCore
	expression Expression
	literal    TemplateMiddleOrTail
}

func (*templateSpanNode) isNodeBase()     {}
func (*templateSpanNode) isTemplateSpan() {}

func (n *templateSpanNode) Expression() Expression {
	return n.expression
}

func (n *templateSpanNode) Literal() TemplateMiddleOrTail {
	return n.literal
}

type SemicolonClassElement interface {
	NodeBase
	DeclarationBase
	ClassElementBase
	isSemicolonClassElement()
}

type semicolonClassElementNode struct {
	nodeCore
}

func (*semicolonClassElementNode) isClassElementBase()      {}
func (*semicolonClassElementNode) isDeclarationBase()       {}
func (*semicolonClassElementNode) isNodeBase()              {}
func (*semicolonClassElementNode) isSemicolonClassElement() {}

type Block interface {
	StatementBase
	isBlock()
	isBlockOrExpression()
	isConciseBody()
	isFunctionBody()
	isNodeBody()
	Statements() []Statement
	MultiLine() bool
}

type blockNode struct {
	nodeCore
	statements []Statement
	multiLine  bool
}

func (*blockNode) isNodeBase()          {}
func (*blockNode) isStatementBase()     {}
func (*blockNode) isBlock()             {}
func (*blockNode) isBlockOrExpression() {}
func (*blockNode) isConciseBody()       {}
func (*blockNode) isFunctionBody()      {}
func (*blockNode) isNodeBody()          {}

func (n *blockNode) Statements() []Statement {
	return cloneSlice(n.statements)
}

func (n *blockNode) MultiLine() bool {
	return n.multiLine
}

type EmptyStatement interface {
	StatementBase
	isEmptyStatement()
}

type emptyStatementNode struct {
	nodeCore
}

func (*emptyStatementNode) isNodeBase()       {}
func (*emptyStatementNode) isStatementBase()  {}
func (*emptyStatementNode) isEmptyStatement() {}

type VariableStatement interface {
	StatementBase
	ModifiersBase
	isVariableStatement()
	Modifiers() []ModifierLike
	DeclarationList() VariableDeclarationList
}

type variableStatementNode struct {
	nodeCore
	modifiers       []ModifierLike
	declarationList VariableDeclarationList
}

func (*variableStatementNode) isModifiersBase()     {}
func (*variableStatementNode) isNodeBase()          {}
func (*variableStatementNode) isStatementBase()     {}
func (*variableStatementNode) isVariableStatement() {}

func (n *variableStatementNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *variableStatementNode) DeclarationList() VariableDeclarationList {
	return n.declarationList
}

type ExpressionStatement interface {
	StatementBase
	isExpressionStatement()
	Expression() Expression
}

type expressionStatementNode struct {
	nodeCore
	expression Expression
}

func (*expressionStatementNode) isNodeBase()            {}
func (*expressionStatementNode) isStatementBase()       {}
func (*expressionStatementNode) isExpressionStatement() {}

func (n *expressionStatementNode) Expression() Expression {
	return n.expression
}

type IfStatement interface {
	StatementBase
	isIfStatement()
	Expression() Expression
	ThenStatement() Statement
	ElseStatement() Statement
}

type ifStatementNode struct {
	nodeCore
	expression    Expression
	thenStatement Statement
	elseStatement Statement
}

func (*ifStatementNode) isNodeBase()      {}
func (*ifStatementNode) isStatementBase() {}
func (*ifStatementNode) isIfStatement()   {}

func (n *ifStatementNode) Expression() Expression {
	return n.expression
}

func (n *ifStatementNode) ThenStatement() Statement {
	return n.thenStatement
}

func (n *ifStatementNode) ElseStatement() Statement {
	return n.elseStatement
}

type DoStatement interface {
	IterationStatementBase
	isDoStatement()
	Statement() Statement
	Expression() Expression
}

type doStatementNode struct {
	nodeCore
	statement  Statement
	expression Expression
}

func (*doStatementNode) isIterationStatementBase() {}
func (*doStatementNode) isNodeBase()               {}
func (*doStatementNode) isStatementBase()          {}
func (*doStatementNode) isDoStatement()            {}

func (n *doStatementNode) Statement() Statement {
	return n.statement
}

func (n *doStatementNode) Expression() Expression {
	return n.expression
}

type WhileStatement interface {
	IterationStatementBase
	isWhileStatement()
	Expression() Expression
	Statement() Statement
}

type whileStatementNode struct {
	nodeCore
	expression Expression
	statement  Statement
}

func (*whileStatementNode) isIterationStatementBase() {}
func (*whileStatementNode) isNodeBase()               {}
func (*whileStatementNode) isStatementBase()          {}
func (*whileStatementNode) isWhileStatement()         {}

func (n *whileStatementNode) Expression() Expression {
	return n.expression
}

func (n *whileStatementNode) Statement() Statement {
	return n.statement
}

type ForStatement interface {
	IterationStatementBase
	isForStatement()
	Initializer() ForInitializer
	Condition() Expression
	Incrementor() Expression
	Statement() Statement
}

type forStatementNode struct {
	nodeCore
	initializer ForInitializer
	condition   Expression
	incrementor Expression
	statement   Statement
}

func (*forStatementNode) isIterationStatementBase() {}
func (*forStatementNode) isNodeBase()               {}
func (*forStatementNode) isStatementBase()          {}
func (*forStatementNode) isForStatement()           {}

func (n *forStatementNode) Initializer() ForInitializer {
	return n.initializer
}

func (n *forStatementNode) Condition() Expression {
	return n.condition
}

func (n *forStatementNode) Incrementor() Expression {
	return n.incrementor
}

func (n *forStatementNode) Statement() Statement {
	return n.statement
}

type ForInStatement interface {
	ForInOrOfStatement
	isForInStatement()
	AwaitModifier() AwaitKeyword
	Initializer() ForInitializer
	Expression() Expression
	Statement() Statement
}

type forInStatementNode struct {
	nodeCore
	awaitModifier AwaitKeyword
	initializer   ForInitializer
	expression    Expression
	statement     Statement
}

func (*forInStatementNode) isNodeBase()           {}
func (*forInStatementNode) isStatementBase()      {}
func (*forInStatementNode) isForInOrOfStatement() {}
func (*forInStatementNode) isForInStatement()     {}

func (n *forInStatementNode) AwaitModifier() AwaitKeyword {
	return n.awaitModifier
}

func (n *forInStatementNode) Initializer() ForInitializer {
	return n.initializer
}

func (n *forInStatementNode) Expression() Expression {
	return n.expression
}

func (n *forInStatementNode) Statement() Statement {
	return n.statement
}

type ForOfStatement interface {
	ForInOrOfStatement
	isForOfStatement()
	AwaitModifier() AwaitKeyword
	Initializer() ForInitializer
	Expression() Expression
	Statement() Statement
}

type forOfStatementNode struct {
	nodeCore
	awaitModifier AwaitKeyword
	initializer   ForInitializer
	expression    Expression
	statement     Statement
}

func (*forOfStatementNode) isNodeBase()           {}
func (*forOfStatementNode) isStatementBase()      {}
func (*forOfStatementNode) isForInOrOfStatement() {}
func (*forOfStatementNode) isForOfStatement()     {}

func (n *forOfStatementNode) AwaitModifier() AwaitKeyword {
	return n.awaitModifier
}

func (n *forOfStatementNode) Initializer() ForInitializer {
	return n.initializer
}

func (n *forOfStatementNode) Expression() Expression {
	return n.expression
}

func (n *forOfStatementNode) Statement() Statement {
	return n.statement
}

type ContinueStatement interface {
	StatementBase
	isContinueStatement()
	isBreakOrContinueStatement()
	Label() Identifier
}

type continueStatementNode struct {
	nodeCore
	label Identifier
}

func (*continueStatementNode) isNodeBase()                 {}
func (*continueStatementNode) isStatementBase()            {}
func (*continueStatementNode) isContinueStatement()        {}
func (*continueStatementNode) isBreakOrContinueStatement() {}

func (n *continueStatementNode) Label() Identifier {
	return n.label
}

type BreakStatement interface {
	StatementBase
	isBreakStatement()
	isBreakOrContinueStatement()
	Label() Identifier
}

type breakStatementNode struct {
	nodeCore
	label Identifier
}

func (*breakStatementNode) isNodeBase()                 {}
func (*breakStatementNode) isStatementBase()            {}
func (*breakStatementNode) isBreakStatement()           {}
func (*breakStatementNode) isBreakOrContinueStatement() {}

func (n *breakStatementNode) Label() Identifier {
	return n.label
}

type ReturnStatement interface {
	StatementBase
	isReturnStatement()
	Expression() Expression
}

type returnStatementNode struct {
	nodeCore
	expression Expression
}

func (*returnStatementNode) isNodeBase()        {}
func (*returnStatementNode) isStatementBase()   {}
func (*returnStatementNode) isReturnStatement() {}

func (n *returnStatementNode) Expression() Expression {
	return n.expression
}

type WithStatement interface {
	StatementBase
	isWithStatement()
	Expression() Expression
	Statement() Statement
}

type withStatementNode struct {
	nodeCore
	expression Expression
	statement  Statement
}

func (*withStatementNode) isNodeBase()      {}
func (*withStatementNode) isStatementBase() {}
func (*withStatementNode) isWithStatement() {}

func (n *withStatementNode) Expression() Expression {
	return n.expression
}

func (n *withStatementNode) Statement() Statement {
	return n.statement
}

type SwitchStatement interface {
	StatementBase
	isSwitchStatement()
	Expression() Expression
	CaseBlock() CaseBlock
}

type switchStatementNode struct {
	nodeCore
	expression Expression
	caseBlock  CaseBlock
}

func (*switchStatementNode) isNodeBase()        {}
func (*switchStatementNode) isStatementBase()   {}
func (*switchStatementNode) isSwitchStatement() {}

func (n *switchStatementNode) Expression() Expression {
	return n.expression
}

func (n *switchStatementNode) CaseBlock() CaseBlock {
	return n.caseBlock
}

type LabeledStatement interface {
	StatementBase
	isLabeledStatement()
	Label() Identifier
	Statement() Statement
}

type labeledStatementNode struct {
	nodeCore
	label     Identifier
	statement Statement
}

func (*labeledStatementNode) isNodeBase()         {}
func (*labeledStatementNode) isStatementBase()    {}
func (*labeledStatementNode) isLabeledStatement() {}

func (n *labeledStatementNode) Label() Identifier {
	return n.label
}

func (n *labeledStatementNode) Statement() Statement {
	return n.statement
}

type ThrowStatement interface {
	StatementBase
	isThrowStatement()
	Expression() Expression
}

type throwStatementNode struct {
	nodeCore
	expression Expression
}

func (*throwStatementNode) isNodeBase()       {}
func (*throwStatementNode) isStatementBase()  {}
func (*throwStatementNode) isThrowStatement() {}

func (n *throwStatementNode) Expression() Expression {
	return n.expression
}

type TryStatement interface {
	StatementBase
	isTryStatement()
	TryBlock() Block
	CatchClause() CatchClause
	FinallyBlock() Block
}

type tryStatementNode struct {
	nodeCore
	tryBlock     Block
	catchClause  CatchClause
	finallyBlock Block
}

func (*tryStatementNode) isNodeBase()      {}
func (*tryStatementNode) isStatementBase() {}
func (*tryStatementNode) isTryStatement()  {}

func (n *tryStatementNode) TryBlock() Block {
	return n.tryBlock
}

func (n *tryStatementNode) CatchClause() CatchClause {
	return n.catchClause
}

func (n *tryStatementNode) FinallyBlock() Block {
	return n.finallyBlock
}

type DebuggerStatement interface {
	StatementBase
	isDebuggerStatement()
}

type debuggerStatementNode struct {
	nodeCore
}

func (*debuggerStatementNode) isNodeBase()          {}
func (*debuggerStatementNode) isStatementBase()     {}
func (*debuggerStatementNode) isDebuggerStatement() {}

type VariableDeclaration interface {
	NodeBase
	DeclarationBase
	isVariableDeclaration()
	isVariableOrParameterDeclaration()
	isVariableOrPropertyDeclaration()
	Name() BindingName
	ExclamationToken() ExclamationToken
	Type() TypeNode
	Initializer() Expression
}

type variableDeclarationNode struct {
	nodeCore
	name             BindingName
	exclamationToken ExclamationToken
	typeNode         TypeNode
	initializer      Expression
}

func (*variableDeclarationNode) isDeclarationBase()                {}
func (*variableDeclarationNode) isNodeBase()                       {}
func (*variableDeclarationNode) isVariableDeclaration()            {}
func (*variableDeclarationNode) isVariableOrParameterDeclaration() {}
func (*variableDeclarationNode) isVariableOrPropertyDeclaration()  {}

func (n *variableDeclarationNode) Name() BindingName {
	return n.name
}

func (n *variableDeclarationNode) ExclamationToken() ExclamationToken {
	return n.exclamationToken
}

func (n *variableDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *variableDeclarationNode) Initializer() Expression {
	return n.initializer
}

type VariableDeclarationList interface {
	NodeBase
	isVariableDeclarationList()
	isForInitializer()
	Declarations() []VariableDeclaration
}

type variableDeclarationListNode struct {
	nodeCore
	declarations []VariableDeclaration
}

func (*variableDeclarationListNode) isNodeBase()                {}
func (*variableDeclarationListNode) isVariableDeclarationList() {}
func (*variableDeclarationListNode) isForInitializer()          {}

func (n *variableDeclarationListNode) Declarations() []VariableDeclaration {
	return cloneSlice(n.declarations)
}

type FunctionDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	FunctionLikeWithBodyBase
	isFunctionDeclaration()
	isFunctionLikeDeclaration()
	isSignatureDeclaration()
	Modifiers() []ModifierLike
	AsteriskToken() AsteriskToken
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
	Body() FunctionBody
}

type functionDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	asteriskToken  AsteriskToken
	name           Identifier
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
	body           FunctionBody
}

func (*functionDeclarationNode) isBodyBase()                 {}
func (*functionDeclarationNode) isDeclarationBase()          {}
func (*functionDeclarationNode) isFunctionLikeBase()         {}
func (*functionDeclarationNode) isFunctionLikeWithBodyBase() {}
func (*functionDeclarationNode) isModifiersBase()            {}
func (*functionDeclarationNode) isNodeBase()                 {}
func (*functionDeclarationNode) isStatementBase()            {}
func (*functionDeclarationNode) isFunctionDeclaration()      {}
func (*functionDeclarationNode) isFunctionLikeDeclaration()  {}
func (*functionDeclarationNode) isSignatureDeclaration()     {}

func (n *functionDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *functionDeclarationNode) AsteriskToken() AsteriskToken {
	return n.asteriskToken
}

func (n *functionDeclarationNode) Name() Identifier {
	return n.name
}

func (n *functionDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *functionDeclarationNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *functionDeclarationNode) Type() TypeNode {
	return n.typeNode
}

func (n *functionDeclarationNode) Body() FunctionBody {
	return n.body
}

type ClassDeclaration interface {
	DeclarationBase
	StatementBase
	ClassLikeBase
	isClassDeclaration()
	isClassLikeDeclaration()
	isObjectTypeDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	HeritageClauses() []HeritageClause
	Members() []ClassElement
}

type classDeclarationNode struct {
	nodeCore
	modifiers       []ModifierLike
	name            Identifier
	typeParameters  []TypeParameterDeclaration
	heritageClauses []HeritageClause
	members         []ClassElement
}

func (*classDeclarationNode) isClassLikeBase()         {}
func (*classDeclarationNode) isDeclarationBase()       {}
func (*classDeclarationNode) isModifiersBase()         {}
func (*classDeclarationNode) isNodeBase()              {}
func (*classDeclarationNode) isStatementBase()         {}
func (*classDeclarationNode) isClassDeclaration()      {}
func (*classDeclarationNode) isClassLikeDeclaration()  {}
func (*classDeclarationNode) isObjectTypeDeclaration() {}

func (n *classDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *classDeclarationNode) Name() Identifier {
	return n.name
}

func (n *classDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *classDeclarationNode) HeritageClauses() []HeritageClause {
	return cloneSlice(n.heritageClauses)
}

func (n *classDeclarationNode) Members() []ClassElement {
	return cloneSlice(n.members)
}

type InterfaceDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isInterfaceDeclaration()
	isObjectTypeDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	HeritageClauses() []HeritageClause
	Members() []TypeElement
}

type interfaceDeclarationNode struct {
	nodeCore
	modifiers       []ModifierLike
	name            Identifier
	typeParameters  []TypeParameterDeclaration
	heritageClauses []HeritageClause
	members         []TypeElement
}

func (*interfaceDeclarationNode) isDeclarationBase()       {}
func (*interfaceDeclarationNode) isModifiersBase()         {}
func (*interfaceDeclarationNode) isNodeBase()              {}
func (*interfaceDeclarationNode) isStatementBase()         {}
func (*interfaceDeclarationNode) isInterfaceDeclaration()  {}
func (*interfaceDeclarationNode) isObjectTypeDeclaration() {}

func (n *interfaceDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *interfaceDeclarationNode) Name() Identifier {
	return n.name
}

func (n *interfaceDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *interfaceDeclarationNode) HeritageClauses() []HeritageClause {
	return cloneSlice(n.heritageClauses)
}

func (n *interfaceDeclarationNode) Members() []TypeElement {
	return cloneSlice(n.members)
}

type TypeAliasDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isTypeAliasDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	TypeParameters() []TypeParameterDeclaration
	Type() TypeNode
}

type typeAliasDeclarationNode struct {
	nodeCore
	modifiers      []ModifierLike
	name           Identifier
	typeParameters []TypeParameterDeclaration
	typeNode       TypeNode
}

func (*typeAliasDeclarationNode) isDeclarationBase()      {}
func (*typeAliasDeclarationNode) isModifiersBase()        {}
func (*typeAliasDeclarationNode) isNodeBase()             {}
func (*typeAliasDeclarationNode) isStatementBase()        {}
func (*typeAliasDeclarationNode) isTypeAliasDeclaration() {}

func (n *typeAliasDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *typeAliasDeclarationNode) Name() Identifier {
	return n.name
}

func (n *typeAliasDeclarationNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *typeAliasDeclarationNode) Type() TypeNode {
	return n.typeNode
}

type EnumDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isEnumDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
	Members() []EnumMember
}

type enumDeclarationNode struct {
	nodeCore
	modifiers []ModifierLike
	name      Identifier
	members   []EnumMember
}

func (*enumDeclarationNode) isDeclarationBase() {}
func (*enumDeclarationNode) isModifiersBase()   {}
func (*enumDeclarationNode) isNodeBase()        {}
func (*enumDeclarationNode) isStatementBase()   {}
func (*enumDeclarationNode) isEnumDeclaration() {}

func (n *enumDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *enumDeclarationNode) Name() Identifier {
	return n.name
}

func (n *enumDeclarationNode) Members() []EnumMember {
	return cloneSlice(n.members)
}

type ModuleDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	BodyBase
	isModuleDeclaration()
	isJSDocFullName()
	isModuleBody()
	isNodeBody()
	Modifiers() []ModifierLike
	Keyword() ModuleDeclarationKeywordKind
	Name() ModuleName
	Body() ModuleBody
}

type moduleDeclarationNode struct {
	nodeCore
	modifiers []ModifierLike
	keyword   ModuleDeclarationKeywordKind
	name      ModuleName
	body      ModuleBody
}

func (*moduleDeclarationNode) isBodyBase()          {}
func (*moduleDeclarationNode) isDeclarationBase()   {}
func (*moduleDeclarationNode) isModifiersBase()     {}
func (*moduleDeclarationNode) isNodeBase()          {}
func (*moduleDeclarationNode) isStatementBase()     {}
func (*moduleDeclarationNode) isModuleDeclaration() {}
func (*moduleDeclarationNode) isJSDocFullName()     {}
func (*moduleDeclarationNode) isModuleBody()        {}
func (*moduleDeclarationNode) isNodeBody()          {}

func (n *moduleDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *moduleDeclarationNode) Keyword() ModuleDeclarationKeywordKind {
	return n.keyword
}

func (n *moduleDeclarationNode) Name() ModuleName {
	return n.name
}

func (n *moduleDeclarationNode) Body() ModuleBody {
	return n.body
}

type ModuleBlock interface {
	StatementBase
	isModuleBlock()
	isModuleBody()
	isNodeBody()
	Statements() []Statement
}

type moduleBlockNode struct {
	nodeCore
	statements []Statement
}

func (*moduleBlockNode) isNodeBase()      {}
func (*moduleBlockNode) isStatementBase() {}
func (*moduleBlockNode) isModuleBlock()   {}
func (*moduleBlockNode) isModuleBody()    {}
func (*moduleBlockNode) isNodeBody()      {}

func (n *moduleBlockNode) Statements() []Statement {
	return cloneSlice(n.statements)
}

type CaseBlock interface {
	NodeBase
	isCaseBlock()
	Clauses() []CaseOrDefaultClause
}

type caseBlockNode struct {
	nodeCore
	clauses []CaseOrDefaultClause
}

func (*caseBlockNode) isNodeBase()  {}
func (*caseBlockNode) isCaseBlock() {}

func (n *caseBlockNode) Clauses() []CaseOrDefaultClause {
	return cloneSlice(n.clauses)
}

type NamespaceExportDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isNamespaceExportDeclaration()
	Modifiers() []ModifierLike
	Name() Identifier
}

type namespaceExportDeclarationNode struct {
	nodeCore
	modifiers []ModifierLike
	name      Identifier
}

func (*namespaceExportDeclarationNode) isDeclarationBase()            {}
func (*namespaceExportDeclarationNode) isModifiersBase()              {}
func (*namespaceExportDeclarationNode) isNodeBase()                   {}
func (*namespaceExportDeclarationNode) isStatementBase()              {}
func (*namespaceExportDeclarationNode) isNamespaceExportDeclaration() {}

func (n *namespaceExportDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *namespaceExportDeclarationNode) Name() Identifier {
	return n.name
}

type ImportEqualsDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isImportEqualsDeclaration()
	isAnyImportSyntax()
	Modifiers() []ModifierLike
	IsTypeOnly() bool
	Name() Identifier
	ModuleReference() ModuleReference
}

type importEqualsDeclarationNode struct {
	nodeCore
	modifiers       []ModifierLike
	isTypeOnly      bool
	name            Identifier
	moduleReference ModuleReference
}

func (*importEqualsDeclarationNode) isDeclarationBase()         {}
func (*importEqualsDeclarationNode) isModifiersBase()           {}
func (*importEqualsDeclarationNode) isNodeBase()                {}
func (*importEqualsDeclarationNode) isStatementBase()           {}
func (*importEqualsDeclarationNode) isImportEqualsDeclaration() {}
func (*importEqualsDeclarationNode) isAnyImportSyntax()         {}

func (n *importEqualsDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *importEqualsDeclarationNode) IsTypeOnly() bool {
	return n.isTypeOnly
}

func (n *importEqualsDeclarationNode) Name() Identifier {
	return n.name
}

func (n *importEqualsDeclarationNode) ModuleReference() ModuleReference {
	return n.moduleReference
}

type ImportDeclaration interface {
	StatementBase
	ModifiersBase
	DeclarationBase
	isImportDeclaration()
	isAnyImportSyntax()
	Modifiers() []ModifierLike
	ImportClause() ImportClause
	ModuleSpecifier() Expression
	Attributes() ImportAttributes
}

type importDeclarationNode struct {
	nodeCore
	modifiers       []ModifierLike
	importClause    ImportClause
	moduleSpecifier Expression
	attributes      ImportAttributes
}

func (*importDeclarationNode) isDeclarationBase()   {}
func (*importDeclarationNode) isModifiersBase()     {}
func (*importDeclarationNode) isNodeBase()          {}
func (*importDeclarationNode) isStatementBase()     {}
func (*importDeclarationNode) isImportDeclaration() {}
func (*importDeclarationNode) isAnyImportSyntax()   {}

func (n *importDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *importDeclarationNode) ImportClause() ImportClause {
	return n.importClause
}

func (n *importDeclarationNode) ModuleSpecifier() Expression {
	return n.moduleSpecifier
}

func (n *importDeclarationNode) Attributes() ImportAttributes {
	return n.attributes
}

type ImportClause interface {
	NodeBase
	DeclarationBase
	isImportClause()
	isImportClauseOrBindingPattern()
	PhaseModifier() ImportPhaseModifierSyntaxKind
	Name() Identifier
	NamedBindings() NamedImportBindings
}

type importClauseNode struct {
	nodeCore
	phaseModifier ImportPhaseModifierSyntaxKind
	name          Identifier
	namedBindings NamedImportBindings
}

func (*importClauseNode) isDeclarationBase()              {}
func (*importClauseNode) isNodeBase()                     {}
func (*importClauseNode) isImportClause()                 {}
func (*importClauseNode) isImportClauseOrBindingPattern() {}

func (n *importClauseNode) PhaseModifier() ImportPhaseModifierSyntaxKind {
	return n.phaseModifier
}

func (n *importClauseNode) Name() Identifier {
	return n.name
}

func (n *importClauseNode) NamedBindings() NamedImportBindings {
	return n.namedBindings
}

type NamespaceImport interface {
	NodeBase
	DeclarationBase
	isNamespaceImport()
	isNamedImportBindings()
	Name() Identifier
}

type namespaceImportNode struct {
	nodeCore
	name Identifier
}

func (*namespaceImportNode) isDeclarationBase()     {}
func (*namespaceImportNode) isNodeBase()            {}
func (*namespaceImportNode) isNamespaceImport()     {}
func (*namespaceImportNode) isNamedImportBindings() {}

func (n *namespaceImportNode) Name() Identifier {
	return n.name
}

type NamedImports interface {
	NodeBase
	isNamedImports()
	isNamedImportBindings()
	isNamedImportsOrExports()
	Elements() []ImportSpecifier
}

type namedImportsNode struct {
	nodeCore
	elements []ImportSpecifier
}

func (*namedImportsNode) isNodeBase()              {}
func (*namedImportsNode) isNamedImports()          {}
func (*namedImportsNode) isNamedImportBindings()   {}
func (*namedImportsNode) isNamedImportsOrExports() {}

func (n *namedImportsNode) Elements() []ImportSpecifier {
	return cloneSlice(n.elements)
}

type ImportSpecifier interface {
	NodeBase
	DeclarationBase
	isImportSpecifier()
	IsTypeOnly() bool
	PropertyName() ModuleExportName
	Name() Identifier
}

type importSpecifierNode struct {
	nodeCore
	isTypeOnly   bool
	propertyName ModuleExportName
	name         Identifier
}

func (*importSpecifierNode) isDeclarationBase() {}
func (*importSpecifierNode) isNodeBase()        {}
func (*importSpecifierNode) isImportSpecifier() {}

func (n *importSpecifierNode) IsTypeOnly() bool {
	return n.isTypeOnly
}

func (n *importSpecifierNode) PropertyName() ModuleExportName {
	return n.propertyName
}

func (n *importSpecifierNode) Name() Identifier {
	return n.name
}

type ExportAssignment interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isExportAssignment()
	Modifiers() []ModifierLike
	IsExportEquals() bool
	Type() TypeNode
	Expression() Expression
}

type exportAssignmentNode struct {
	nodeCore
	modifiers      []ModifierLike
	isExportEquals bool
	typeNode       TypeNode
	expression     Expression
}

func (*exportAssignmentNode) isDeclarationBase()  {}
func (*exportAssignmentNode) isModifiersBase()    {}
func (*exportAssignmentNode) isNodeBase()         {}
func (*exportAssignmentNode) isStatementBase()    {}
func (*exportAssignmentNode) isExportAssignment() {}

func (n *exportAssignmentNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *exportAssignmentNode) IsExportEquals() bool {
	return n.isExportEquals
}

func (n *exportAssignmentNode) Type() TypeNode {
	return n.typeNode
}

func (n *exportAssignmentNode) Expression() Expression {
	return n.expression
}

type ExportDeclaration interface {
	DeclarationBase
	StatementBase
	ModifiersBase
	isExportDeclaration()
	Modifiers() []ModifierLike
	IsTypeOnly() bool
	ExportClause() NamedExportBindings
	ModuleSpecifier() Expression
	Attributes() ImportAttributes
}

type exportDeclarationNode struct {
	nodeCore
	modifiers       []ModifierLike
	isTypeOnly      bool
	exportClause    NamedExportBindings
	moduleSpecifier Expression
	attributes      ImportAttributes
}

func (*exportDeclarationNode) isDeclarationBase()   {}
func (*exportDeclarationNode) isModifiersBase()     {}
func (*exportDeclarationNode) isNodeBase()          {}
func (*exportDeclarationNode) isStatementBase()     {}
func (*exportDeclarationNode) isExportDeclaration() {}

func (n *exportDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *exportDeclarationNode) IsTypeOnly() bool {
	return n.isTypeOnly
}

func (n *exportDeclarationNode) ExportClause() NamedExportBindings {
	return n.exportClause
}

func (n *exportDeclarationNode) ModuleSpecifier() Expression {
	return n.moduleSpecifier
}

func (n *exportDeclarationNode) Attributes() ImportAttributes {
	return n.attributes
}

type NamedExports interface {
	NodeBase
	isNamedExports()
	isNamedExportBindings()
	isNamedImportsOrExports()
	Elements() []ExportSpecifier
}

type namedExportsNode struct {
	nodeCore
	elements []ExportSpecifier
}

func (*namedExportsNode) isNodeBase()              {}
func (*namedExportsNode) isNamedExports()          {}
func (*namedExportsNode) isNamedExportBindings()   {}
func (*namedExportsNode) isNamedImportsOrExports() {}

func (n *namedExportsNode) Elements() []ExportSpecifier {
	return cloneSlice(n.elements)
}

type NamespaceExport interface {
	NodeBase
	DeclarationBase
	isNamespaceExport()
	isNamedExportBindings()
	Name() ModuleExportName
}

type namespaceExportNode struct {
	nodeCore
	name ModuleExportName
}

func (*namespaceExportNode) isDeclarationBase()     {}
func (*namespaceExportNode) isNodeBase()            {}
func (*namespaceExportNode) isNamespaceExport()     {}
func (*namespaceExportNode) isNamedExportBindings() {}

func (n *namespaceExportNode) Name() ModuleExportName {
	return n.name
}

type ExportSpecifier interface {
	NodeBase
	DeclarationBase
	isExportSpecifier()
	IsTypeOnly() bool
	PropertyName() ModuleExportName
	Name() ModuleExportName
}

type exportSpecifierNode struct {
	nodeCore
	isTypeOnly   bool
	propertyName ModuleExportName
	name         ModuleExportName
}

func (*exportSpecifierNode) isDeclarationBase() {}
func (*exportSpecifierNode) isNodeBase()        {}
func (*exportSpecifierNode) isExportSpecifier() {}

func (n *exportSpecifierNode) IsTypeOnly() bool {
	return n.isTypeOnly
}

func (n *exportSpecifierNode) PropertyName() ModuleExportName {
	return n.propertyName
}

func (n *exportSpecifierNode) Name() ModuleExportName {
	return n.name
}

type MissingDeclaration interface {
	StatementBase
	DeclarationBase
	ModifiersBase
	isMissingDeclaration()
	isForInitializer()
	Modifiers() []ModifierLike
}

type missingDeclarationNode struct {
	nodeCore
	modifiers []ModifierLike
}

func (*missingDeclarationNode) isDeclarationBase()    {}
func (*missingDeclarationNode) isModifiersBase()      {}
func (*missingDeclarationNode) isNodeBase()           {}
func (*missingDeclarationNode) isStatementBase()      {}
func (*missingDeclarationNode) isMissingDeclaration() {}
func (*missingDeclarationNode) isForInitializer()     {}

func (n *missingDeclarationNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

type ExternalModuleReference interface {
	NodeBase
	isExternalModuleReference()
	isModuleReference()
	Expression() Expression
}

type externalModuleReferenceNode struct {
	nodeCore
	expression Expression
}

func (*externalModuleReferenceNode) isNodeBase()                {}
func (*externalModuleReferenceNode) isExternalModuleReference() {}
func (*externalModuleReferenceNode) isModuleReference()         {}

func (n *externalModuleReferenceNode) Expression() Expression {
	return n.expression
}

type JsxElement interface {
	PrimaryExpressionBase
	isJsxElement()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isJsxAttributeValue()
	isJsxChild()
	isNodeBody()
	OpeningElement() JsxOpeningElement
	Children() []JsxChild
	ClosingElement() JsxClosingElement
}

type jsxElementNode struct {
	nodeCore
	openingElement JsxOpeningElement
	children       []JsxChild
	closingElement JsxClosingElement
}

func (*jsxElementNode) isExpressionBase()             {}
func (*jsxElementNode) isLeftHandSideExpressionBase() {}
func (*jsxElementNode) isMemberExpressionBase()       {}
func (*jsxElementNode) isNodeBase()                   {}
func (*jsxElementNode) isPrimaryExpressionBase()      {}
func (*jsxElementNode) isUnaryExpressionBase()        {}
func (*jsxElementNode) isUpdateExpressionBase()       {}
func (*jsxElementNode) isJsxElement()                 {}
func (*jsxElementNode) isBlockOrExpression()          {}
func (*jsxElementNode) isConciseBody()                {}
func (*jsxElementNode) isForInitializer()             {}
func (*jsxElementNode) isIncrementExpression()        {}
func (*jsxElementNode) isJsxAttributeValue()          {}
func (*jsxElementNode) isJsxChild()                   {}
func (*jsxElementNode) isNodeBody()                   {}

func (n *jsxElementNode) OpeningElement() JsxOpeningElement {
	return n.openingElement
}

func (n *jsxElementNode) Children() []JsxChild {
	return cloneSlice(n.children)
}

func (n *jsxElementNode) ClosingElement() JsxClosingElement {
	return n.closingElement
}

type JsxSelfClosingElement interface {
	PrimaryExpressionBase
	isJsxSelfClosingElement()
	isBlockOrExpression()
	isCallLikeExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isJsxAttributeValue()
	isJsxChild()
	isJsxOpeningLikeElement()
	isNodeBody()
	TagName() JsxTagNameExpression
	TypeArguments() []TypeNode
	Attributes() JsxAttributes
}

type jsxSelfClosingElementNode struct {
	nodeCore
	tagName       JsxTagNameExpression
	typeArguments []TypeNode
	attributes    JsxAttributes
}

func (*jsxSelfClosingElementNode) isExpressionBase()             {}
func (*jsxSelfClosingElementNode) isLeftHandSideExpressionBase() {}
func (*jsxSelfClosingElementNode) isMemberExpressionBase()       {}
func (*jsxSelfClosingElementNode) isNodeBase()                   {}
func (*jsxSelfClosingElementNode) isPrimaryExpressionBase()      {}
func (*jsxSelfClosingElementNode) isUnaryExpressionBase()        {}
func (*jsxSelfClosingElementNode) isUpdateExpressionBase()       {}
func (*jsxSelfClosingElementNode) isJsxSelfClosingElement()      {}
func (*jsxSelfClosingElementNode) isBlockOrExpression()          {}
func (*jsxSelfClosingElementNode) isCallLikeExpression()         {}
func (*jsxSelfClosingElementNode) isConciseBody()                {}
func (*jsxSelfClosingElementNode) isForInitializer()             {}
func (*jsxSelfClosingElementNode) isIncrementExpression()        {}
func (*jsxSelfClosingElementNode) isJsxAttributeValue()          {}
func (*jsxSelfClosingElementNode) isJsxChild()                   {}
func (*jsxSelfClosingElementNode) isJsxOpeningLikeElement()      {}
func (*jsxSelfClosingElementNode) isNodeBody()                   {}

func (n *jsxSelfClosingElementNode) TagName() JsxTagNameExpression {
	return n.tagName
}

func (n *jsxSelfClosingElementNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

func (n *jsxSelfClosingElementNode) Attributes() JsxAttributes {
	return n.attributes
}

type JsxOpeningElement interface {
	ExpressionBase
	isJsxOpeningElement()
	isBlockOrExpression()
	isCallLikeExpression()
	isConciseBody()
	isForInitializer()
	isJsxOpeningLikeElement()
	isNodeBody()
	TagName() JsxTagNameExpression
	TypeArguments() []TypeNode
	Attributes() JsxAttributes
}

type jsxOpeningElementNode struct {
	nodeCore
	tagName       JsxTagNameExpression
	typeArguments []TypeNode
	attributes    JsxAttributes
}

func (*jsxOpeningElementNode) isExpressionBase()        {}
func (*jsxOpeningElementNode) isNodeBase()              {}
func (*jsxOpeningElementNode) isJsxOpeningElement()     {}
func (*jsxOpeningElementNode) isBlockOrExpression()     {}
func (*jsxOpeningElementNode) isCallLikeExpression()    {}
func (*jsxOpeningElementNode) isConciseBody()           {}
func (*jsxOpeningElementNode) isForInitializer()        {}
func (*jsxOpeningElementNode) isJsxOpeningLikeElement() {}
func (*jsxOpeningElementNode) isNodeBody()              {}

func (n *jsxOpeningElementNode) TagName() JsxTagNameExpression {
	return n.tagName
}

func (n *jsxOpeningElementNode) TypeArguments() []TypeNode {
	return cloneSlice(n.typeArguments)
}

func (n *jsxOpeningElementNode) Attributes() JsxAttributes {
	return n.attributes
}

type JsxClosingElement interface {
	NodeBase
	isJsxClosingElement()
	TagName() JsxTagNameExpression
}

type jsxClosingElementNode struct {
	nodeCore
	tagName JsxTagNameExpression
}

func (*jsxClosingElementNode) isNodeBase()          {}
func (*jsxClosingElementNode) isJsxClosingElement() {}

func (n *jsxClosingElementNode) TagName() JsxTagNameExpression {
	return n.tagName
}

type JsxFragment interface {
	PrimaryExpressionBase
	isJsxFragment()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isJsxAttributeValue()
	isJsxChild()
	isNodeBody()
	OpeningFragment() JsxOpeningFragment
	Children() []JsxChild
	ClosingFragment() JsxClosingFragment
}

type jsxFragmentNode struct {
	nodeCore
	openingFragment JsxOpeningFragment
	children        []JsxChild
	closingFragment JsxClosingFragment
}

func (*jsxFragmentNode) isExpressionBase()             {}
func (*jsxFragmentNode) isLeftHandSideExpressionBase() {}
func (*jsxFragmentNode) isMemberExpressionBase()       {}
func (*jsxFragmentNode) isNodeBase()                   {}
func (*jsxFragmentNode) isPrimaryExpressionBase()      {}
func (*jsxFragmentNode) isUnaryExpressionBase()        {}
func (*jsxFragmentNode) isUpdateExpressionBase()       {}
func (*jsxFragmentNode) isJsxFragment()                {}
func (*jsxFragmentNode) isBlockOrExpression()          {}
func (*jsxFragmentNode) isConciseBody()                {}
func (*jsxFragmentNode) isForInitializer()             {}
func (*jsxFragmentNode) isIncrementExpression()        {}
func (*jsxFragmentNode) isJsxAttributeValue()          {}
func (*jsxFragmentNode) isJsxChild()                   {}
func (*jsxFragmentNode) isNodeBody()                   {}

func (n *jsxFragmentNode) OpeningFragment() JsxOpeningFragment {
	return n.openingFragment
}

func (n *jsxFragmentNode) Children() []JsxChild {
	return cloneSlice(n.children)
}

func (n *jsxFragmentNode) ClosingFragment() JsxClosingFragment {
	return n.closingFragment
}

type JsxOpeningFragment interface {
	ExpressionBase
	isJsxOpeningFragment()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
}

type jsxOpeningFragmentNode struct {
	nodeCore
}

func (*jsxOpeningFragmentNode) isExpressionBase()     {}
func (*jsxOpeningFragmentNode) isNodeBase()           {}
func (*jsxOpeningFragmentNode) isJsxOpeningFragment() {}
func (*jsxOpeningFragmentNode) isBlockOrExpression()  {}
func (*jsxOpeningFragmentNode) isConciseBody()        {}
func (*jsxOpeningFragmentNode) isForInitializer()     {}
func (*jsxOpeningFragmentNode) isNodeBody()           {}

type JsxClosingFragment interface {
	ExpressionBase
	isJsxClosingFragment()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
}

type jsxClosingFragmentNode struct {
	nodeCore
}

func (*jsxClosingFragmentNode) isExpressionBase()     {}
func (*jsxClosingFragmentNode) isNodeBase()           {}
func (*jsxClosingFragmentNode) isJsxClosingFragment() {}
func (*jsxClosingFragmentNode) isBlockOrExpression()  {}
func (*jsxClosingFragmentNode) isConciseBody()        {}
func (*jsxClosingFragmentNode) isForInitializer()     {}
func (*jsxClosingFragmentNode) isNodeBody()           {}

type JsxAttribute interface {
	NodeBase
	DeclarationBase
	isJsxAttribute()
	isJsxAttributeLike()
	Name() JsxAttributeName
	Initializer() JsxAttributeValue
}

type jsxAttributeNode struct {
	nodeCore
	name        JsxAttributeName
	initializer JsxAttributeValue
}

func (*jsxAttributeNode) isDeclarationBase()  {}
func (*jsxAttributeNode) isNodeBase()         {}
func (*jsxAttributeNode) isJsxAttribute()     {}
func (*jsxAttributeNode) isJsxAttributeLike() {}

func (n *jsxAttributeNode) Name() JsxAttributeName {
	return n.name
}

func (n *jsxAttributeNode) Initializer() JsxAttributeValue {
	return n.initializer
}

type JsxAttributes interface {
	PrimaryExpressionBase
	DeclarationBase
	isJsxAttributes()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Properties() []JsxAttributeLike
}

type jsxAttributesNode struct {
	nodeCore
	properties []JsxAttributeLike
}

func (*jsxAttributesNode) isDeclarationBase()            {}
func (*jsxAttributesNode) isExpressionBase()             {}
func (*jsxAttributesNode) isLeftHandSideExpressionBase() {}
func (*jsxAttributesNode) isMemberExpressionBase()       {}
func (*jsxAttributesNode) isNodeBase()                   {}
func (*jsxAttributesNode) isPrimaryExpressionBase()      {}
func (*jsxAttributesNode) isUnaryExpressionBase()        {}
func (*jsxAttributesNode) isUpdateExpressionBase()       {}
func (*jsxAttributesNode) isJsxAttributes()              {}
func (*jsxAttributesNode) isBlockOrExpression()          {}
func (*jsxAttributesNode) isConciseBody()                {}
func (*jsxAttributesNode) isForInitializer()             {}
func (*jsxAttributesNode) isIncrementExpression()        {}
func (*jsxAttributesNode) isNodeBody()                   {}

func (n *jsxAttributesNode) Properties() []JsxAttributeLike {
	return cloneSlice(n.properties)
}

type JsxSpreadAttribute interface {
	ObjectLiteralElementBase
	NodeBase
	isJsxSpreadAttribute()
	isJsxAttributeLike()
	Expression() Expression
}

type jsxSpreadAttributeNode struct {
	nodeCore
	expression Expression
}

func (*jsxSpreadAttributeNode) isNodeBase()                 {}
func (*jsxSpreadAttributeNode) isObjectLiteralElementBase() {}
func (*jsxSpreadAttributeNode) isJsxSpreadAttribute()       {}
func (*jsxSpreadAttributeNode) isJsxAttributeLike()         {}

func (n *jsxSpreadAttributeNode) Expression() Expression {
	return n.expression
}

type JsxExpression interface {
	ExpressionBase
	isJsxExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxAttributeValue()
	isJsxChild()
	isNodeBody()
	DotDotDotToken() DotDotDotToken
	Expression() Expression
}

type jsxExpressionNode struct {
	nodeCore
	dotDotDotToken DotDotDotToken
	expression     Expression
}

func (*jsxExpressionNode) isExpressionBase()    {}
func (*jsxExpressionNode) isNodeBase()          {}
func (*jsxExpressionNode) isJsxExpression()     {}
func (*jsxExpressionNode) isBlockOrExpression() {}
func (*jsxExpressionNode) isConciseBody()       {}
func (*jsxExpressionNode) isForInitializer()    {}
func (*jsxExpressionNode) isJsxAttributeValue() {}
func (*jsxExpressionNode) isJsxChild()          {}
func (*jsxExpressionNode) isNodeBody()          {}

func (n *jsxExpressionNode) DotDotDotToken() DotDotDotToken {
	return n.dotDotDotToken
}

func (n *jsxExpressionNode) Expression() Expression {
	return n.expression
}

type JsxNamespacedName interface {
	ExpressionBase
	isJsxNamespacedName()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isJsxAttributeName()
	isJsxTagNameExpression()
	isNodeBody()
	Namespace() Identifier
	Name() Identifier
}

type jsxNamespacedNameNode struct {
	nodeCore
	namespace Identifier
	name      Identifier
}

func (*jsxNamespacedNameNode) isExpressionBase()       {}
func (*jsxNamespacedNameNode) isNodeBase()             {}
func (*jsxNamespacedNameNode) isJsxNamespacedName()    {}
func (*jsxNamespacedNameNode) isBlockOrExpression()    {}
func (*jsxNamespacedNameNode) isConciseBody()          {}
func (*jsxNamespacedNameNode) isForInitializer()       {}
func (*jsxNamespacedNameNode) isJsxAttributeName()     {}
func (*jsxNamespacedNameNode) isJsxTagNameExpression() {}
func (*jsxNamespacedNameNode) isNodeBody()             {}

func (n *jsxNamespacedNameNode) Namespace() Identifier {
	return n.namespace
}

func (n *jsxNamespacedNameNode) Name() Identifier {
	return n.name
}

type CaseClause interface {
	CaseOrDefaultClause
	isCaseClause()
	Expression() Expression
	Statements() []Statement
}

type caseClauseNode struct {
	nodeCore
	expression Expression
	statements []Statement
}

func (*caseClauseNode) isNodeBase()            {}
func (*caseClauseNode) isCaseOrDefaultClause() {}
func (*caseClauseNode) isCaseClause()          {}

func (n *caseClauseNode) Expression() Expression {
	return n.expression
}

func (n *caseClauseNode) Statements() []Statement {
	return cloneSlice(n.statements)
}

type DefaultClause interface {
	CaseOrDefaultClause
	isDefaultClause()
	Expression() Expression
	Statements() []Statement
}

type defaultClauseNode struct {
	nodeCore
	expression Expression
	statements []Statement
}

func (*defaultClauseNode) isNodeBase()            {}
func (*defaultClauseNode) isCaseOrDefaultClause() {}
func (*defaultClauseNode) isDefaultClause()       {}

func (n *defaultClauseNode) Expression() Expression {
	return n.expression
}

func (n *defaultClauseNode) Statements() []Statement {
	return cloneSlice(n.statements)
}

type HeritageClause interface {
	NodeBase
	isHeritageClause()
	Token() HeritageClauseTokenKind
	Types() []ExpressionWithTypeArguments
}

type heritageClauseNode struct {
	nodeCore
	token HeritageClauseTokenKind
	types []ExpressionWithTypeArguments
}

func (*heritageClauseNode) isNodeBase()       {}
func (*heritageClauseNode) isHeritageClause() {}

func (n *heritageClauseNode) Token() HeritageClauseTokenKind {
	return n.token
}

func (n *heritageClauseNode) Types() []ExpressionWithTypeArguments {
	return cloneSlice(n.types)
}

type CatchClause interface {
	NodeBase
	isCatchClause()
	VariableDeclaration() VariableDeclaration
	Block() Block
}

type catchClauseNode struct {
	nodeCore
	variableDeclaration VariableDeclaration
	block               Block
}

func (*catchClauseNode) isNodeBase()    {}
func (*catchClauseNode) isCatchClause() {}

func (n *catchClauseNode) VariableDeclaration() VariableDeclaration {
	return n.variableDeclaration
}

func (n *catchClauseNode) Block() Block {
	return n.block
}

type ImportAttributes interface {
	NodeBase
	isImportAttributes()
	Token() ImportAttributesTokenKind
	Attributes() []ImportAttribute
	MultiLine() bool
}

type importAttributesNode struct {
	nodeCore
	token      ImportAttributesTokenKind
	attributes []ImportAttribute
	multiLine  bool
}

func (*importAttributesNode) isNodeBase()         {}
func (*importAttributesNode) isImportAttributes() {}

func (n *importAttributesNode) Token() ImportAttributesTokenKind {
	return n.token
}

func (n *importAttributesNode) Attributes() []ImportAttribute {
	return cloneSlice(n.attributes)
}

func (n *importAttributesNode) MultiLine() bool {
	return n.multiLine
}

type ImportAttribute interface {
	NodeBase
	isImportAttribute()
	Name() ImportAttributeName
	Value() Expression
}

type importAttributeNode struct {
	nodeCore
	name  ImportAttributeName
	value Expression
}

func (*importAttributeNode) isNodeBase()        {}
func (*importAttributeNode) isImportAttribute() {}

func (n *importAttributeNode) Name() ImportAttributeName {
	return n.name
}

func (n *importAttributeNode) Value() Expression {
	return n.value
}

type PropertyAssignment interface {
	NodeBase
	NamedMemberBase
	ObjectLiteralElementBase
	isPropertyAssignment()
	isObjectLiteralElementLike()
	Modifiers() []ModifierLike
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	Type() TypeNode
	Initializer() Expression
}

type propertyAssignmentNode struct {
	nodeCore
	modifiers    []ModifierLike
	name         PropertyName
	postfixToken NamedMemberBasePostfixToken
	typeNode     TypeNode
	initializer  Expression
}

func (*propertyAssignmentNode) isDeclarationBase()          {}
func (*propertyAssignmentNode) isModifiersBase()            {}
func (*propertyAssignmentNode) isNamedMemberBase()          {}
func (*propertyAssignmentNode) isNodeBase()                 {}
func (*propertyAssignmentNode) isObjectLiteralElementBase() {}
func (*propertyAssignmentNode) isPropertyAssignment()       {}
func (*propertyAssignmentNode) isObjectLiteralElementLike() {}

func (n *propertyAssignmentNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *propertyAssignmentNode) Name() PropertyName {
	return n.name
}

func (n *propertyAssignmentNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *propertyAssignmentNode) Type() TypeNode {
	return n.typeNode
}

func (n *propertyAssignmentNode) Initializer() Expression {
	return n.initializer
}

type ShorthandPropertyAssignment interface {
	NodeBase
	NamedMemberBase
	ObjectLiteralElementBase
	isShorthandPropertyAssignment()
	isObjectLiteralElementLike()
	Modifiers() []ModifierLike
	Name() PropertyName
	PostfixToken() NamedMemberBasePostfixToken
	Type() TypeNode
	EqualsToken() EqualsToken
	ObjectAssignmentInitializer() Expression
}

type shorthandPropertyAssignmentNode struct {
	nodeCore
	modifiers                   []ModifierLike
	name                        PropertyName
	postfixToken                NamedMemberBasePostfixToken
	typeNode                    TypeNode
	equalsToken                 EqualsToken
	objectAssignmentInitializer Expression
}

func (*shorthandPropertyAssignmentNode) isDeclarationBase()             {}
func (*shorthandPropertyAssignmentNode) isModifiersBase()               {}
func (*shorthandPropertyAssignmentNode) isNamedMemberBase()             {}
func (*shorthandPropertyAssignmentNode) isNodeBase()                    {}
func (*shorthandPropertyAssignmentNode) isObjectLiteralElementBase()    {}
func (*shorthandPropertyAssignmentNode) isShorthandPropertyAssignment() {}
func (*shorthandPropertyAssignmentNode) isObjectLiteralElementLike()    {}

func (n *shorthandPropertyAssignmentNode) Modifiers() []ModifierLike {
	return cloneSlice(n.modifiers)
}

func (n *shorthandPropertyAssignmentNode) Name() PropertyName {
	return n.name
}

func (n *shorthandPropertyAssignmentNode) PostfixToken() NamedMemberBasePostfixToken {
	return n.postfixToken
}

func (n *shorthandPropertyAssignmentNode) Type() TypeNode {
	return n.typeNode
}

func (n *shorthandPropertyAssignmentNode) EqualsToken() EqualsToken {
	return n.equalsToken
}

func (n *shorthandPropertyAssignmentNode) ObjectAssignmentInitializer() Expression {
	return n.objectAssignmentInitializer
}

type SpreadAssignment interface {
	NodeBase
	DeclarationBase
	ObjectLiteralElementBase
	isSpreadAssignment()
	isObjectLiteralElementLike()
	Expression() Expression
}

type spreadAssignmentNode struct {
	nodeCore
	expression Expression
}

func (*spreadAssignmentNode) isDeclarationBase()          {}
func (*spreadAssignmentNode) isNodeBase()                 {}
func (*spreadAssignmentNode) isObjectLiteralElementBase() {}
func (*spreadAssignmentNode) isSpreadAssignment()         {}
func (*spreadAssignmentNode) isObjectLiteralElementLike() {}

func (n *spreadAssignmentNode) Expression() Expression {
	return n.expression
}

type EnumMember interface {
	NodeBase
	NamedMemberBase
	isEnumMember()
	Name() PropertyName
	Initializer() Expression
}

type enumMemberNode struct {
	nodeCore
	name        PropertyName
	initializer Expression
}

func (*enumMemberNode) isDeclarationBase() {}
func (*enumMemberNode) isModifiersBase()   {}
func (*enumMemberNode) isNamedMemberBase() {}
func (*enumMemberNode) isNodeBase()        {}
func (*enumMemberNode) isEnumMember()      {}

func (n *enumMemberNode) Name() PropertyName {
	return n.name
}

func (n *enumMemberNode) Initializer() Expression {
	return n.initializer
}

type SourceFile interface {
	NodeBase
	DeclarationBase
	isSourceFile()
	Statements() []Statement
	EndOfFileToken() EndOfFile
	SourceData() SourceFileData
}

type sourceFileNode struct {
	nodeCore
	statements     []Statement
	endOfFileToken EndOfFile
	sourceData     SourceFileData
}

func (*sourceFileNode) isDeclarationBase() {}
func (*sourceFileNode) isNodeBase()        {}
func (*sourceFileNode) isSourceFile()      {}

func (n *sourceFileNode) Statements() []Statement {
	return cloneSlice(n.statements)
}

func (n *sourceFileNode) EndOfFileToken() EndOfFile {
	return n.endOfFileToken
}

func (n *sourceFileNode) SourceData() SourceFileData {
	return cloneSourceFileData(n.sourceData)
}

type JSDocTypeExpression interface {
	TypeNodeBase
	isJSDocTypeExpression()
	Type() TypeNode
}

type jsDocTypeExpressionNode struct {
	nodeCore
	typeNode TypeNode
}

func (*jsDocTypeExpressionNode) isNodeBase()            {}
func (*jsDocTypeExpressionNode) isTypeNodeBase()        {}
func (*jsDocTypeExpressionNode) isJSDocTypeExpression() {}

func (n *jsDocTypeExpressionNode) Type() TypeNode {
	return n.typeNode
}

type JSDocNameReference interface {
	TypeNodeBase
	isJSDocNameReference()
	Name() EntityName
}

type jsDocNameReferenceNode struct {
	nodeCore
	name EntityName
}

func (*jsDocNameReferenceNode) isNodeBase()           {}
func (*jsDocNameReferenceNode) isTypeNodeBase()       {}
func (*jsDocNameReferenceNode) isJSDocNameReference() {}

func (n *jsDocNameReferenceNode) Name() EntityName {
	return n.name
}

type JSDocAllType interface {
	JSDocTypeBase
	isJSDocAllType()
}

type jsDocAllTypeNode struct {
	nodeCore
}

func (*jsDocAllTypeNode) isJSDocTypeBase() {}
func (*jsDocAllTypeNode) isNodeBase()      {}
func (*jsDocAllTypeNode) isTypeNodeBase()  {}
func (*jsDocAllTypeNode) isJSDocAllType()  {}

type JSDocNullableType interface {
	JSDocTypeBase
	isJSDocNullableType()
	Type() TypeNode
}

type jsDocNullableTypeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*jsDocNullableTypeNode) isJSDocTypeBase()     {}
func (*jsDocNullableTypeNode) isNodeBase()          {}
func (*jsDocNullableTypeNode) isTypeNodeBase()      {}
func (*jsDocNullableTypeNode) isJSDocNullableType() {}

func (n *jsDocNullableTypeNode) Type() TypeNode {
	return n.typeNode
}

type JSDocNonNullableType interface {
	JSDocTypeBase
	isJSDocNonNullableType()
	Type() TypeNode
}

type jsDocNonNullableTypeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*jsDocNonNullableTypeNode) isJSDocTypeBase()        {}
func (*jsDocNonNullableTypeNode) isNodeBase()             {}
func (*jsDocNonNullableTypeNode) isTypeNodeBase()         {}
func (*jsDocNonNullableTypeNode) isJSDocNonNullableType() {}

func (n *jsDocNonNullableTypeNode) Type() TypeNode {
	return n.typeNode
}

type JSDocOptionalType interface {
	JSDocTypeBase
	isJSDocOptionalType()
	Type() TypeNode
}

type jsDocOptionalTypeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*jsDocOptionalTypeNode) isJSDocTypeBase()     {}
func (*jsDocOptionalTypeNode) isNodeBase()          {}
func (*jsDocOptionalTypeNode) isTypeNodeBase()      {}
func (*jsDocOptionalTypeNode) isJSDocOptionalType() {}

func (n *jsDocOptionalTypeNode) Type() TypeNode {
	return n.typeNode
}

type JSDocVariadicType interface {
	JSDocTypeBase
	isJSDocVariadicType()
	Type() TypeNode
}

type jsDocVariadicTypeNode struct {
	nodeCore
	typeNode TypeNode
}

func (*jsDocVariadicTypeNode) isJSDocTypeBase()     {}
func (*jsDocVariadicTypeNode) isNodeBase()          {}
func (*jsDocVariadicTypeNode) isTypeNodeBase()      {}
func (*jsDocVariadicTypeNode) isJSDocVariadicType() {}

func (n *jsDocVariadicTypeNode) Type() TypeNode {
	return n.typeNode
}

type JSDoc interface {
	NodeBase
	isJSDoc()
	Comment() []JSDocComment
	Tags() []JSDocTag
}

type jsDocNode struct {
	nodeCore
	comment []JSDocComment
	tags    []JSDocTag
}

func (*jsDocNode) isNodeBase() {}
func (*jsDocNode) isJSDoc()    {}

func (n *jsDocNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

func (n *jsDocNode) Tags() []JSDocTag {
	return cloneSlice(n.tags)
}

type JSDocText interface {
	JSDocCommentBase
	isJSDocText()
	isJSDocComment()
	Text() []string
}

type jsDocTextNode struct {
	nodeCore
	text []string
}

func (*jsDocTextNode) isJSDocCommentBase() {}
func (*jsDocTextNode) isNodeBase()         {}
func (*jsDocTextNode) isJSDocText()        {}
func (*jsDocTextNode) isJSDocComment()     {}

func (n *jsDocTextNode) Text() []string {
	return cloneSlice(n.text)
}

type JSDocTypeLiteral interface {
	JSDocTypeBase
	DeclarationBase
	isJSDocTypeLiteral()
	JSDocPropertyTags() []JSDocTag
	IsArrayType() bool
}

type jsDocTypeLiteralNode struct {
	nodeCore
	jSDocPropertyTags []JSDocTag
	isArrayType       bool
}

func (*jsDocTypeLiteralNode) isDeclarationBase()  {}
func (*jsDocTypeLiteralNode) isJSDocTypeBase()    {}
func (*jsDocTypeLiteralNode) isNodeBase()         {}
func (*jsDocTypeLiteralNode) isTypeNodeBase()     {}
func (*jsDocTypeLiteralNode) isJSDocTypeLiteral() {}

func (n *jsDocTypeLiteralNode) JSDocPropertyTags() []JSDocTag {
	return cloneSlice(n.jSDocPropertyTags)
}

func (n *jsDocTypeLiteralNode) IsArrayType() bool {
	return n.isArrayType
}

type JSDocSignature interface {
	JSDocTypeBase
	FunctionLikeBase
	isJSDocSignature()
	TypeParameters() []TypeParameterDeclaration
	Parameters() []ParameterDeclaration
	Type() TypeNode
}

type jsDocSignatureNode struct {
	nodeCore
	typeParameters []TypeParameterDeclaration
	parameters     []ParameterDeclaration
	typeNode       TypeNode
}

func (*jsDocSignatureNode) isDeclarationBase()  {}
func (*jsDocSignatureNode) isFunctionLikeBase() {}
func (*jsDocSignatureNode) isJSDocTypeBase()    {}
func (*jsDocSignatureNode) isNodeBase()         {}
func (*jsDocSignatureNode) isTypeNodeBase()     {}
func (*jsDocSignatureNode) isJSDocSignature()   {}

func (n *jsDocSignatureNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *jsDocSignatureNode) Parameters() []ParameterDeclaration {
	return cloneSlice(n.parameters)
}

func (n *jsDocSignatureNode) Type() TypeNode {
	return n.typeNode
}

type JSDocLink interface {
	JSDocCommentBase
	isJSDocLink()
	isJSDocComment()
	Name() EntityName
	Text() []string
}

type jsDocLinkNode struct {
	nodeCore
	name EntityName
	text []string
}

func (*jsDocLinkNode) isJSDocCommentBase() {}
func (*jsDocLinkNode) isNodeBase()         {}
func (*jsDocLinkNode) isJSDocLink()        {}
func (*jsDocLinkNode) isJSDocComment()     {}

func (n *jsDocLinkNode) Name() EntityName {
	return n.name
}

func (n *jsDocLinkNode) Text() []string {
	return cloneSlice(n.text)
}

type JSDocLinkCode interface {
	JSDocCommentBase
	isJSDocLinkCode()
	isJSDocComment()
	Name() EntityName
	Text() []string
}

type jsDocLinkCodeNode struct {
	nodeCore
	name EntityName
	text []string
}

func (*jsDocLinkCodeNode) isJSDocCommentBase() {}
func (*jsDocLinkCodeNode) isNodeBase()         {}
func (*jsDocLinkCodeNode) isJSDocLinkCode()    {}
func (*jsDocLinkCodeNode) isJSDocComment()     {}

func (n *jsDocLinkCodeNode) Name() EntityName {
	return n.name
}

func (n *jsDocLinkCodeNode) Text() []string {
	return cloneSlice(n.text)
}

type JSDocLinkPlain interface {
	JSDocCommentBase
	isJSDocLinkPlain()
	isJSDocComment()
	Name() EntityName
	Text() []string
}

type jsDocLinkPlainNode struct {
	nodeCore
	name EntityName
	text []string
}

func (*jsDocLinkPlainNode) isJSDocCommentBase() {}
func (*jsDocLinkPlainNode) isNodeBase()         {}
func (*jsDocLinkPlainNode) isJSDocLinkPlain()   {}
func (*jsDocLinkPlainNode) isJSDocComment()     {}

func (n *jsDocLinkPlainNode) Name() EntityName {
	return n.name
}

func (n *jsDocLinkPlainNode) Text() []string {
	return cloneSlice(n.text)
}

type JSDocUnknownTag interface {
	JSDocTagBase
	isJSDocUnknownTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocUnknownTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocUnknownTagNode) isJSDocTagBase()    {}
func (*jsDocUnknownTagNode) isNodeBase()        {}
func (*jsDocUnknownTagNode) isJSDocUnknownTag() {}

func (n *jsDocUnknownTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocUnknownTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocAugmentsTag interface {
	JSDocTagBase
	isJSDocAugmentsTag()
	TagName() Identifier
	ClassName() ExpressionWithTypeArguments
	Comment() []JSDocComment
}

type jsDocAugmentsTagNode struct {
	nodeCore
	tagName   Identifier
	className ExpressionWithTypeArguments
	comment   []JSDocComment
}

func (*jsDocAugmentsTagNode) isJSDocTagBase()     {}
func (*jsDocAugmentsTagNode) isNodeBase()         {}
func (*jsDocAugmentsTagNode) isJSDocAugmentsTag() {}

func (n *jsDocAugmentsTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocAugmentsTagNode) ClassName() ExpressionWithTypeArguments {
	return n.className
}

func (n *jsDocAugmentsTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocImplementsTag interface {
	JSDocTagBase
	isJSDocImplementsTag()
	TagName() Identifier
	ClassName() ExpressionWithTypeArguments
	Comment() []JSDocComment
}

type jsDocImplementsTagNode struct {
	nodeCore
	tagName   Identifier
	className ExpressionWithTypeArguments
	comment   []JSDocComment
}

func (*jsDocImplementsTagNode) isJSDocTagBase()       {}
func (*jsDocImplementsTagNode) isNodeBase()           {}
func (*jsDocImplementsTagNode) isJSDocImplementsTag() {}

func (n *jsDocImplementsTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocImplementsTagNode) ClassName() ExpressionWithTypeArguments {
	return n.className
}

func (n *jsDocImplementsTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocDeprecatedTag interface {
	JSDocTagBase
	isJSDocDeprecatedTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocDeprecatedTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocDeprecatedTagNode) isJSDocTagBase()       {}
func (*jsDocDeprecatedTagNode) isNodeBase()           {}
func (*jsDocDeprecatedTagNode) isJSDocDeprecatedTag() {}

func (n *jsDocDeprecatedTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocDeprecatedTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocPublicTag interface {
	JSDocTagBase
	isJSDocPublicTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocPublicTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocPublicTagNode) isJSDocTagBase()   {}
func (*jsDocPublicTagNode) isNodeBase()       {}
func (*jsDocPublicTagNode) isJSDocPublicTag() {}

func (n *jsDocPublicTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocPublicTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocPrivateTag interface {
	JSDocTagBase
	isJSDocPrivateTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocPrivateTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocPrivateTagNode) isJSDocTagBase()    {}
func (*jsDocPrivateTagNode) isNodeBase()        {}
func (*jsDocPrivateTagNode) isJSDocPrivateTag() {}

func (n *jsDocPrivateTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocPrivateTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocProtectedTag interface {
	JSDocTagBase
	isJSDocProtectedTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocProtectedTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocProtectedTagNode) isJSDocTagBase()      {}
func (*jsDocProtectedTagNode) isNodeBase()          {}
func (*jsDocProtectedTagNode) isJSDocProtectedTag() {}

func (n *jsDocProtectedTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocProtectedTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocReadonlyTag interface {
	JSDocTagBase
	isJSDocReadonlyTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocReadonlyTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocReadonlyTagNode) isJSDocTagBase()     {}
func (*jsDocReadonlyTagNode) isNodeBase()         {}
func (*jsDocReadonlyTagNode) isJSDocReadonlyTag() {}

func (n *jsDocReadonlyTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocReadonlyTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocOverrideTag interface {
	JSDocTagBase
	isJSDocOverrideTag()
	TagName() Identifier
	Comment() []JSDocComment
}

type jsDocOverrideTagNode struct {
	nodeCore
	tagName Identifier
	comment []JSDocComment
}

func (*jsDocOverrideTagNode) isJSDocTagBase()     {}
func (*jsDocOverrideTagNode) isNodeBase()         {}
func (*jsDocOverrideTagNode) isJSDocOverrideTag() {}

func (n *jsDocOverrideTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocOverrideTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocCallbackTag interface {
	JSDocTagBase
	isJSDocCallbackTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Name() JSDocFullName
	Comment() []JSDocComment
}

type jsDocCallbackTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	name           JSDocFullName
	comment        []JSDocComment
}

func (*jsDocCallbackTagNode) isJSDocTagBase()     {}
func (*jsDocCallbackTagNode) isNodeBase()         {}
func (*jsDocCallbackTagNode) isJSDocCallbackTag() {}

func (n *jsDocCallbackTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocCallbackTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocCallbackTagNode) Name() JSDocFullName {
	return n.name
}

func (n *jsDocCallbackTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocOverloadTag interface {
	JSDocTagBase
	isJSDocOverloadTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocOverloadTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocOverloadTagNode) isJSDocTagBase()     {}
func (*jsDocOverloadTagNode) isNodeBase()         {}
func (*jsDocOverloadTagNode) isJSDocOverloadTag() {}

func (n *jsDocOverloadTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocOverloadTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocOverloadTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocParameterTag interface {
	JSDocParameterOrPropertyTag
	isJSDocParameterTag()
	TagName() Identifier
	Name() EntityName
	IsBracketed() bool
	TypeExpression() TypeNode
	IsNameFirst() bool
	Comment() []JSDocComment
}

type jsDocParameterTagNode struct {
	nodeCore
	tagName        Identifier
	name           EntityName
	isBracketed    bool
	typeExpression TypeNode
	isNameFirst    bool
	comment        []JSDocComment
}

func (*jsDocParameterTagNode) isJSDocTagBase()                {}
func (*jsDocParameterTagNode) isNodeBase()                    {}
func (*jsDocParameterTagNode) isJSDocParameterOrPropertyTag() {}
func (*jsDocParameterTagNode) isJSDocParameterTag()           {}

func (n *jsDocParameterTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocParameterTagNode) Name() EntityName {
	return n.name
}

func (n *jsDocParameterTagNode) IsBracketed() bool {
	return n.isBracketed
}

func (n *jsDocParameterTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocParameterTagNode) IsNameFirst() bool {
	return n.isNameFirst
}

func (n *jsDocParameterTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocReturnTag interface {
	JSDocTagBase
	isJSDocReturnTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocReturnTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocReturnTagNode) isJSDocTagBase()   {}
func (*jsDocReturnTagNode) isNodeBase()       {}
func (*jsDocReturnTagNode) isJSDocReturnTag() {}

func (n *jsDocReturnTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocReturnTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocReturnTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocThisTag interface {
	JSDocTagBase
	isJSDocThisTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocThisTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocThisTagNode) isJSDocTagBase() {}
func (*jsDocThisTagNode) isNodeBase()     {}
func (*jsDocThisTagNode) isJSDocThisTag() {}

func (n *jsDocThisTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocThisTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocThisTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocTypeTag interface {
	JSDocTagBase
	isJSDocTypeTag()
	TagName() Identifier
	TypeExpression() Node
	Comment() []JSDocComment
}

type jsDocTypeTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression Node
	comment        []JSDocComment
}

func (*jsDocTypeTagNode) isJSDocTagBase() {}
func (*jsDocTypeTagNode) isNodeBase()     {}
func (*jsDocTypeTagNode) isJSDocTypeTag() {}

func (n *jsDocTypeTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocTypeTagNode) TypeExpression() Node {
	return n.typeExpression
}

func (n *jsDocTypeTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocTemplateTag interface {
	JSDocTagBase
	isJSDocTemplateTag()
	TagName() Identifier
	Constraint() Node
	TypeParameters() []TypeParameterDeclaration
	Comment() []JSDocComment
}

type jsDocTemplateTagNode struct {
	nodeCore
	tagName        Identifier
	constraint     Node
	typeParameters []TypeParameterDeclaration
	comment        []JSDocComment
}

func (*jsDocTemplateTagNode) isJSDocTagBase()     {}
func (*jsDocTemplateTagNode) isNodeBase()         {}
func (*jsDocTemplateTagNode) isJSDocTemplateTag() {}

func (n *jsDocTemplateTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocTemplateTagNode) Constraint() Node {
	return n.constraint
}

func (n *jsDocTemplateTagNode) TypeParameters() []TypeParameterDeclaration {
	return cloneSlice(n.typeParameters)
}

func (n *jsDocTemplateTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocTypedefTag interface {
	JSDocTagBase
	isJSDocTypedefTag()
	TagName() Identifier
	TypeExpression() Node
	Name() JSDocFullName
	Comment() []JSDocComment
}

type jsDocTypedefTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression Node
	name           JSDocFullName
	comment        []JSDocComment
}

func (*jsDocTypedefTagNode) isJSDocTagBase()    {}
func (*jsDocTypedefTagNode) isNodeBase()        {}
func (*jsDocTypedefTagNode) isJSDocTypedefTag() {}

func (n *jsDocTypedefTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocTypedefTagNode) TypeExpression() Node {
	return n.typeExpression
}

func (n *jsDocTypedefTagNode) Name() JSDocFullName {
	return n.name
}

func (n *jsDocTypedefTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocSeeTag interface {
	JSDocTagBase
	isJSDocSeeTag()
	TagName() Identifier
	NameExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocSeeTagNode struct {
	nodeCore
	tagName        Identifier
	nameExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocSeeTagNode) isJSDocTagBase() {}
func (*jsDocSeeTagNode) isNodeBase()     {}
func (*jsDocSeeTagNode) isJSDocSeeTag()  {}

func (n *jsDocSeeTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocSeeTagNode) NameExpression() TypeNode {
	return n.nameExpression
}

func (n *jsDocSeeTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocPropertyTag interface {
	JSDocParameterOrPropertyTag
	isJSDocPropertyTag()
	TagName() Identifier
	Name() EntityName
	IsBracketed() bool
	TypeExpression() TypeNode
	IsNameFirst() bool
	Comment() []JSDocComment
}

type jsDocPropertyTagNode struct {
	nodeCore
	tagName        Identifier
	name           EntityName
	isBracketed    bool
	typeExpression TypeNode
	isNameFirst    bool
	comment        []JSDocComment
}

func (*jsDocPropertyTagNode) isJSDocTagBase()                {}
func (*jsDocPropertyTagNode) isNodeBase()                    {}
func (*jsDocPropertyTagNode) isJSDocParameterOrPropertyTag() {}
func (*jsDocPropertyTagNode) isJSDocPropertyTag()            {}

func (n *jsDocPropertyTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocPropertyTagNode) Name() EntityName {
	return n.name
}

func (n *jsDocPropertyTagNode) IsBracketed() bool {
	return n.isBracketed
}

func (n *jsDocPropertyTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocPropertyTagNode) IsNameFirst() bool {
	return n.isNameFirst
}

func (n *jsDocPropertyTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocThrowsTag interface {
	JSDocTagBase
	isJSDocThrowsTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocThrowsTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocThrowsTagNode) isJSDocTagBase()   {}
func (*jsDocThrowsTagNode) isNodeBase()       {}
func (*jsDocThrowsTagNode) isJSDocThrowsTag() {}

func (n *jsDocThrowsTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocThrowsTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocThrowsTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocSatisfiesTag interface {
	JSDocTagBase
	isJSDocSatisfiesTag()
	TagName() Identifier
	TypeExpression() TypeNode
	Comment() []JSDocComment
}

type jsDocSatisfiesTagNode struct {
	nodeCore
	tagName        Identifier
	typeExpression TypeNode
	comment        []JSDocComment
}

func (*jsDocSatisfiesTagNode) isJSDocTagBase()      {}
func (*jsDocSatisfiesTagNode) isNodeBase()          {}
func (*jsDocSatisfiesTagNode) isJSDocSatisfiesTag() {}

func (n *jsDocSatisfiesTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocSatisfiesTagNode) TypeExpression() TypeNode {
	return n.typeExpression
}

func (n *jsDocSatisfiesTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type JSDocImportTag interface {
	JSDocTagBase
	isJSDocImportTag()
	TagName() Identifier
	ImportClause() ImportClause
	ModuleSpecifier() Expression
	Attributes() ImportAttributes
	Comment() []JSDocComment
}

type jsDocImportTagNode struct {
	nodeCore
	tagName         Identifier
	importClause    ImportClause
	moduleSpecifier Expression
	attributes      ImportAttributes
	comment         []JSDocComment
}

func (*jsDocImportTagNode) isJSDocTagBase()   {}
func (*jsDocImportTagNode) isNodeBase()       {}
func (*jsDocImportTagNode) isJSDocImportTag() {}

func (n *jsDocImportTagNode) TagName() Identifier {
	return n.tagName
}

func (n *jsDocImportTagNode) ImportClause() ImportClause {
	return n.importClause
}

func (n *jsDocImportTagNode) ModuleSpecifier() Expression {
	return n.moduleSpecifier
}

func (n *jsDocImportTagNode) Attributes() ImportAttributes {
	return n.attributes
}

func (n *jsDocImportTagNode) Comment() []JSDocComment {
	return cloneSlice(n.comment)
}

type SyntaxList interface {
	NodeBase
	isSyntaxList()
	Children() []Node
}

type syntaxListNode struct {
	nodeCore
	children []Node
}

func (*syntaxListNode) isNodeBase()   {}
func (*syntaxListNode) isSyntaxList() {}

func (n *syntaxListNode) Children() []Node {
	return cloneSlice(n.children)
}

type NotEmittedStatement interface {
	StatementBase
	isNotEmittedStatement()
}

type notEmittedStatementNode struct {
	nodeCore
}

func (*notEmittedStatementNode) isNodeBase()            {}
func (*notEmittedStatementNode) isStatementBase()       {}
func (*notEmittedStatementNode) isNotEmittedStatement() {}

type PartiallyEmittedExpression interface {
	LeftHandSideExpressionBase
	isPartiallyEmittedExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isIncrementExpression()
	isNodeBody()
	Expression() Expression
}

type partiallyEmittedExpressionNode struct {
	nodeCore
	expression Expression
}

func (*partiallyEmittedExpressionNode) isExpressionBase()             {}
func (*partiallyEmittedExpressionNode) isLeftHandSideExpressionBase() {}
func (*partiallyEmittedExpressionNode) isNodeBase()                   {}
func (*partiallyEmittedExpressionNode) isUnaryExpressionBase()        {}
func (*partiallyEmittedExpressionNode) isUpdateExpressionBase()       {}
func (*partiallyEmittedExpressionNode) isPartiallyEmittedExpression() {}
func (*partiallyEmittedExpressionNode) isBlockOrExpression()          {}
func (*partiallyEmittedExpressionNode) isConciseBody()                {}
func (*partiallyEmittedExpressionNode) isForInitializer()             {}
func (*partiallyEmittedExpressionNode) isIncrementExpression()        {}
func (*partiallyEmittedExpressionNode) isNodeBody()                   {}

func (n *partiallyEmittedExpressionNode) Expression() Expression {
	return n.expression
}

type SyntheticReferenceExpression interface {
	ExpressionBase
	isSyntheticReferenceExpression()
	isBlockOrExpression()
	isConciseBody()
	isForInitializer()
	isNodeBody()
	Expression() Expression
	ThisArg() Expression
}

type syntheticReferenceExpressionNode struct {
	nodeCore
	expression Expression
	thisArg    Expression
}

func (*syntheticReferenceExpressionNode) isExpressionBase()               {}
func (*syntheticReferenceExpressionNode) isNodeBase()                     {}
func (*syntheticReferenceExpressionNode) isSyntheticReferenceExpression() {}
func (*syntheticReferenceExpressionNode) isBlockOrExpression()            {}
func (*syntheticReferenceExpressionNode) isConciseBody()                  {}
func (*syntheticReferenceExpressionNode) isForInitializer()               {}
func (*syntheticReferenceExpressionNode) isNodeBody()                     {}

func (n *syntheticReferenceExpressionNode) Expression() Expression {
	return n.expression
}

func (n *syntheticReferenceExpressionNode) ThisArg() Expression {
	return n.thisArg
}

type NotEmittedTypeElement interface {
	NodeBase
	TypeElementBase
	isNotEmittedTypeElement()
}

type notEmittedTypeElementNode struct {
	nodeCore
}

func (*notEmittedTypeElementNode) isNodeBase()              {}
func (*notEmittedTypeElementNode) isTypeElementBase()       {}
func (*notEmittedTypeElementNode) isNotEmittedTypeElement() {}
