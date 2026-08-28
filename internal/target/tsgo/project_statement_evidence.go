package tsgo

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type ProjectStatementEvidence struct {
	kind     SyntaxKind
	typeOnly bool
}

func (e ProjectStatementEvidence) Kind() SyntaxKind { return e.kind }
func (e ProjectStatementEvidence) TypeOnly() bool   { return e.typeOnly }

type CallableImplementationSourceRole uint8

const (
	CallableImplementationSourceInvalid CallableImplementationSourceRole = iota
	CallableImplementationSourceModule
	CallableImplementationSourceCertification
)

func (r CallableImplementationSourceRole) Valid() bool {
	return r == CallableImplementationSourceModule ||
		r == CallableImplementationSourceCertification
}

type CallableImplementationSourceViolation uint8

const (
	CallableImplementationViolationInvalid CallableImplementationSourceViolation = iota
	CallableImplementationViolationSideEffectImport
	CallableImplementationViolationExecutableTopLevel
	CallableImplementationViolationBodylessFunction
	CallableImplementationViolationTypeAssertion
	CallableImplementationViolationNonNullAssertion
	CallableImplementationViolationSuppressionDirective
	CallableImplementationViolationExplicitAny
	CallableImplementationViolationExplicitUnknown
	CallableImplementationViolationInferredAny
	CallableImplementationViolationInferredUnknown
)

func (v CallableImplementationSourceViolation) String() string {
	switch v {
	case CallableImplementationViolationSideEffectImport:
		return "side-effect-import"
	case CallableImplementationViolationExecutableTopLevel:
		return "executable-top-level"
	case CallableImplementationViolationBodylessFunction:
		return "bodyless-function"
	case CallableImplementationViolationTypeAssertion:
		return "unchecked-type-assertion"
	case CallableImplementationViolationNonNullAssertion:
		return "non-null-assertion"
	case CallableImplementationViolationSuppressionDirective:
		return "diagnostic-suppression"
	case CallableImplementationViolationExplicitAny:
		return "explicit-any"
	case CallableImplementationViolationExplicitUnknown:
		return "explicit-unknown"
	case CallableImplementationViolationInferredAny:
		return "inferred-any"
	case CallableImplementationViolationInferredUnknown:
		return "inferred-unknown"
	default:
		return "invalid"
	}
}

func (p *ProjectInspection) SourceStatements(
	sourceFile string,
) ([]ProjectStatementEvidence, error) {
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("statements", sourceFile, err)
	}
	path := filepath.ToSlash(absolute)
	source, err := p.projectSourceEvidence(path)
	if err != nil {
		return nil, err
	}
	statementIndexes, err := source.statementIndexes(path)
	if err != nil {
		return nil, err
	}
	result := make([]ProjectStatementEvidence, len(statementIndexes))
	for index, statementIndex := range statementIndexes {
		statement := source.nodes[statementIndex]
		result[index] = ProjectStatementEvidence{kind: SyntaxKind(statement.kind)}
		switch statement.kind {
		case uint32(SyntaxKindExportDeclaration):
			result[index].typeOnly = statement.data>>24 == 1
		case uint32(SyntaxKindImportDeclaration):
			for _, child := range source.directChildren(statementIndex) {
				clause := source.nodes[child]
				if clause.kind != uint32(SyntaxKindImportClause) {
					continue
				}
				result[index].typeOnly = clause.data>>24 == 1
				break
			}
		default:
			continue
		}
	}
	return result, nil
}

