package catalog

// Explicit, permanent edge identities, pinned in per-kind visit order.
// Do not renumber; append only.
const (
	EdgeInvalid Edge = 0

	EdgeEllipsisElt          Edge = 1
	EdgeFuncLitType          Edge = 2
	EdgeFuncLitBody          Edge = 3
	EdgeCompositeLitType     Edge = 4
	EdgeCompositeLitElts     Edge = 5
	EdgeParenExprX           Edge = 6
	EdgeSelectorExprX        Edge = 7
	EdgeSelectorExprSel      Edge = 8
	EdgeIndexExprX           Edge = 9
	EdgeIndexExprIndex       Edge = 10
	EdgeIndexListExprX       Edge = 11
	EdgeIndexListExprIndices Edge = 12
	EdgeSliceExprX           Edge = 13
	EdgeSliceExprLow         Edge = 14
	EdgeSliceExprHigh        Edge = 15
	EdgeSliceExprMax         Edge = 16
	EdgeTypeAssertExprX      Edge = 17
	EdgeTypeAssertExprType   Edge = 18
	EdgeCallExprFun          Edge = 19
	EdgeCallExprArgs         Edge = 20
	EdgeStarExprX            Edge = 21
	EdgeUnaryExprX           Edge = 22
	EdgeBinaryExprX          Edge = 23
	EdgeBinaryExprY          Edge = 24
	EdgeKeyValueExprKey      Edge = 25
	EdgeKeyValueExprValue    Edge = 26
	EdgeArrayTypeLen         Edge = 27
	EdgeArrayTypeElt         Edge = 28
	EdgeStructTypeFields     Edge = 29
	EdgeFuncTypeTypeParams   Edge = 30
	EdgeFuncTypeParams       Edge = 31
	EdgeFuncTypeResults      Edge = 32
	EdgeInterfaceTypeMethods Edge = 33
	EdgeMapTypeKey           Edge = 34
	EdgeMapTypeValue         Edge = 35
	EdgeChanTypeValue        Edge = 36
	EdgeDeclStmtDecl         Edge = 37
	EdgeLabeledStmtLabel     Edge = 38
	EdgeLabeledStmtStmt      Edge = 39
	EdgeExprStmtX            Edge = 40
	EdgeSendStmtChan         Edge = 41
	EdgeSendStmtValue        Edge = 42
	EdgeIncDecStmtX          Edge = 43
	EdgeAssignStmtLhs        Edge = 44
	EdgeAssignStmtRhs        Edge = 45
	EdgeGoStmtCall           Edge = 46
	EdgeDeferStmtCall        Edge = 47
	EdgeReturnStmtResults    Edge = 48
	EdgeBranchStmtLabel      Edge = 49
	EdgeBlockStmtList        Edge = 50
	EdgeIfStmtInit           Edge = 51
	EdgeIfStmtCond           Edge = 52
	EdgeIfStmtBody           Edge = 53
	EdgeIfStmtElse           Edge = 54
	EdgeCaseClauseList       Edge = 55
	EdgeCaseClauseBody       Edge = 56
	EdgeSwitchStmtInit       Edge = 57
	EdgeSwitchStmtTag        Edge = 58
	EdgeSwitchStmtBody       Edge = 59
	EdgeTypeSwitchStmtInit   Edge = 60
	EdgeTypeSwitchStmtAssign Edge = 61
	EdgeTypeSwitchStmtBody   Edge = 62
	EdgeCommClauseComm       Edge = 63
	EdgeCommClauseBody       Edge = 64
	EdgeSelectStmtBody       Edge = 65
	EdgeForStmtInit          Edge = 66
	EdgeForStmtCond          Edge = 67
	EdgeForStmtPost          Edge = 68
	EdgeForStmtBody          Edge = 69
	EdgeRangeStmtKey         Edge = 70
	EdgeRangeStmtValue       Edge = 71
	EdgeRangeStmtX           Edge = 72
	EdgeRangeStmtBody        Edge = 73
	EdgeImportSpecDoc        Edge = 74
	EdgeImportSpecName       Edge = 75
	EdgeImportSpecPath       Edge = 76
	EdgeImportSpecComment    Edge = 77
	EdgeValueSpecDoc         Edge = 78
	EdgeValueSpecNames       Edge = 79
	EdgeValueSpecType        Edge = 80
	EdgeValueSpecValues      Edge = 81
	EdgeValueSpecComment     Edge = 82
	EdgeTypeSpecDoc          Edge = 83
	EdgeTypeSpecName         Edge = 84
	EdgeTypeSpecTypeParams   Edge = 85
	EdgeTypeSpecType         Edge = 86
	EdgeTypeSpecComment      Edge = 87
	EdgeGenDeclDoc           Edge = 88
	EdgeGenDeclSpecs         Edge = 89
	EdgeFuncDeclDoc          Edge = 90
	EdgeFuncDeclRecv         Edge = 91
	EdgeFuncDeclName         Edge = 92
	EdgeFuncDeclType         Edge = 93
	EdgeFuncDeclBody         Edge = 94
	EdgeFileDoc              Edge = 95
	EdgeFileName             Edge = 96
	EdgeFileDecls            Edge = 97
	EdgeCommentGroupList     Edge = 98
	EdgeFieldDoc             Edge = 99
	EdgeFieldNames           Edge = 100
	EdgeFieldType            Edge = 101
	EdgeFieldTag             Edge = 102
	EdgeFieldComment         Edge = 103
	EdgeFieldListList        Edge = 104

	// edgeCount is the highest assigned identity; append-only.
	edgeCount = 104
)

