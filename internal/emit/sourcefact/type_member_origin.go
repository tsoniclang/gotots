package sourcefact

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type MemberOriginSet struct {
	byObject         map[types.Object]DeclarationOrigin
	orderedField     []*types.Var
	fieldOrigin      []DeclarationOrigin
	environmentBasis string
}

func NewMemberOriginSet(
	objects []types.Object,
	origins []DeclarationOrigin,
) (MemberOriginSet, error) {
	if len(objects) != len(origins) {
		return MemberOriginSet{}, &Error{Reason: "member origin denominator is inexact"}
	}
	result := MemberOriginSet{
		byObject: make(map[types.Object]DeclarationOrigin, len(objects)),
	}
	for index, object := range objects {
		origin := origins[index]
		if object == nil || !origin.Valid() {
			return MemberOriginSet{}, &Error{Reason: "member origin is incomplete"}
		}
		if _, duplicate := result.byObject[object]; duplicate {
			return MemberOriginSet{}, &Error{Subject: object.Name(), Reason: "member origin is duplicated"}
		}
		result.byObject[object] = origin
		if basis, bounded := origin.EnvironmentBasis(); bounded {
			if result.environmentBasis != "" && result.environmentBasis != basis {
				return MemberOriginSet{}, &Error{Reason: "member environment basis is inconsistent"}
			}
			result.environmentBasis = basis
		}
		if field, ok := object.(*types.Var); ok {
			result.orderedField = append(result.orderedField, field)
			result.fieldOrigin = append(result.fieldOrigin, origin)
		}
	}
	return result, nil
}

func (s MemberOriginSet) boundedEnvironment() bool {
	return s.environmentBasis != ""
}

func (s MemberOriginSet) field(
	ordinal int,
	field *types.Var,
) (DeclarationOrigin, bool) {
	if field == nil || ordinal < 0 {
		return DeclarationOrigin{}, false
	}
	if origin := s.byObject[field]; origin.Valid() {
		return origin, true
	}
	if ordinal >= len(s.orderedField) || ordinal >= len(s.fieldOrigin) ||
		!sameField(s.orderedField[ordinal], field) {
		return DeclarationOrigin{}, false
	}
	return s.fieldOrigin[ordinal], true
}

func (s MemberOriginSet) method(method *types.Func) (DeclarationOrigin, bool) {
	origin := s.byObject[method]
	return origin, origin.Valid()
}

func sameField(left *types.Var, right *types.Var) bool {
	if left == nil || right == nil ||
		left.Name() != right.Name() ||
		left.Embedded() != right.Embedded() ||
		!types.Identical(left.Type(), right.Type()) {
		return false
	}
	leftPackage := ""
	if left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	return leftPackage == rightPackage
}

func TypeMemberOrigins(
	context api.Context,
	owner *types.TypeName,
	source *ast.TypeSpec,
	origin DeclarationOrigin,
) (MemberOriginSet, error) {
	if owner == nil || source == nil || !origin.Valid() ||
		context.TypesInfo().DefOf(source.Name) != owner {
		return MemberOriginSet{}, &Error{Reason: "type declaration occurrence is not exact"}
	}
	var objects []types.Object
	var origins []DeclarationOrigin
	switch selected := source.Type.(type) {
	case *ast.StructType:
		structure, ok := owner.Type().Underlying().(*types.Struct)
		if !ok {
			return MemberOriginSet{}, &Error{Subject: owner.Name(), Reason: "struct type evidence is inconsistent"}
		}
		fieldIndex := 0
		for _, field := range selected.Fields.List {
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			fieldOrigin, err := AuthoredOrigin(context, field)
			if err != nil {
				return MemberOriginSet{}, err
			}
			if basis, bounded := origin.EnvironmentBasis(); bounded {
				fieldOrigin, err = fieldOrigin.WithEnvironmentBasis(basis)
				if err != nil {
					return MemberOriginSet{}, err
				}
			}
			for range count {
				if fieldIndex >= structure.NumFields() {
					return MemberOriginSet{}, &Error{Subject: owner.Name(), Reason: "struct field occurrence denominator overflowed"}
				}
				objects = append(objects, structure.Field(fieldIndex))
				origins = append(origins, fieldOrigin)
				fieldIndex++
			}
		}
		if fieldIndex != structure.NumFields() {
			return MemberOriginSet{}, &Error{Subject: owner.Name(), Reason: "struct field occurrence denominator is incomplete"}
		}
	case *ast.InterfaceType:
		for _, field := range selected.Methods.List {
			for _, name := range field.Names {
				method, ok := context.TypesInfo().DefOf(name).(*types.Func)
				if !ok {
					return MemberOriginSet{}, &Error{Subject: owner.Name(), Reason: "interface method occurrence has no object"}
				}
				methodOrigin, err := AuthoredOrigin(context, field)
				if err != nil {
					return MemberOriginSet{}, err
				}
				if basis, bounded := origin.EnvironmentBasis(); bounded {
					methodOrigin, err = methodOrigin.WithEnvironmentBasis(basis)
					if err != nil {
						return MemberOriginSet{}, err
					}
				}
				objects = append(objects, method)
				origins = append(origins, methodOrigin)
			}
		}
	}
	if len(objects) == 0 {
		if structure, ok := owner.Type().Underlying().(*types.Struct); ok &&
			structure.NumFields() != 0 {
			for index := range structure.NumFields() {
				objects = append(objects, structure.Field(index))
				origins = append(origins, origin)
			}
		}
	}
	return NewMemberOriginSet(objects, origins)
}

func TypeDeclarationOrigin(
	context api.Context,
	source *ast.TypeSpec,
	origin DeclarationOrigin,
) (DeclarationOrigin, error) {
	if source == nil || !origin.Valid() {
		return DeclarationOrigin{}, &Error{Reason: "type declaration origin is invalid"}
	}
	basis := context.TypesInfo().TypeOf(source.Type)
	named, ok := types.Unalias(basis).(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return origin, nil
	}
	if named.Obj().Pkg() == context.TypesPackage() {
		return origin, nil
	}
	environmentOwned, err := context.Names().EnvironmentOwnedDeclaration(named.Obj())
	if err != nil {
		return DeclarationOrigin{}, err
	}
	if !environmentOwned {
		return origin, nil
	}
	contract, err := environmentcontract.Describe(named.Obj())
	if err != nil {
		return DeclarationOrigin{}, err
	}
	return origin.WithEnvironmentBasis(contract.Identity())
}