func (p *ProjectInspection) CallableImplementationSourceViolations(
	sourceFile string,
	role CallableImplementationSourceRole,
) ([]CallableImplementationSourceViolation, error) {
	if p == nil || !role.Valid() {
		return nil, &ProjectInspectionError{
			Operation: "callable implementation source",
			Path:      sourceFile,
			Reason:    "inspection or source role is invalid",
		}
	}
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("callable implementation source", sourceFile, err)
	}
	path := filepath.ToSlash(absolute)
	source, err := p.projectSourceEvidence(path)
	if err != nil {
		return nil, err
	}
	violations := make(map[CallableImplementationSourceViolation]struct{})
	if role == CallableImplementationSourceModule {
		if err := source.collectCallableTopLevelViolations(path, violations); err != nil {
			return nil, err
		}
	}
	for _, node := range source.nodes {
		switch SyntaxKind(node.kind) {
		case SyntaxKindAsExpression, SyntaxKindTypeAssertionExpression:
			violations[CallableImplementationViolationTypeAssertion] = struct{}{}
		case SyntaxKindNonNullExpression:
			violations[CallableImplementationViolationNonNullAssertion] = struct{}{}
		case SyntaxKindAnyKeyword:
			violations[CallableImplementationViolationExplicitAny] = struct{}{}
		case SyntaxKindUnknownKeyword:
			violations[CallableImplementationViolationExplicitUnknown] = struct{}{}
		}
	}
	for _, directive := range []string{"@ts-ignore", "@ts-nocheck", "@ts-expect-error"} {
		if strings.Contains(source.text, directive) {
			violations[CallableImplementationViolationSuppressionDirective] = struct{}{}
			break
		}
	}
	dynamic, err := p.sourceDynamicTypeFlags(path, source)
	if err != nil {
		return nil, err
	}
	if dynamic&typeFlagAny != 0 {
		violations[CallableImplementationViolationInferredAny] = struct{}{}
	}
	if dynamic&typeFlagUnknown != 0 {
		violations[CallableImplementationViolationInferredUnknown] = struct{}{}
	}
	result := make([]CallableImplementationSourceViolation, 0, len(violations))
	for violation := range violations {
		result = append(result, violation)
	}
	slices.Sort(result)
	return result, nil
}

func (source projectSourceEvidence) statementIndexes(path string) ([]uint32, error) {
	if len(source.nodes) < 3 ||
		source.nodes[1].kind != uint32(SyntaxKindSourceFile) ||
		source.nodes[1].parent != 0 {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    "official source-file root is invalid",
		}
	}
	statementLists := make([]uint32, 0, 1)
	for _, child := range source.directChildren(1) {
		if source.nodes[child].kind == kindNodeList {
			statementLists = append(statementLists, child)
		}
	}
	if len(statementLists) != 1 {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    fmt.Sprintf("official source has %d statement lists", len(statementLists)),
		}
	}
	indexes := source.directChildren(statementLists[0])
	if uint32(len(indexes)) != source.nodes[statementLists[0]].data {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    "official statement-list cardinality differs",
		}
	}
	return indexes, nil
}

func (source projectSourceEvidence) collectCallableTopLevelViolations(
	path string,
	violations map[CallableImplementationSourceViolation]struct{},
) error {
	statements, err := source.statementIndexes(path)
	if err != nil {
		return err
	}
	for _, index := range statements {
		switch SyntaxKind(source.nodes[index].kind) {
		case SyntaxKindImportDeclaration:
			if !source.hasDirectChild(index, SyntaxKindImportClause) {
				violations[CallableImplementationViolationSideEffectImport] = struct{}{}
			}
		case SyntaxKindInterfaceDeclaration, SyntaxKindTypeAliasDeclaration:
		case SyntaxKindFunctionDeclaration:
			if !source.hasDirectChild(index, SyntaxKindBlock) {
				violations[CallableImplementationViolationBodylessFunction] = struct{}{}
			}
		default:
			violations[CallableImplementationViolationExecutableTopLevel] = struct{}{}
		}
	}
	return nil
}

func (source projectSourceEvidence) hasDirectChild(parent uint32, kind SyntaxKind) bool {
	for _, child := range source.directChildren(parent) {
		if source.nodes[child].kind == uint32(kind) {
			return true
		}
	}
	return false
}

