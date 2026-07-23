package catalog

// ResolutionClass is the catalog-owned legal semantic disposition of one
// occurrence. The semantic model has its own record variants; this enum keeps
// legality in the Go-language catalog without importing that model.
type ResolutionClass uint8

const (
	ResolutionClassInvalid ResolutionClass = iota
	ResolutionClassStructural
	ResolutionClassDefinitionComponent
	ResolutionClassDeclaration
	ResolutionClassBinding
	ResolutionClassType
	ResolutionClassOperation
	ResolutionClassUnsupported

	resolutionClassCount = ResolutionClassUnsupported
)

func (class ResolutionClass) Valid() bool {
	return class > ResolutionClassInvalid &&
		class <= resolutionClassCount
}

type resolutionMask uint8

func resolutionClasses(classes ...ResolutionClass) resolutionMask {
	var mask resolutionMask
	for _, class := range classes {
		mask |= 1 << class
	}
	return mask
}

func (mask resolutionMask) has(class ResolutionClass) bool {
	return class.Valid() && mask&(1<<class) != 0
}

var resolutionByKind = [kindCount + 1]resolutionMask{
	KindBadExpr: resolutionClasses(ResolutionClassUnsupported),
	KindIdent: resolutionClasses(
		ResolutionClassStructural,
		ResolutionClassDeclaration,
		ResolutionClassBinding,
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindEllipsis: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindBasicLit: resolutionClasses(
		ResolutionClassStructural,
		ResolutionClassOperation,
	),
	KindFuncLit: resolutionClasses(
		ResolutionClassDefinitionComponent,
	),
	KindCompositeLit: resolutionClasses(ResolutionClassOperation),
	KindParenExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindSelectorExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindIndexExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindIndexListExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindSliceExpr:      resolutionClasses(ResolutionClassOperation),
	KindTypeAssertExpr: resolutionClasses(ResolutionClassOperation),
	KindCallExpr:       resolutionClasses(ResolutionClassOperation),
	KindStarExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindUnaryExpr:    resolutionClasses(ResolutionClassOperation),
	KindBinaryExpr:   resolutionClasses(ResolutionClassOperation),
	KindKeyValueExpr: resolutionClasses(ResolutionClassOperation),

	KindArrayType:     resolutionClasses(ResolutionClassType),
	KindStructType:    resolutionClasses(ResolutionClassType),
	KindFuncType:      resolutionClasses(ResolutionClassType),
	KindInterfaceType: resolutionClasses(ResolutionClassType),
	KindMapType:       resolutionClasses(ResolutionClassType),
	KindChanType:      resolutionClasses(ResolutionClassType),

	KindBadStmt:        resolutionClasses(ResolutionClassUnsupported),
	KindDeclStmt:       resolutionClasses(ResolutionClassOperation),
	KindEmptyStmt:      resolutionClasses(ResolutionClassOperation),
	KindLabeledStmt:    resolutionClasses(ResolutionClassOperation),
	KindExprStmt:       resolutionClasses(ResolutionClassOperation),
	KindSendStmt:       resolutionClasses(ResolutionClassOperation),
	KindIncDecStmt:     resolutionClasses(ResolutionClassOperation),
	KindAssignStmt:     resolutionClasses(ResolutionClassOperation),
	KindGoStmt:         resolutionClasses(ResolutionClassOperation),
	KindDeferStmt:      resolutionClasses(ResolutionClassOperation),
	KindReturnStmt:     resolutionClasses(ResolutionClassOperation),
	KindBranchStmt:     resolutionClasses(ResolutionClassOperation),
	KindBlockStmt:      resolutionClasses(ResolutionClassOperation),
	KindIfStmt:         resolutionClasses(ResolutionClassOperation),
	KindCaseClause:     resolutionClasses(ResolutionClassOperation),
	KindSwitchStmt:     resolutionClasses(ResolutionClassOperation),
	KindTypeSwitchStmt: resolutionClasses(ResolutionClassOperation),
	KindCommClause:     resolutionClasses(ResolutionClassOperation),
	KindSelectStmt:     resolutionClasses(ResolutionClassOperation),
	KindForStmt:        resolutionClasses(ResolutionClassOperation),
	KindRangeStmt:      resolutionClasses(ResolutionClassOperation),

	KindBadDecl: resolutionClasses(ResolutionClassUnsupported),
	KindGenDecl: resolutionClasses(ResolutionClassStructural),
	KindFuncDecl: resolutionClasses(
		ResolutionClassDefinitionComponent,
		ResolutionClassDeclaration,
	),
	KindImportSpec: resolutionClasses(
		ResolutionClassStructural,
		ResolutionClassDeclaration,
	),
	KindValueSpec: resolutionClasses(
		ResolutionClassStructural,
		ResolutionClassDefinitionComponent,
		ResolutionClassDeclaration,
	),
	KindTypeSpec: resolutionClasses(
		ResolutionClassDeclaration,
		ResolutionClassType,
	),
	KindFile:         resolutionClasses(ResolutionClassStructural),
	KindComment:      resolutionClasses(ResolutionClassStructural),
	KindCommentGroup: resolutionClasses(ResolutionClassStructural),
	KindField: resolutionClasses(
		ResolutionClassStructural,
		ResolutionClassDeclaration,
		ResolutionClassBinding,
	),
	KindFieldList: resolutionClasses(ResolutionClassStructural),
	KindDirective: resolutionClasses(ResolutionClassStructural),
	KindPackage:   resolutionClasses(ResolutionClassUnsupported),
}

// AllowsResolution reports whether class is legal for one exact
// kind/role/variant combination. Invalid combinations fail closed.
func AllowsResolution(
	kind Kind,
	role Role,
	variant Variant,
	class ResolutionClass,
) bool {
	if !kind.Valid() ||
		(role != RoleInvalid && !role.Valid()) ||
		!VariantAllowed(kind, variant) ||
		!class.Valid() {
		return false
	}
	mask := resolutionByKind[kind]
	if kind == KindIdent {
		mask = identifierResolution(role)
	}
	if kind == KindBasicLit && role == RoleImportPath {
		mask = resolutionClasses(ResolutionClassStructural)
	}
	return mask.has(class)
}

func identifierResolution(role Role) resolutionMask {
	switch role {
	case RolePackageName,
		RoleDocumentation,
		RoleTrailingDocumentation,
		RoleCommentText:
		return resolutionClasses(ResolutionClassStructural)
	case RoleTypeExpression,
		RoleConstructedType,
		RoleAssertedType,
		RoleElementType,
		RoleKeyType,
		RoleValueType,
		RoleTypeParameters:
		return resolutionClasses(ResolutionClassType)
	case RoleDeclarationName,
		RoleImportAlias,
		RoleLabelDeclaration,
		RoleReceiver,
		RoleParameters,
		RoleResults,
		RoleRangeKey,
		RoleRangeValue:
		return resolutionClasses(
			ResolutionClassDeclaration,
			ResolutionClassBinding,
			ResolutionClassStructural,
		)
	case RoleLabelReference:
		return resolutionClasses(ResolutionClassBinding)
	case RoleSelectedName:
		return resolutionClasses(
			ResolutionClassDeclaration,
			ResolutionClassBinding,
			ResolutionClassType,
		)
	default:
		return resolutionClasses(ResolutionClassOperation)
	}
}

// VariantAllowed is the one kind-to-variant authority.
func VariantAllowed(kind Kind, variant Variant) bool {
	if !kind.Valid() || !variant.Valid() {
		return false
	}
	switch kind {
	case KindCallExpr:
		return variant == VariantCallFunction ||
			variant == VariantCallMethod ||
			variant == VariantCallBuiltin ||
			variant == VariantConversion
	case KindIndexExpr:
		return variant == VariantIndexElement ||
			variant == VariantMapLookupValue ||
			variant == VariantMapLookupCommaOk ||
			variant == VariantGenericInstantiation
	case KindIndexListExpr:
		return variant == VariantGenericInstantiation
	case KindTypeAssertExpr:
		return variant == VariantAssertValue ||
			variant == VariantAssertCommaOk ||
			variant == VariantTypeSwitchGuard
	case KindSelectorExpr:
		return variant == VariantSelectField ||
			variant == VariantSelectPromotedField ||
			variant == VariantSelectMethodValue ||
			variant == VariantSelectMethodExpression ||
			variant == VariantSelectPackageMember
	case KindAssignStmt:
		return variant >= VariantAssignBalanced &&
			variant <= VariantAssignCompound
	case KindCompositeLit:
		return variant >= VariantLitStruct &&
			variant <= VariantLitMap
	case KindKeyValueExpr:
		return variant >= VariantKeyFieldName &&
			variant <= VariantKeyArrayIndex
	case KindUnaryExpr:
		return variant == VariantNone ||
			variant == VariantReceiveValue ||
			variant == VariantReceiveCommaOk
	case KindStarExpr:
		return variant == VariantStarPointerType ||
			variant == VariantStarDereference
	case KindReturnStmt:
		return variant >= VariantReturnVoid &&
			variant <= VariantReturnBare
	case KindRangeStmt:
		return variant >= VariantRangeArray &&
			variant <= VariantRangeFunc
	case KindSwitchStmt:
		return variant == VariantSwitchExpression ||
			variant == VariantSwitchTrue
	case KindTypeSpec:
		return variant == VariantTypeDefinition ||
			variant == VariantTypeAlias
	case KindCommClause:
		return variant >= VariantCommSend &&
			variant <= VariantCommDefault
	default:
		return variant == VariantNone
	}
}