// edges is the exact-size descriptor table indexed by Edge.
var edges = [edgeCount + 1]edgeDescriptor{
	EdgeEllipsisElt:          {"Ellipsis.Elt", KindEllipsis, "Elt", RoleTypeExpression, false},
	EdgeFuncLitType:          {"FuncLit.Type", KindFuncLit, "Type", RoleFunctionSignature, false},
	EdgeFuncLitBody:          {"FuncLit.Body", KindFuncLit, "Body", RoleFunctionBody, false},
	EdgeCompositeLitType:     {"CompositeLit.Type", KindCompositeLit, "Type", RoleConstructedType, false},
	EdgeCompositeLitElts:     {"CompositeLit.Elts", KindCompositeLit, "Elts", RoleCompositeElement, true},
	EdgeParenExprX:           {"ParenExpr.X", KindParenExpr, "X", RoleOperand, false},
	EdgeSelectorExprX:        {"SelectorExpr.X", KindSelectorExpr, "X", RoleSelectorBase, false},
	EdgeSelectorExprSel:      {"SelectorExpr.Sel", KindSelectorExpr, "Sel", RoleSelectedName, false},
	EdgeIndexExprX:           {"IndexExpr.X", KindIndexExpr, "X", RoleIndexedOperand, false},
	EdgeIndexExprIndex:       {"IndexExpr.Index", KindIndexExpr, "Index", RoleIndex, false},
	EdgeIndexListExprX:       {"IndexListExpr.X", KindIndexListExpr, "X", RoleIndexedOperand, false},
	EdgeIndexListExprIndices: {"IndexListExpr.Indices", KindIndexListExpr, "Indices", RoleIndex, true},
	EdgeSliceExprX:           {"SliceExpr.X", KindSliceExpr, "X", RoleSlicedOperand, false},
	EdgeSliceExprLow:         {"SliceExpr.Low", KindSliceExpr, "Low", RoleSliceBound, false},
	EdgeSliceExprHigh:        {"SliceExpr.High", KindSliceExpr, "High", RoleSliceBound, false},
	EdgeSliceExprMax:         {"SliceExpr.Max", KindSliceExpr, "Max", RoleSliceBound, false},
	EdgeTypeAssertExprX:      {"TypeAssertExpr.X", KindTypeAssertExpr, "X", RoleAssertedValue, false},
	EdgeTypeAssertExprType:   {"TypeAssertExpr.Type", KindTypeAssertExpr, "Type", RoleAssertedType, false},
	EdgeCallExprFun:          {"CallExpr.Fun", KindCallExpr, "Fun", RoleCallee, false},
	EdgeCallExprArgs:         {"CallExpr.Args", KindCallExpr, "Args", RoleCallArgument, true},
	EdgeStarExprX:            {"StarExpr.X", KindStarExpr, "X", RoleOperand, false},
	EdgeUnaryExprX:           {"UnaryExpr.X", KindUnaryExpr, "X", RoleOperand, false},
	EdgeBinaryExprX:          {"BinaryExpr.X", KindBinaryExpr, "X", RoleLeftOperand, false},
	EdgeBinaryExprY:          {"BinaryExpr.Y", KindBinaryExpr, "Y", RoleRightOperand, false},
	EdgeKeyValueExprKey:      {"KeyValueExpr.Key", KindKeyValueExpr, "Key", RoleElementKey, false},
	EdgeKeyValueExprValue:    {"KeyValueExpr.Value", KindKeyValueExpr, "Value", RoleElementValue, false},
	EdgeArrayTypeLen:         {"ArrayType.Len", KindArrayType, "Len", RoleArrayLength, false},
	EdgeArrayTypeElt:         {"ArrayType.Elt", KindArrayType, "Elt", RoleElementType, false},
	EdgeStructTypeFields:     {"StructType.Fields", KindStructType, "Fields", RoleStructFields, false},
	EdgeFuncTypeTypeParams:   {"FuncType.TypeParams", KindFuncType, "TypeParams", RoleTypeParameters, false},
	EdgeFuncTypeParams:       {"FuncType.Params", KindFuncType, "Params", RoleParameters, false},
	EdgeFuncTypeResults:      {"FuncType.Results", KindFuncType, "Results", RoleResults, false},
	EdgeInterfaceTypeMethods: {"InterfaceType.Methods", KindInterfaceType, "Methods", RoleInterfaceMethods, false},
	EdgeMapTypeKey:           {"MapType.Key", KindMapType, "Key", RoleKeyType, false},
	EdgeMapTypeValue:         {"MapType.Value", KindMapType, "Value", RoleValueType, false},
	EdgeChanTypeValue:        {"ChanType.Value", KindChanType, "Value", RoleElementType, false},
	EdgeDeclStmtDecl:         {"DeclStmt.Decl", KindDeclStmt, "Decl", RoleDeclaration, false},
	EdgeLabeledStmtLabel:     {"LabeledStmt.Label", KindLabeledStmt, "Label", RoleLabelDeclaration, false},
	EdgeLabeledStmtStmt:      {"LabeledStmt.Stmt", KindLabeledStmt, "Stmt", RoleLabeledStatement, false},
	EdgeExprStmtX:            {"ExprStmt.X", KindExprStmt, "X", RoleStatementExpression, false},
	EdgeSendStmtChan:         {"SendStmt.Chan", KindSendStmt, "Chan", RoleChannelOperand, false},
	EdgeSendStmtValue:        {"SendStmt.Value", KindSendStmt, "Value", RoleSentValue, false},
	EdgeIncDecStmtX:          {"IncDecStmt.X", KindIncDecStmt, "X", RoleAssignablePlace, false},
	EdgeAssignStmtLhs:        {"AssignStmt.Lhs", KindAssignStmt, "Lhs", RoleAssignmentTarget, true},
	EdgeAssignStmtRhs:        {"AssignStmt.Rhs", KindAssignStmt, "Rhs", RoleAssignedValue, true},
	EdgeGoStmtCall:           {"GoStmt.Call", KindGoStmt, "Call", RoleSpawnedCall, false},
	EdgeDeferStmtCall:        {"DeferStmt.Call", KindDeferStmt, "Call", RoleDeferredCall, false},
	EdgeReturnStmtResults:    {"ReturnStmt.Results", KindReturnStmt, "Results", RoleReturnValue, true},
	EdgeBranchStmtLabel:      {"BranchStmt.Label", KindBranchStmt, "Label", RoleLabelReference, false},
	EdgeBlockStmtList:        {"BlockStmt.List", KindBlockStmt, "List", RoleStatement, true},
	EdgeIfStmtInit:           {"IfStmt.Init", KindIfStmt, "Init", RoleInitStatement, false},
	EdgeIfStmtCond:           {"IfStmt.Cond", KindIfStmt, "Cond", RoleCondition, false},
	EdgeIfStmtBody:           {"IfStmt.Body", KindIfStmt, "Body", RoleBody, false},
	EdgeIfStmtElse:           {"IfStmt.Else", KindIfStmt, "Else", RoleElseBranch, false},
	EdgeCaseClauseList:       {"CaseClause.List", KindCaseClause, "List", RoleCaseValue, true},
	EdgeCaseClauseBody:       {"CaseClause.Body", KindCaseClause, "Body", RoleStatement, true},
	EdgeSwitchStmtInit:       {"SwitchStmt.Init", KindSwitchStmt, "Init", RoleInitStatement, false},
	EdgeSwitchStmtTag:        {"SwitchStmt.Tag", KindSwitchStmt, "Tag", RoleSwitchTag, false},
	EdgeSwitchStmtBody:       {"SwitchStmt.Body", KindSwitchStmt, "Body", RoleBody, false},
	EdgeTypeSwitchStmtInit:   {"TypeSwitchStmt.Init", KindTypeSwitchStmt, "Init", RoleInitStatement, false},
	EdgeTypeSwitchStmtAssign: {"TypeSwitchStmt.Assign", KindTypeSwitchStmt, "Assign", RoleTypeSwitchGuard, false},
	EdgeTypeSwitchStmtBody:   {"TypeSwitchStmt.Body", KindTypeSwitchStmt, "Body", RoleBody, false},
	EdgeCommClauseComm:       {"CommClause.Comm", KindCommClause, "Comm", RoleCommStatement, false},
	EdgeCommClauseBody:       {"CommClause.Body", KindCommClause, "Body", RoleStatement, true},
	EdgeSelectStmtBody:       {"SelectStmt.Body", KindSelectStmt, "Body", RoleBody, false},
	EdgeForStmtInit:          {"ForStmt.Init", KindForStmt, "Init", RoleInitStatement, false},
	EdgeForStmtCond:          {"ForStmt.Cond", KindForStmt, "Cond", RoleCondition, false},
	EdgeForStmtPost:          {"ForStmt.Post", KindForStmt, "Post", RolePostStatement, false},
	EdgeForStmtBody:          {"ForStmt.Body", KindForStmt, "Body", RoleBody, false},
	EdgeRangeStmtKey:         {"RangeStmt.Key", KindRangeStmt, "Key", RoleRangeKey, false},
	EdgeRangeStmtValue:       {"RangeStmt.Value", KindRangeStmt, "Value", RoleRangeValue, false},
	EdgeRangeStmtX:           {"RangeStmt.X", KindRangeStmt, "X", RoleRangeOperand, false},
	EdgeRangeStmtBody:        {"RangeStmt.Body", KindRangeStmt, "Body", RoleBody, false},
	EdgeImportSpecDoc:        {"ImportSpec.Doc", KindImportSpec, "Doc", RoleDocumentation, false},
	EdgeImportSpecName:       {"ImportSpec.Name", KindImportSpec, "Name", RoleImportAlias, false},
	EdgeImportSpecPath:       {"ImportSpec.Path", KindImportSpec, "Path", RoleImportPath, false},
	EdgeImportSpecComment:    {"ImportSpec.Comment", KindImportSpec, "Comment", RoleTrailingDocumentation, false},
	EdgeValueSpecDoc:         {"ValueSpec.Doc", KindValueSpec, "Doc", RoleDocumentation, false},
	EdgeValueSpecNames:       {"ValueSpec.Names", KindValueSpec, "Names", RoleDeclarationName, true},
	EdgeValueSpecType:        {"ValueSpec.Type", KindValueSpec, "Type", RoleTypeExpression, false},
	EdgeValueSpecValues:      {"ValueSpec.Values", KindValueSpec, "Values", RoleInitializerValue, true},
	EdgeValueSpecComment:     {"ValueSpec.Comment", KindValueSpec, "Comment", RoleTrailingDocumentation, false},
	EdgeTypeSpecDoc:          {"TypeSpec.Doc", KindTypeSpec, "Doc", RoleDocumentation, false},
	EdgeTypeSpecName:         {"TypeSpec.Name", KindTypeSpec, "Name", RoleDeclarationName, false},
	EdgeTypeSpecTypeParams:   {"TypeSpec.TypeParams", KindTypeSpec, "TypeParams", RoleTypeParameters, false},
	EdgeTypeSpecType:         {"TypeSpec.Type", KindTypeSpec, "Type", RoleTypeExpression, false},
	EdgeTypeSpecComment:      {"TypeSpec.Comment", KindTypeSpec, "Comment", RoleTrailingDocumentation, false},
	EdgeGenDeclDoc:           {"GenDecl.Doc", KindGenDecl, "Doc", RoleDocumentation, false},
	EdgeGenDeclSpecs:         {"GenDecl.Specs", KindGenDecl, "Specs", RoleSpecification, true},
	EdgeFuncDeclDoc:          {"FuncDecl.Doc", KindFuncDecl, "Doc", RoleDocumentation, false},
	EdgeFuncDeclRecv:         {"FuncDecl.Recv", KindFuncDecl, "Recv", RoleReceiver, false},
	EdgeFuncDeclName:         {"FuncDecl.Name", KindFuncDecl, "Name", RoleDeclarationName, false},
	EdgeFuncDeclType:         {"FuncDecl.Type", KindFuncDecl, "Type", RoleFunctionSignature, false},
	EdgeFuncDeclBody:         {"FuncDecl.Body", KindFuncDecl, "Body", RoleFunctionBody, false},
	EdgeFileDoc:              {"File.Doc", KindFile, "Doc", RoleDocumentation, false},
	EdgeFileName:             {"File.Name", KindFile, "Name", RolePackageName, false},
	EdgeFileDecls:            {"File.Decls", KindFile, "Decls", RoleDeclaration, true},
	EdgeCommentGroupList:     {"CommentGroup.List", KindCommentGroup, "List", RoleCommentText, true},
	EdgeFieldDoc:             {"Field.Doc", KindField, "Doc", RoleDocumentation, false},
	EdgeFieldNames:           {"Field.Names", KindField, "Names", RoleDeclarationName, true},
	EdgeFieldType:            {"Field.Type", KindField, "Type", RoleTypeExpression, false},
	EdgeFieldTag:             {"Field.Tag", KindField, "Tag", RoleFieldTag, false},
	EdgeFieldComment:         {"Field.Comment", KindField, "Comment", RoleTrailingDocumentation, false},
	EdgeFieldListList:        {"FieldList.List", KindFieldList, "List", RoleFieldGroup, true},
}
