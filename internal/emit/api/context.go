package api

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Context struct {
	role            Role
	fileSet         *token.FileSet
	typesPackage    *types.Package
	typesInfo       *types.Info
	typesSizes      types.Sizes
	factory         tsgo.Factory
	names           Names
	expectedType    types.Type
	functionResults *types.Tuple
	breakDepth      uint32
	continueDepth   uint32
}

func NewContext(
	role Role,
	fileSet *token.FileSet,
	typesPackage *types.Package,
	typesInfo *types.Info,
	typesSizes types.Sizes,
	factory tsgo.Factory,
	names Names,
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
	}
	return Context{
		role:         role,
		fileSet:      fileSet,
		typesPackage: typesPackage,
		typesInfo:    typesInfo,
		typesSizes:   typesSizes,
		factory:      factory,
		names:        names,
	}, nil
}

func (c Context) WithRole(role Role) Context {
	c.role = role
	return c
}

func (c Context) WithExpectedType(expectedType types.Type) Context {
	c.expectedType = expectedType
	return c
}

func (c Context) EnterFunction(results *types.Tuple) Context {
	c.functionResults = results
	c.expectedType = nil
	c.breakDepth = 0
	c.continueDepth = 0
	return c
}

func (c Context) EnterLoop() Context {
	c.breakDepth++
	c.continueDepth++
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

func (c Context) ExpectedType() types.Type {
	return c.expectedType
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
