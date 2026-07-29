package api

import (
	"fmt"
	"go/types"
)

type ArtifactOwner struct {
	source             types.Object
	initializerPackage *types.Package
	initializer        *types.Initializer
	generated          *GeneratedArtifact
}

func PackageInitializerArtifactOwner(
	sourcePackage *types.Package,
	initializer *types.Initializer,
) (ArtifactOwner, error) {
	if sourcePackage == nil ||
		initializer == nil ||
		initializer.Rhs == nil ||
		len(initializer.Lhs) == 0 {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "package initializer artifact owner is invalid",
		}
	}
	for _, variable := range initializer.Lhs {
		if !validPackageInitializerTarget(sourcePackage, variable) {
			return ArtifactOwner{}, &RootRequestError{
				Reason: "package initializer artifact owner has a foreign target",
			}
		}
	}
	return ArtifactOwner{
		initializerPackage: sourcePackage,
		initializer:        initializer,
	}, nil
}

func SourceArtifactOwner(source types.Object) (ArtifactOwner, error) {
	if source == nil {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "source artifact owner is nil",
		}
	}
	return ArtifactOwner{source: source}, nil
}

func (c Context) WithSourceArtifactOwner(
	owner ArtifactOwner,
) (Context, error) {
	source, ok := owner.Source()
	if !ok || source == nil {
		return Context{}, &ContextError{
			Reason: "source artifact owner is invalid",
		}
	}
	if existing, bound := c.artifactOwner.Source(); bound &&
		existing != source {
		return Context{}, &ContextError{
			Reason: "source artifact owner is already bound",
		}
	}
	c.artifactOwner = owner
	return c, nil
}

func GeneratedArtifactOwner(
	generated *GeneratedArtifact,
) (ArtifactOwner, error) {
	if !generated.Valid() {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "generated artifact owner is invalid",
		}
	}
	return ArtifactOwner{generated: generated}, nil
}

func MustSourceArtifactOwner(source types.Object) ArtifactOwner {
	owner, err := SourceArtifactOwner(source)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustPackageInitializerArtifactOwner(
	sourcePackage *types.Package,
	initializer *types.Initializer,
) ArtifactOwner {
	owner, err := PackageInitializerArtifactOwner(sourcePackage, initializer)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustGeneratedArtifactOwner(generated *GeneratedArtifact) ArtifactOwner {
	owner, err := GeneratedArtifactOwner(generated)
	if err != nil {
		panic(err)
	}
	return owner
}

func (o ArtifactOwner) Valid() bool {
	variants := 0
	if o.source != nil {
		variants++
	}
	if o.initializerPackage != nil || o.initializer != nil {
		if o.initializerPackage == nil ||
			o.initializer == nil ||
			o.initializer.Rhs == nil ||
			len(o.initializer.Lhs) == 0 {
			return false
		}
		for _, variable := range o.initializer.Lhs {
			if !validPackageInitializerTarget(
				o.initializerPackage,
				variable,
			) {
				return false
			}
		}
		variants++
	}
	if o.generated != nil {
		variants++
	}
	return variants == 1 &&
		(o.generated == nil || o.generated.Valid())
}

func (o ArtifactOwner) Source() (types.Object, bool) {
	return o.source, o.Valid() && o.source != nil
}

func (o ArtifactOwner) PackageInitializer() (
	*types.Package,
	*types.Initializer,
	bool,
) {
	return o.initializerPackage,
		o.initializer,
		o.Valid() && o.initializer != nil
}

func (o ArtifactOwner) Generated() (*GeneratedArtifact, bool) {
	return o.generated, o.Valid() && o.generated != nil
}

func (o ArtifactOwner) Package() *types.Package {
	if source, ok := o.Source(); ok {
		return source.Pkg()
	}
	if sourcePackage, _, ok := o.PackageInitializer(); ok {
		return sourcePackage
	}
	if generated, ok := o.Generated(); ok {
		return generated.LexicalOwner().Package()
	}
	return nil
}

func (o ArtifactOwner) Name() string {
	if source, ok := o.Source(); ok {
		return source.Name()
	}
	if generated, ok := o.Generated(); ok {
		return generated.TargetName()
	}
	if sourcePackage, initializer, ok := o.PackageInitializer(); ok {
		return sourcePackage.Path() + ".$init@" +
			fmt.Sprint(initializer.Rhs.Pos())
	}
	return ""
}

func validPackageInitializerTarget(
	sourcePackage *types.Package,
	variable *types.Var,
) bool {
	if sourcePackage == nil ||
		variable == nil ||
		variable.Pkg() != sourcePackage {
		return false
	}
	if variable.Name() == "_" {
		return variable.Parent() == nil
	}
	return variable.Parent() == sourcePackage.Scope()
}
