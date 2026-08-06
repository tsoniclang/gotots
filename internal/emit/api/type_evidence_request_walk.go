package api

import (
	"fmt"
	diagnosticcontract "github.com/tsoniclang/gotots/internal/emit/api/diagnostic"
	typesubstitution "github.com/tsoniclang/gotots/internal/emit/api/typesubstitution"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

type TypeArgumentList struct {
	values []types.Type
}

func NewTypeArgumentList(values []types.Type) (TypeArgumentList, error) {
	for _, value := range values {
		if value == nil {
			return TypeArgumentList{}, &InvariantError{
				Role:   RoleCallTypeArgument,
				Reason: "type argument is nil",
			}
		}
	}
	return TypeArgumentList{values: slices.Clone(values)}, nil
}

func TypeArgumentsFromGo(source *types.TypeList) TypeArgumentList {
	if source == nil {
		return TypeArgumentList{}
	}
	values := make([]types.Type, 0, source.Len())
	for index := range source.Len() {
		values = append(values, source.At(index))
	}
	result, err := NewTypeArgumentList(values)
	if err != nil {
		panic(err)
	}
	return result
}

func (a TypeArgumentList) Len() int {
	return len(a.values)
}

func (a TypeArgumentList) At(index int) types.Type {
	return a.values[index]
}

func (a TypeArgumentList) Values() []types.Type {
	return slices.Clone(a.values)
}

func (a TypeArgumentList) ContainsGenericTypeParameter() bool {
	for _, value := range a.values {
		if ContainsGenericTypeParameter(value) {
			return true
		}
	}
	return false
}

type TypeInstance struct {
	TypeArgs TypeArgumentList
	Type     types.Type
}

type TypeInfoView struct {
	source *types.Info
}

func (v TypeInfoView) Valid() bool {
	return v.source != nil
}

func newTypeInfoView(
	source *types.Info,
) TypeInfoView {
	return TypeInfoView{source: source}
}

func DirectTypeInfo(source *types.Info) TypeInfoView {
	return newTypeInfoView(source)
}

func (v TypeInfoView) TypeOf(source ast.Expr) types.Type {
	if v.source == nil {
		return nil
	}
	return v.source.TypeOf(source)
}

func (v TypeInfoView) TypeOfObject(source types.Object) types.Type {
	if v.source == nil || source == nil {
		return nil
	}
	return source.Type()
}

func (v TypeInfoView) TypeAndValue(
	source ast.Expr,
) (types.TypeAndValue, bool) {
	if v.source == nil {
		return types.TypeAndValue{}, false
	}
	facts, ok := v.source.Types[source]
	if !ok {
		return types.TypeAndValue{}, false
	}
	return facts, true
}

func (v TypeInfoView) DefOf(source *ast.Ident) types.Object {
	if v.source == nil || source == nil {
		return nil
	}
	return v.source.Defs[source]
}

func (v TypeInfoView) UseOf(source *ast.Ident) types.Object {
	if v.source == nil || source == nil {
		return nil
	}
	return v.source.Uses[source]
}

func (v TypeInfoView) ImplicitOf(source ast.Node) types.Object {
	if v.source == nil || source == nil {
		return nil
	}
	return v.source.Implicits[source]
}

func (v TypeInfoView) SelectionOf(
	source *ast.SelectorExpr,
) *types.Selection {
	if v.source == nil || source == nil {
		return nil
	}
	return v.source.Selections[source]
}

func (v TypeInfoView) InstanceOf(
	source *ast.Ident,
) (TypeInstance, bool) {
	if v.source == nil || source == nil {
		return TypeInstance{}, false
	}
	instance, ok := v.source.Instances[source]
	if !ok {
		return TypeInstance{}, false
	}
	arguments := make([]types.Type, 0, instance.TypeArgs.Len())
	for index := range instance.TypeArgs.Len() {
		arguments = append(arguments, instance.TypeArgs.At(index))
	}
	typeArguments, err := NewTypeArgumentList(arguments)
	if err != nil {
		panic(err)
	}
	return TypeInstance{TypeArgs: typeArguments, Type: instance.Type}, true
}

type TypeUse struct {
	Identifier *ast.Ident
	Object     types.Object
}

func (v TypeInfoView) Uses() []TypeUse {
	if v.source == nil {
		return nil
	}
	result := make([]TypeUse, 0, len(v.source.Uses))
	for identifier, object := range v.source.Uses {
		result = append(result, TypeUse{
			Identifier: identifier,
			Object:     object,
		})
	}
	return result
}

func SubstituteType(
	source types.Type,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	target, err := typesubstitution.SubstituteType(source, replacements)
	if substitutionError, ok := err.(*typesubstitution.Error); ok {
		return nil, &InvariantError{
			Role:   RoleCallTypeArgument,
			Reason: substitutionError.Reason,
		}
	}
	return target, err
}

func ContainsGenericTypeParameter(sourceType types.Type) bool {
	return typesubstitution.ContainsGenericTypeParameter(sourceType)
}

func genericTypeParametersIn(
	sourceType types.Type,
) map[*types.TypeParam]struct{} {
	return typesubstitution.GenericTypeParametersIn(sourceType)
}

type ContextualStructField struct {
	declaration *types.Var
	selected    *types.Var
	index       int
}

func (f ContextualStructField) Declaration() *types.Var {
	return f.declaration
}

func (f ContextualStructField) Selected() *types.Var {
	return f.selected
}

func (f ContextualStructField) Index() int {
	return f.index
}

func (v TypeInfoView) StructFieldAt(
	source *ast.CompositeLit,
	index int,
) (ContextualStructField, bool) {
	declaration, selected, ok := v.structFields(source)
	if !ok || index < 0 || index >= declaration.NumFields() {
		return ContextualStructField{}, false
	}
	return contextualStructField(declaration, selected, index)
}

func (v TypeInfoView) StructFieldOf(
	source *ast.CompositeLit,
	key *ast.Ident,
) (ContextualStructField, bool) {
	declaration, selected, ok := v.structFields(source)
	field, fieldOK := v.UseOf(key).(*types.Var)
	if !ok || !fieldOK {
		return ContextualStructField{}, false
	}
	for index := range declaration.NumFields() {
		if declaration.Field(index) == field || selected.Field(index) == field {
			return contextualStructField(declaration, selected, index)
		}
	}
	return ContextualStructField{}, false
}

func (v TypeInfoView) structFields(
	source *ast.CompositeLit,
) (*types.Struct, *types.Struct, bool) {
	if v.source == nil || source == nil {
		return nil, nil, false
	}
	declaration, selected, ok := contextualStructTypes(v.TypeOf(source))
	return declaration, selected,
		ok && declaration.NumFields() == selected.NumFields()
}

func contextualStructField(
	declaration *types.Struct,
	selected *types.Struct,
	index int,
) (ContextualStructField, bool) {
	sourceField := declaration.Field(index)
	targetField := selected.Field(index)
	if sourceField.Id() != targetField.Id() ||
		sourceField.Embedded() != targetField.Embedded() ||
		declaration.Tag(index) != selected.Tag(index) {
		return ContextualStructField{}, false
	}
	return ContextualStructField{
		declaration: sourceField,
		selected:    targetField,
		index:       index,
	}, true
}

func contextualStructTypes(
	source types.Type,
) (*types.Struct, *types.Struct, bool) {
	if source == nil {
		return nil, nil, false
	}
	selectedType := types.Unalias(source)
	if pointer, ok := selectedType.(*types.Pointer); ok {
		selectedType = types.Unalias(pointer.Elem())
	}
	if named, ok := selectedType.(*types.Named); ok {
		selected, selectedOK := named.Underlying().(*types.Struct)
		declaration, declarationOK := named.Origin().Underlying().(*types.Struct)
		return declaration, selected, declarationOK && selectedOK
	}
	selected, ok := selectedType.Underlying().(*types.Struct)
	return selected, selected, ok
}

type Category = diagnosticcontract.Category

const (
	CategoryDeclaration = diagnosticcontract.CategoryDeclaration
	CategoryStatement   = diagnosticcontract.CategoryStatement
	CategoryExpression  = diagnosticcontract.CategoryExpression
	CategoryType        = diagnosticcontract.CategoryType
)

type Role = diagnosticcontract.Role

const (
	RoleFileDeclaration       = diagnosticcontract.RoleFileDeclaration
	RoleFunctionBody          = diagnosticcontract.RoleFunctionBody
	RoleParameterType         = diagnosticcontract.RoleParameterType
	RoleResultType            = diagnosticcontract.RoleResultType
	RoleBlockStatement        = diagnosticcontract.RoleBlockStatement
	RoleReturnResult          = diagnosticcontract.RoleReturnResult
	RoleBinaryLeft            = diagnosticcontract.RoleBinaryLeft
	RoleBinaryRight           = diagnosticcontract.RoleBinaryRight
	RoleIndexOperand          = diagnosticcontract.RoleIndexOperand
	RoleIndexValue            = diagnosticcontract.RoleIndexValue
	RoleSliceOperand          = diagnosticcontract.RoleSliceOperand
	RoleSliceLow              = diagnosticcontract.RoleSliceLow
	RoleSliceHigh             = diagnosticcontract.RoleSliceHigh
	RoleParenthesizedValue    = diagnosticcontract.RoleParenthesizedValue
	RoleLocalType             = diagnosticcontract.RoleLocalType
	RoleLocalValue            = diagnosticcontract.RoleLocalValue
	RoleLocalDeclaration      = diagnosticcontract.RoleLocalDeclaration
	RoleLocalConstantType     = diagnosticcontract.RoleLocalConstantType
	RoleLocalConstantValue    = diagnosticcontract.RoleLocalConstantValue
	RoleAssignmentValue       = diagnosticcontract.RoleAssignmentValue
	RoleIfInitializer         = diagnosticcontract.RoleIfInitializer
	RoleIfCondition           = diagnosticcontract.RoleIfCondition
	RoleIfThen                = diagnosticcontract.RoleIfThen
	RoleIfElse                = diagnosticcontract.RoleIfElse
	RoleUnaryOperand          = diagnosticcontract.RoleUnaryOperand
	RoleConversionOperand     = diagnosticcontract.RoleConversionOperand
	RoleCallCallee            = diagnosticcontract.RoleCallCallee
	RoleCallArgument          = diagnosticcontract.RoleCallArgument
	RoleCallArgumentType      = diagnosticcontract.RoleCallArgumentType
	RoleCallTypeArgument      = diagnosticcontract.RoleCallTypeArgument
	RoleFunctionLiteralBody   = diagnosticcontract.RoleFunctionLiteralBody
	RoleCallableParameter     = diagnosticcontract.RoleCallableParameter
	RoleCallableResult        = diagnosticcontract.RoleCallableResult
	RoleNamedResultZero       = diagnosticcontract.RoleNamedResultZero
	RoleExpressionStatement   = diagnosticcontract.RoleExpressionStatement
	RoleIntegerConstantType   = diagnosticcontract.RoleIntegerConstantType
	RoleBooleanConstantType   = diagnosticcontract.RoleBooleanConstantType
	RolePackageConstantType   = diagnosticcontract.RolePackageConstantType
	RolePackageConstantValue  = diagnosticcontract.RolePackageConstantValue
	RolePackageVariableType   = diagnosticcontract.RolePackageVariableType
	RolePackageVariableZero   = diagnosticcontract.RolePackageVariableZero
	RolePackageVariableValue  = diagnosticcontract.RolePackageVariableValue
	RolePackageInitBody       = diagnosticcontract.RolePackageInitBody
	RoleForInitializer        = diagnosticcontract.RoleForInitializer
	RoleForCondition          = diagnosticcontract.RoleForCondition
	RoleForPost               = diagnosticcontract.RoleForPost
	RoleForBody               = diagnosticcontract.RoleForBody
	RoleSwitchInitializer     = diagnosticcontract.RoleSwitchInitializer
	RoleSwitchTag             = diagnosticcontract.RoleSwitchTag
	RoleSwitchClause          = diagnosticcontract.RoleSwitchClause
	RoleSwitchCaseExpression  = diagnosticcontract.RoleSwitchCaseExpression
	RoleSwitchCaseStatement   = diagnosticcontract.RoleSwitchCaseStatement
	RoleTypeSwitchInitializer = diagnosticcontract.RoleTypeSwitchInitializer
	RoleTypeSwitchOperand     = diagnosticcontract.RoleTypeSwitchOperand
	RoleTypeSwitchClause      = diagnosticcontract.RoleTypeSwitchClause
	RoleTypeSwitchCaseType    = diagnosticcontract.RoleTypeSwitchCaseType
	RoleTypeSwitchBinding     = diagnosticcontract.RoleTypeSwitchBinding
	RoleTypeSwitchStatement   = diagnosticcontract.RoleTypeSwitchStatement
	RoleStructField           = diagnosticcontract.RoleStructField
	RoleStructFieldType       = diagnosticcontract.RoleStructFieldType
	RoleStructZeroField       = diagnosticcontract.RoleStructZeroField
	RoleStructCopyField       = diagnosticcontract.RoleStructCopyField
	RoleStructAssignField     = diagnosticcontract.RoleStructAssignField
	RoleStructEqualField      = diagnosticcontract.RoleStructEqualField
	RoleStructHashField       = diagnosticcontract.RoleStructHashField
	RoleStorageType           = diagnosticcontract.RoleStorageType
	RoleDefinedUnderlyingType = diagnosticcontract.RoleDefinedUnderlyingType
	RoleDefinedTypeArgument   = diagnosticcontract.RoleDefinedTypeArgument
	RoleDefinedValue          = diagnosticcontract.RoleDefinedValue
	RoleCompositeElement      = diagnosticcontract.RoleCompositeElement
	RoleSliceElementType      = diagnosticcontract.RoleSliceElementType
	RoleSliceElement          = diagnosticcontract.RoleSliceElement
	RoleSliceReceiver         = diagnosticcontract.RoleSliceReceiver
	RoleSliceIndex            = diagnosticcontract.RoleSliceIndex
	RoleSliceMax              = diagnosticcontract.RoleSliceMax
	RoleFieldReceiver         = diagnosticcontract.RoleFieldReceiver
	RoleArrayReceiver         = diagnosticcontract.RoleArrayReceiver
	RoleArrayIndex            = diagnosticcontract.RoleArrayIndex
	RoleArrayElement          = diagnosticcontract.RoleArrayElement
	RoleBuiltinArgument       = diagnosticcontract.RoleBuiltinArgument
	RoleAssignmentTarget      = diagnosticcontract.RoleAssignmentTarget
	RoleReceiverType          = diagnosticcontract.RoleReceiverType
	RoleReceiverValue         = diagnosticcontract.RoleReceiverValue
	RoleMapKey                = diagnosticcontract.RoleMapKey
	RoleMapValue              = diagnosticcontract.RoleMapValue
	RoleMapSize               = diagnosticcontract.RoleMapSize
	RoleMapReceiver           = diagnosticcontract.RoleMapReceiver
	RoleChannelElementType    = diagnosticcontract.RoleChannelElementType
	RoleChannelElement        = diagnosticcontract.RoleChannelElement
	RoleChannelCapacity       = diagnosticcontract.RoleChannelCapacity
	RoleChannelOperand        = diagnosticcontract.RoleChannelOperand
	RoleGoroutineCall         = diagnosticcontract.RoleGoroutineCall
	RoleSelectClause          = diagnosticcontract.RoleSelectClause
	RoleSelectBody            = diagnosticcontract.RoleSelectBody
	RoleSelectReceiveTarget   = diagnosticcontract.RoleSelectReceiveTarget
	RoleRangeExpression       = diagnosticcontract.RoleRangeExpression
	RoleRangeKey              = diagnosticcontract.RoleRangeKey
	RoleRangeValue            = diagnosticcontract.RoleRangeValue
	RoleRangeBody             = diagnosticcontract.RoleRangeBody
	RoleLabelTarget           = diagnosticcontract.RoleLabelTarget
)

type UnsupportedError = diagnosticcontract.UnsupportedError

func Unsupported(
	context Context,
	category Category,
	source ast.Node,
) *UnsupportedError {
	var position token.Position
	if source != nil {
		position = context.FileSet().Position(source.Pos())
	}
	return &UnsupportedError{
		Category:  category,
		Construct: fmt.Sprintf("%T", source),
		Role:      context.Role(),
		Position:  position,
	}
}

type BuiltinBoundaryError = diagnosticcontract.BuiltinBoundaryError

func BuiltinBoundary(
	context Context,
	source ast.Node,
	builtin *types.Builtin,
	reason string,
) error {
	if builtin == nil || reason == "" {
		return &InvariantError{
			Role:   context.Role(),
			Reason: "built-in boundary lacks exact identity or reason",
		}
	}
	var position token.Position
	if source != nil {
		position = context.FileSet().Position(source.Pos())
	}
	return &BuiltinBoundaryError{
		Builtin:  builtin,
		Role:     context.Role(),
		Position: position,
		Reason:   reason,
	}
}

type ContextError = diagnosticcontract.ContextError

type RootRequestError = diagnosticcontract.RootRequestError

type InvariantError = diagnosticcontract.InvariantError

type NameError = diagnosticcontract.NameError

type PlacementError = diagnosticcontract.PlacementError

type rootRequestSequence struct {
	children []RootRequest
}

type rootRequestFrame struct {
	requests []RootRequest
	index    int
}

func combineRootRequests(groups ...[]RootRequest) []RootRequest {
	rootCount := 0
	for _, group := range groups {
		rootCount += len(group)
	}
	switch rootCount {
	case 0:
		return nil
	case 1:
		for _, group := range groups {
			if len(group) != 0 {
				return slices.Clone(group)
			}
		}
		panic("non-empty root request group disappeared")
	}

	children := make([]RootRequest, 0, rootCount)
	for _, group := range groups {
		children = append(children, group...)
	}
	return []RootRequest{{
		sequence: &rootRequestSequence{children: children},
	}}
}

func WalkRootRequests(
	requests []RootRequest,
	visit func(RootRequest) error,
) error {
	return walkRootRequestPayloads(requests, false, visit)
}

// WalkUniqueRootRequestPayloads visits each immutable payload and persistent
// sequence node once, preserving the order of their first occurrence.
func WalkUniqueRootRequestPayloads(
	requests []RootRequest,
	visit func(RootRequest) error,
) error {
	return walkRootRequestPayloads(requests, true, visit)
}

func walkRootRequestPayloads(
	requests []RootRequest,
	unique bool,
	visit func(RootRequest) error,
) error {
	if visit == nil {
		return &RootRequestError{Reason: "root request visitor is nil"}
	}
	frames := []rootRequestFrame{{requests: requests}}
	var visitedSequences map[*rootRequestSequence]struct{}
	var visitedPayloads map[*rootRequestPayload]struct{}
	if unique {
		visitedSequences = make(map[*rootRequestSequence]struct{})
		visitedPayloads = make(map[*rootRequestPayload]struct{})
	}
	for len(frames) != 0 {
		frame := &frames[len(frames)-1]
		if frame.index == len(frame.requests) {
			frames = frames[:len(frames)-1]
			continue
		}
		request := frame.requests[frame.index]
		frame.index++
		if request.sequence != nil {
			if len(request.sequence.children) == 0 {
				return &RootRequestError{
					Reason: "root request sequence is empty",
				}
			}
			if _, visited := visitedSequences[request.sequence]; visited {
				continue
			}
			if unique {
				visitedSequences[request.sequence] = struct{}{}
			}
			frames = append(frames, rootRequestFrame{
				requests: request.sequence.children,
			})
			continue
		}
		if request.Kind() == RootRequestInvalid {
			return &RootRequestError{Reason: "root request is invalid"}
		}
		if _, visited := visitedPayloads[request.payload]; visited {
			continue
		}
		if unique {
			visitedPayloads[request.payload] = struct{}{}
		}
		if err := visit(request); err != nil {
			return err
		}
	}
	return nil
}
