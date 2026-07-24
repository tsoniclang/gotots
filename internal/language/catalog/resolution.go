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

// ResolutionDomain is the one retained-region class in which an occurrence
// is interpreted. It distinguishes declaration/header compile-time syntax
// from executable syntax without asking a child to inspect its parent.
type ResolutionDomain uint8

const (
	ResolutionDomainInvalid ResolutionDomain = iota
	ResolutionDomainOwner
	ResolutionDomainHeader
	ResolutionDomainBoundary
	ResolutionDomainExecutable
)

func (domain ResolutionDomain) Valid() bool {
	return domain >= ResolutionDomainOwner &&
		domain <= ResolutionDomainExecutable
}

func (domain ResolutionDomain) String() string {
	switch domain {
	case ResolutionDomainOwner:
		return "owner"
	case ResolutionDomainHeader:
		return "header"
	case ResolutionDomainBoundary:
		return "boundary"
	case ResolutionDomainExecutable:
		return "executable"
	default:
		return "invalid"
	}
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
	KindUnaryExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
	KindBinaryExpr: resolutionClasses(
		ResolutionClassType,
		ResolutionClassOperation,
	),
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

var compileTimeStructuralByKind = [kindCount + 1]bool{
	KindIdent:         true,
	KindEllipsis:      true,
	KindBasicLit:      true,
	KindCompositeLit:  true,
	KindParenExpr:     true,
	KindSelectorExpr:  true,
	KindIndexExpr:     true,
	KindIndexListExpr: true,
	KindCallExpr:      true,
	KindUnaryExpr:     true,
	KindBinaryExpr:    true,
	KindKeyValueExpr:  true,
	KindArrayType:     true,
	KindStructType:    true,
	KindFuncType:      true,
	KindInterfaceType: true,
	KindMapType:       true,
	KindChanType:      true,
}

// AllowsCompileTimeStructural reports whether one syntax kind may be
// represented by positive declaration/type coverage in a non-executable
// domain. It is the sole catalog owner of this closed exception.
func AllowsCompileTimeStructural(kind Kind) bool {
	return kind.Valid() && compileTimeStructuralByKind[kind]
}

func AllowsCompileTimeStructuralResolution(
	kind Kind,
	role Role,
	domain ResolutionDomain,
) bool {
	return kind.Disposition() == DispositionActive &&
		(role == RoleInvalid || role.Valid()) &&
		domain.Valid() &&
		domain != ResolutionDomainBoundary &&
		AllowsCompileTimeStructural(kind)
}

func AllowsIntrinsicContract(
	kind Kind,
	role Role,
	domain ResolutionDomain,
) bool {
	if !kind.Valid() ||
		kind.Disposition() != DispositionActive ||
		domain != ResolutionDomainHeader ||
		(role != RoleInvalid && !role.Valid()) {
		return false
	}
	if AllowsCompileTimeStructural(kind) {
		return true
	}
	mask := resolutionMaskFor(kind, role)
	return mask.has(ResolutionClassStructural) ||
		mask.has(ResolutionClassDefinitionComponent) ||
		mask.has(ResolutionClassDeclaration) ||
		mask.has(ResolutionClassBinding) ||
		mask.has(ResolutionClassType)
}

// AllowsResolution reports whether class is legal for one exact
// kind/role/variant combination. Invalid combinations fail closed.
func AllowsResolution(
	kind Kind,
	role Role,
	variant Variant,
	domain ResolutionDomain,
	class ResolutionClass,
) bool {
	if !kind.Valid() ||
		(role != RoleInvalid && !role.Valid()) ||
		!domain.Valid() ||
		!class.Valid() {
		return false
	}
	variantAllowed := VariantAllowed(kind, variant)
	if class == ResolutionClassStructural &&
		domain != ResolutionDomainExecutable &&
		variant == VariantNone &&
		AllowsCompileTimeStructural(kind) {
		variantAllowed = true
	}
	if class == ResolutionClassUnsupported &&
		domain == ResolutionDomainExecutable &&
		variant == VariantNone &&
		kind.Disposition() == DispositionActive {
		variantAllowed = true
	}
	if !variantAllowed {
		return false
	}
	if domain == ResolutionDomainBoundary {
		return class == ResolutionClassDefinitionComponent ||
			(class == ResolutionClassUnsupported &&
				resolutionMaskFor(kind, role).has(class))
	}
	if class == ResolutionClassStructural &&
		domain == ResolutionDomainOwner &&
		kind.Disposition() == DispositionActive {
		return true
	}
	if class == ResolutionClassOperation {
		return domain == ResolutionDomainExecutable &&
			RoleMayOwnRuntimeOperation(role) &&
			resolutionMaskFor(kind, role).has(class)
	}
	if class == ResolutionClassUnsupported &&
		domain == ResolutionDomainExecutable &&
		kind.Disposition() == DispositionActive {
		return true
	}
	if class == ResolutionClassStructural &&
		(domain == ResolutionDomainOwner ||
			domain == ResolutionDomainHeader) &&
		compileTimeStructuralByKind[kind] {
		return true
	}
	mask := resolutionMaskFor(kind, role)
	if class == ResolutionClassStructural &&
		domain == ResolutionDomainExecutable &&
		!RoleMayOwnRuntimeOperation(role) {
		return mask.has(class)
	}
	if kind == KindIdent &&
		domain != ResolutionDomainExecutable &&
		mask.has(ResolutionClassOperation) {
		mask &^= 1 << ResolutionClassOperation
		mask |= resolutionClasses(
			ResolutionClassDeclaration,
			ResolutionClassBinding,
		)
	}
	return mask.has(class)
}

func resolutionMaskFor(
	kind Kind,
	role Role,
) resolutionMask {
	mask := resolutionByKind[kind]
	if kind == KindIdent {
		mask = identifierResolution(role)
	}
	if kind == KindBasicLit && role == RoleImportPath {
		mask = resolutionClasses(ResolutionClassStructural)
	}
	return mask
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
		RoleResults:
		return resolutionClasses(
			ResolutionClassDefinitionComponent,
			ResolutionClassDeclaration,
			ResolutionClassBinding,
			ResolutionClassStructural,
		)
	case RoleAssignmentTarget:
		return resolutionClasses(
			ResolutionClassOperation,
			ResolutionClassStructural,
		)
	case
		RoleRangeKey,
		RoleRangeValue:
		return resolutionClasses(
			ResolutionClassDefinitionComponent,
			ResolutionClassDeclaration,
			ResolutionClassBinding,
			ResolutionClassStructural,
			ResolutionClassOperation,
		)
	case RoleLabelReference:
		return resolutionClasses(ResolutionClassBinding)
	case RoleCallee:
		return resolutionClasses(
			ResolutionClassType,
			ResolutionClassOperation,
		)
	case RoleCallArgument,
		RoleOperand,
		RoleLeftOperand,
		RoleRightOperand,
		RoleIndexedOperand,
		RoleCaseValue:
		return resolutionClasses(
			ResolutionClassType,
			ResolutionClassOperation,
		)
	case RoleIndex:
		return resolutionClasses(
			ResolutionClassBinding,
			ResolutionClassType,
			ResolutionClassOperation,
		)
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
