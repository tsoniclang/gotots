package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

type ExternalFunctionObligation struct {
	identity     string
	function     *types.Func
	signature    *types.Signature
	role         api.Role
	position     token.Position
	buildProfile load.BuildProfile
}

func (o ExternalFunctionObligation) Identity() string {
	return o.identity
}

func (o ExternalFunctionObligation) Function() *types.Func {
	return o.function
}

func (o ExternalFunctionObligation) Signature() *types.Signature {
	return o.signature
}

func (o ExternalFunctionObligation) Role() api.Role {
	return o.role
}

func (o ExternalFunctionObligation) Position() token.Position {
	return o.position
}

func (o ExternalFunctionObligation) BuildProfile() load.BuildProfile {
	return o.buildProfile
}

func newExternalFunctionObligation(
	site declarationSite,
	function *types.Func,
	profile load.BuildProfile,
) (ExternalFunctionObligation, error) {
	declaration, ok := site.Declaration.(*ast.FuncDecl)
	if !ok || declaration.Body != nil || function == nil ||
		function != function.Origin() || !profile.Valid() {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks a bodyless source owner",
		}
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks an exact signature",
		}
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return ExternalFunctionObligation{}, err
	}
	position := site.Source.FileSet().Position(declaration.Pos())
	if contract.Identity() == "" || !position.IsValid() {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks canonical evidence",
		}
	}
	return ExternalFunctionObligation{
		identity:     contract.Identity(),
		function:     function,
		signature:    signature,
		role:         api.RoleFileDeclaration,
		position:     position,
		buildProfile: profile,
	}, nil
}

func (s *programSession) recordExternalFunctionObligation(
	site declarationSite,
	object types.Object,
) error {
	declaration, ok := site.Declaration.(*ast.FuncDecl)
	if !ok || declaration.Body != nil {
		return nil
	}
	function, ok := object.(*types.Func)
	if !ok {
		return &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "bodyless declaration is not a function",
		}
	}
	function = function.Origin()
	obligation, err := newExternalFunctionObligation(
		site,
		function,
		s.source.BuildProfile(),
	)
	if err != nil {
		return err
	}
	if existing, duplicate := s.externalFunctions[function]; duplicate {
		if existing.identity != obligation.identity ||
			existing.position != obligation.position {
			return &api.InvariantError{
				Role:   api.RoleFileDeclaration,
				Reason: "external function obligation identity is inconsistent",
			}
		}
		return nil
	}
	s.externalFunctions[function] = obligation
	return nil
}

func (s *programSession) externalFunctionObligations() []ExternalFunctionObligation {
	result := make(
		[]ExternalFunctionObligation,
		0,
		len(s.externalFunctions),
	)
	for _, obligation := range s.externalFunctions {
		result = append(result, obligation)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].identity != result[right].identity {
			return result[left].identity < result[right].identity
		}
		return result[left].position.String() < result[right].position.String()
	})
	return slices.Clone(result)
}
