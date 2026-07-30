package api

import (
	"fmt"
	"go/types"
)

type ArtifactOwner struct {
	source             types.Object
	initializerPackage *types.Package
	initializer        *types.Initializer
	assemblyPackage    *types.Package
	generated          *GeneratedArtifact
}

func PackageAssemblyArtifactOwner(
	sourcePackage *types.Package,
) (ArtifactOwner, error) {
	if sourcePackage == nil || sourcePackage.Path() == "" {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "package assembly artifact owner is invalid",
		}
	}
	return ArtifactOwner{assemblyPackage: sourcePackage}, nil
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

func MustPackageAssemblyArtifactOwner(
	sourcePackage *types.Package,
) ArtifactOwner {
	owner, err := PackageAssemblyArtifactOwner(sourcePackage)
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
	if o.assemblyPackage != nil {
		if o.assemblyPackage.Path() == "" {
			return false
		}
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

func (o ArtifactOwner) PackageAssembly() (*types.Package, bool) {
	return o.assemblyPackage,
		o.Valid() && o.assemblyPackage != nil
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
	if sourcePackage, ok := o.PackageAssembly(); ok {
		return sourcePackage
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
	if sourcePackage, ok := o.PackageAssembly(); ok {
		return sourcePackage.Path() + ".$assembly"
	}
	return ""
}

func (c Context) WithArtifactOwner(owner ArtifactOwner) Context {
	if !owner.Valid() ||
		c.artifactOwner.Valid() && c.artifactOwner != owner {
		panic("target artifact context owner is invalid")
	}
	c.artifactOwner = owner
	return c
}

func (c Context) ArtifactOwner() ArtifactOwner {
	return c.artifactOwner
}

func (c Context) FunctionArtifactOwner() (*types.Func, bool) {
	source, sourceOwned := c.artifactOwner.Source()
	function, callable := source.(*types.Func)
	return function, sourceOwned && callable
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

type ArtifactFacet uint8

const (
	ArtifactFacetInvalid             ArtifactFacet = 0
	ArtifactFacetCallableSignature   ArtifactFacet = 1
	ArtifactFacetConstructorSurface  ArtifactFacet = 2
	ArtifactFacetInstanceTypeSurface ArtifactFacet = 3
	ArtifactFacetStaticSurface       ArtifactFacet = 4
	ArtifactFacetValueSurface        ArtifactFacet = 5
	ArtifactFacetExportSurface       ArtifactFacet = 6
	ArtifactFacetImplementation      ArtifactFacet = 7
)

func (f ArtifactFacet) Valid() bool {
	return f >= ArtifactFacetCallableSignature &&
		f <= ArtifactFacetImplementation
}

func (f ArtifactFacet) String() string {
	switch f {
	case ArtifactFacetCallableSignature:
		return "callable-signature"
	case ArtifactFacetConstructorSurface:
		return "constructor-surface"
	case ArtifactFacetInstanceTypeSurface:
		return "instance-type-surface"
	case ArtifactFacetStaticSurface:
		return "static-surface"
	case ArtifactFacetValueSurface:
		return "value-surface"
	case ArtifactFacetExportSurface:
		return "export-surface"
	case ArtifactFacetImplementation:
		return "implementation"
	default:
		return fmt.Sprintf("artifact-facet(%d)", f)
	}
}

type ArtifactDependency struct {
	provider ArtifactOwner
	facet    ArtifactFacet
}

func NewArtifactDependency(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (ArtifactDependency, error) {
	if !provider.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency provider is invalid"}
	}
	if !facet.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency facet is invalid"}
	}
	return ArtifactDependency{provider: provider, facet: facet}, nil
}

func (d ArtifactDependency) Valid() bool {
	return d.provider.Valid() && d.facet.Valid()
}

func (d ArtifactDependency) Provider() ArtifactOwner {
	return d.provider
}

func (d ArtifactDependency) Facet() ArtifactFacet {
	return d.facet
}

func NewArtifactDependencyRequest(
	provider types.Object,
	facet ArtifactFacet,
) (RootRequest, error) {
	owner, err := SourceArtifactOwner(provider)
	if err != nil {
		return RootRequest{}, err
	}
	return newArtifactDependencyRequest(owner, facet)
}

func NewGeneratedArtifactDependencyRequest(
	provider *GeneratedArtifact,
	facet ArtifactFacet,
) (RootRequest, error) {
	owner, err := GeneratedArtifactOwner(provider)
	if err != nil {
		return RootRequest{}, err
	}
	return newArtifactDependencyRequest(owner, facet)
}

func NewOwnedArtifactDependencyRequest(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (RootRequest, error) {
	return newArtifactDependencyRequest(provider, facet)
}

func newArtifactDependencyRequest(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (RootRequest, error) {
	dependency, err := NewArtifactDependency(provider, facet)
	if err != nil {
		return RootRequest{}, err
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:               RootRequestArtifactDependency,
			artifactDependency: dependency,
		},
	}}, nil
}