const (
	typeFlagAny             uint32 = 1 << 0
	typeFlagUnknown         uint32 = 1 << 1
	typeFlagObject          uint32 = 1 << 20
	typeFlagIndex           uint32 = 1 << 21
	typeFlagTemplateLiteral uint32 = 1 << 22
	typeFlagStringMapping   uint32 = 1 << 23
	typeFlagSubstitution    uint32 = 1 << 24
	typeFlagIndexedAccess   uint32 = 1 << 25
	typeFlagConditional     uint32 = 1 << 26
	objectFlagReference     uint32 = 1 << 2
)

func (p *ProjectInspection) sourceDynamicTypeFlags(
	path string,
	source projectSourceEvidence,
) (uint32, error) {
	const batchSize = 512
	type dynamicTypeLocation struct {
		handle                    string
		inspectCallableSignatures bool
	}
	locations := make([]dynamicTypeLocation, 0, len(source.nodes))
	for index := 1; index < len(source.nodes); index++ {
		if !source.callableDynamicTypeLocation(index) {
			continue
		}
		locations = append(
			locations,
			dynamicTypeLocation{
				handle: projectNodeHandle(
					path,
					uint32(index),
					source.nodes[index].kind,
				),
				inspectCallableSignatures: !source.directInvocationReference(index),
			},
		)
	}
	seen := [2]map[uint32]struct{}{
		make(map[uint32]struct{}),
		make(map[uint32]struct{}),
	}
	var result uint32
	for start := 0; start < len(locations); start += batchSize {
		end := min(start+batchSize, len(locations))
		handles := make([]string, end-start)
		for offset, location := range locations[start:end] {
			handles[offset] = location.handle
		}
		var types []*typeResponse
		if err := requestProjectJSON(
			p.client,
			"getTypeAtLocations",
			getTypesAtLocationsParams{
				Snapshot:  p.snapshot,
				Project:   p.project,
				Locations: handles,
			},
			&types,
		); err != nil {
			return 0, err
		}
		if len(types) != end-start {
			return 0, &ProjectInspectionError{
				Operation: "callable implementation source",
				Path:      path,
				Reason:    "checker type denominator differs",
			}
		}
		for offset, selected := range types {
			if selected == nil || selected.ID == 0 {
				continue
			}
			inspectCallableSignatures := locations[start+offset].inspectCallableSignatures
			seenIndex := 0
			if inspectCallableSignatures {
				seenIndex = 1
			}
			flags, err := p.dynamicTypeFlags(
				*selected,
				seen[seenIndex],
				inspectCallableSignatures,
			)
			if err != nil {
				return 0, err
			}
			result |= flags
		}
	}
	return result, nil
}

func (source projectSourceEvidence) directInvocationReference(index int) bool {
	current := uint32(index)
	for current != 0 {
		parent := source.nodes[current].parent
		if parent == 0 || parent >= uint32(len(source.nodes)) {
			return false
		}
		switch SyntaxKind(source.nodes[parent].kind) {
		case SyntaxKindCallExpression,
			SyntaxKindNewExpression,
			SyntaxKindTaggedTemplateExpression:
			return source.firstChild(parent) == current
		case SyntaxKindPropertyAccessExpression:
			current = parent
		case SyntaxKindElementAccessExpression:
			if source.firstChild(parent) != current {
				return false
			}
			current = parent
		case SyntaxKindParenthesizedExpression, SyntaxKindSatisfiesExpression:
			if source.firstChild(parent) != current {
				return false
			}
			current = parent
		default:
			return false
		}
	}
	return false
}

func (source projectSourceEvidence) callableDynamicTypeLocation(index int) bool {
	kind := SyntaxKind(source.nodes[index].kind)
	if kind == SyntaxKindIdentifier {
		for parent := source.nodes[index].parent; parent != 0 &&
			parent < uint32(len(source.nodes)); parent = source.nodes[parent].parent {
			parentKind := SyntaxKind(source.nodes[parent].kind)
			if parentKind == SyntaxKindTypeParameter ||
				parentKind >= SyntaxKindFirstTypeNode && parentKind <= SyntaxKindLastTypeNode {
				return false
			}
		}
	}
	if kind >= SyntaxKindFirstTypeNode && kind <= SyntaxKindLastTypeNode {
		return true
	}
	switch kind {
	case SyntaxKindIdentifier,
		SyntaxKindPrivateIdentifier,
		SyntaxKindJsxElement,
		SyntaxKindJsxSelfClosingElement,
		SyntaxKindJsxFragment,
		SyntaxKindPartiallyEmittedExpression,
		SyntaxKindSyntheticReferenceExpression:
		return true
	default:
		return kind >= SyntaxKindArrayLiteralExpression &&
			kind <= SyntaxKindSatisfiesExpression
	}
}

