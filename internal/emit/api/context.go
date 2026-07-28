package api

import (
	"go/token"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Context struct {
	role                     Role
	fileSet                  *token.FileSet
	typesPackage             *types.Package
	typesInfo                *types.Info
	typesSizes               types.Sizes
	factory                  tsgo.Factory
	names                    Names
	values                   Values
	storage                  AddressableStorage
	integer                  IntegerRepresentation
	evaluationOrder          EvaluationOrder
	expectedType             types.Type
	expectedResults          *types.Tuple
	functionResults          *types.Tuple
	breakDepth               uint32
	continueDepth            uint32
	artifactOwner            *types.Func
	storageNames             map[*types.Var]string
	localConstantProjections map[*types.Const][]types.BasicKind
	lexicalAnonymousStructs  map[*types.TypeName][]DeclarationRequirement
}

func (c Context) WithAddressableStorage(
	owner *types.Func,
	storageNames map[*types.Var]string,
) Context {
	if owner == nil {
		panic("addressable-storage owner is nil")
	}
	c.artifactOwner = owner
	c.storageNames = make(map[*types.Var]string, len(storageNames))
	for variable, name := range storageNames {
		if variable == nil || variable.IsField() || name == "" {
			panic("addressable-storage selection is invalid")
		}
		c.storageNames[variable] = name
	}
	return c
}

func (c Context) WithLocalConstantProjections(
	owner *types.Func,
	projections map[*types.Const][]types.BasicKind,
) Context {
	if owner == nil || c.artifactOwner != owner {
		panic("local-constant projection owner differs from artifact owner")
	}
	c.localConstantProjections = make(
		map[*types.Const][]types.BasicKind,
		len(projections),
	)
	for selected, kinds := range projections {
		if selected == nil || len(kinds) == 0 {
			panic("local-constant projection selection is invalid")
		}
		c.localConstantProjections[selected] = slices.Clone(kinds)
	}
	return c
}

func (c Context) WithLexicalAnonymousStructs(
	owner ArtifactOwner,
	requirements map[*types.TypeName][]DeclarationRequirement,
) Context {
	if _, sourceOwned := owner.Source(); !sourceOwned {
		panic("lexical anonymous-struct owner is not source-backed")
	}
	c.lexicalAnonymousStructs = make(
		map[*types.TypeName][]DeclarationRequirement,
		len(requirements),
	)
	for anchor, selected := range requirements {
		if anchor == nil || len(selected) == 0 {
			panic("lexical anonymous-struct selection is invalid")
		}
		for _, requirement := range selected {
			artifact, _, ok := requirement.AnonymousStruct()
			if !ok ||
				artifact.Placement() !=
					GeneratedArtifactPlacementLexical ||
				artifact.LexicalOwner() != owner ||
				artifact.LexicalAnchor() != anchor {
				panic("lexical anonymous-struct requirement is inconsistent")
			}
		}
		c.lexicalAnonymousStructs[anchor] = slices.Clone(selected)
	}
	return c
}

func NewContext(
	role Role,
	fileSet *token.FileSet,
	typesPackage *types.Package,
	typesInfo *types.Info,
	typesSizes types.Sizes,
	factory tsgo.Factory,
	names Names,
	values Values,
	storage AddressableStorage,
	integer IntegerRepresentation,
	evaluationOrder EvaluationOrder,
) (Context, error) {
	switch {
	case role == "":
		return Context{}, &ContextError{Reason: "role is empty"}
	case fileSet == nil:
		return Context{}, &ContextError{Reason: "file set is nil"}
	case typesPackage == nil:
		return Context{}, &ContextError{Reason: "types package is nil"}
	case typesInfo == nil:
		return Context{}, &ContextError{Reason: "types info is nil"}
	case typesSizes == nil:
		return Context{}, &ContextError{Reason: "types sizes are nil"}
	case names == nil:
		return Context{}, &ContextError{Reason: "name owner is nil"}
	case values == nil:
		return Context{}, &ContextError{Reason: "value owner is nil"}
	case storage == nil:
		return Context{}, &ContextError{Reason: "addressable-storage owner is nil"}
	case !integer.Valid():
		return Context{}, &ContextError{Reason: "integer representation is invalid"}
	case !evaluationOrder.Valid():
		return Context{}, &ContextError{Reason: "evaluation order is invalid"}
	}
	return Context{
		role:            role,
		fileSet:         fileSet,
		typesPackage:    typesPackage,
		typesInfo:       typesInfo,
		typesSizes:      typesSizes,
		factory:         factory,
		names:           names,
		values:          values,
		storage:         storage,
		integer:         integer,
		evaluationOrder: evaluationOrder,
	}, nil
}

func (c Context) WithRole(role Role) Context {
	c.role = role
	return c
}

func (c Context) WithExpectedType(expectedType types.Type) Context {
	c.expectedType = expectedType
	c.expectedResults = nil
	return c
}

func (c Context) WithExpectedResults(expectedResults *types.Tuple) Context {
	if expectedResults == nil || expectedResults.Len() < 2 {
		panic("expected result tuple has fewer than two elements")
	}
	c.expectedType = nil
	c.expectedResults = expectedResults
	return c
}

func (c Context) EnterFunction(results *types.Tuple) Context {
	c.functionResults = results
	c.expectedType = nil
	c.expectedResults = nil
	c.breakDepth = 0
	c.continueDepth = 0
	return c
}

func (c Context) EnterLoop() Context {
	c.breakDepth++
	c.continueDepth++
	return c
}

func (c Context) EnterBreakable() Context {
	c.breakDepth++
	return c
}

func (c Context) Role() Role {
	return c.role
}

func (c Context) FileSet() *token.FileSet {
	return c.fileSet
}

func (c Context) TypesPackage() *types.Package {
	return c.typesPackage
}

func (c Context) TypesInfo() *types.Info {
	return c.typesInfo
}

func (c Context) TypesSizes() types.Sizes {
	return c.typesSizes
}

func (c Context) Factory() tsgo.Factory {
	return c.factory
}

func (c Context) Names() Names {
	return c.names
}

func (c Context) Values() Values {
	return c.values
}

func (c Context) AddressableStorage() AddressableStorage {
	return c.storage
}

func (c Context) IntegerRepresentation() IntegerRepresentation {
	return c.integer
}

func (c Context) EvaluationOrder() EvaluationOrder {
	return c.evaluationOrder
}

func (c Context) ExpectedType() types.Type {
	return c.expectedType
}

func (c Context) ExpectedResults() *types.Tuple {
	return c.expectedResults
}

func (c Context) FunctionResults() *types.Tuple {
	return c.functionResults
}

func (c Context) CanBreak() bool {
	return c.breakDepth != 0
}

func (c Context) CanContinue() bool {
	return c.continueDepth != 0
}

func (c Context) ArtifactOwner() *types.Func {
	return c.artifactOwner
}

func (c Context) LocalConstantProjections(
	selected *types.Const,
) []types.BasicKind {
	return slices.Clone(c.localConstantProjections[selected])
}

func (c Context) LexicalAnonymousStructs(
	anchor *types.TypeName,
) []DeclarationRequirement {
	return slices.Clone(c.lexicalAnonymousStructs[anchor])
}

func (c Context) AddressableStorageName(variable *types.Var) (string, bool) {
	if variable == nil {
		return "", false
	}
	name, ok := c.storageNames[variable]
	return name, ok
}