func (p *ProjectInspection) dynamicTypeFlags(
	selected typeResponse,
	seen map[uint32]struct{},
	inspectCallableSignatures bool,
) (uint32, error) {
	result := selected.Flags & (typeFlagAny | typeFlagUnknown)
	if selected.ID == 0 {
		return result, nil
	}
	if _, visited := seen[selected.ID]; visited {
		return result, nil
	}
	seen[selected.ID] = struct{}{}
	var nested []typeResponse
	switch {
	case selected.Flags&typeFlagUnionOrIntersection != 0,
		selected.Flags&typeFlagTemplateLiteral != 0:
		members, err := p.compositeTypes(selected.ID)
		if err != nil {
			return 0, err
		}
		nested = append(nested, members...)
	case selected.Flags&typeFlagIndex != 0,
		selected.Flags&typeFlagStringMapping != 0:
		target, err := p.dynamicTypeProperty("getTargetOfType", selected.ID)
		if err != nil {
			return 0, err
		}
		if target != nil {
			nested = append(nested, *target)
		}
	case selected.Flags&typeFlagSubstitution != 0:
		for _, method := range []string{"getBaseTypeOfType", "getConstraintOfType"} {
			child, err := p.dynamicTypeProperty(method, selected.ID)
			if err != nil {
				return 0, err
			}
			if child != nil {
				nested = append(nested, *child)
			}
		}
	case selected.Flags&typeFlagIndexedAccess != 0:
		for _, method := range []string{"getObjectTypeOfType", "getIndexTypeOfType"} {
			child, err := p.dynamicTypeProperty(method, selected.ID)
			if err != nil {
				return 0, err
			}
			if child != nil {
				nested = append(nested, *child)
			}
		}
	case selected.Flags&typeFlagConditional != 0:
		for _, method := range []string{"getCheckTypeOfType", "getExtendsTypeOfType"} {
			child, err := p.dynamicTypeProperty(method, selected.ID)
			if err != nil {
				return 0, err
			}
			if child != nil {
				nested = append(nested, *child)
			}
		}
	}
	if selected.Flags&typeFlagObject != 0 &&
		selected.ObjectFlags&objectFlagReference != 0 {
		arguments, err := p.typeArguments(selected.ID)
		if err != nil {
			return 0, err
		}
		nested = append(nested, arguments...)
	}
	if len(selected.AliasTypeArguments) != 0 {
		arguments, err := p.dynamicTypeArrayProperty(
			"getAliasTypeArgumentsOfType",
			selected.ID,
		)
		if err != nil {
			return 0, err
		}
		nested = append(nested, arguments...)
	}
	if selected.Flags&typeFlagObject != 0 && inspectCallableSignatures {
		callableTypes, err := p.dynamicCallableTypes(selected.ID)
		if err != nil {
			return 0, err
		}
		nested = append(nested, callableTypes...)
	}
	if selected.Flags&typeFlagObject != 0 {
		indexTypes, err := p.dynamicIndexTypes(selected.ID)
		if err != nil {
			return 0, err
		}
		nested = append(nested, indexTypes...)
	}
	for _, child := range nested {
		flags, err := p.dynamicTypeFlags(child, seen, inspectCallableSignatures)
		if err != nil {
			return 0, err
		}
		result |= flags
	}
	return result, nil
}

type getTypesAtLocationsParams struct {
	Snapshot  uint64   `json:"snapshot"`
	Project   string   `json:"project"`
	Locations []string `json:"locations"`
}
