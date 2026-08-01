package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	ArtifactFacetImplementation      ArtifactFacet = 6
	ArtifactFacetExportSurface       ArtifactFacet = 7
)

func (f ArtifactFacet) Valid() bool {
	return f >= ArtifactFacetCallableSignature &&
		f <= ArtifactFacetExportSurface
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

type NameReference struct {
	qualifier string
	name      string
	requests  []RootRequest
	provider  bool
}

func NewNameReference(name string, requests ...RootRequest) (NameReference, error) {
	if name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	return NameReference{name: name, requests: slices.Clone(requests)}, nil
}

func NewQualifiedNameReference(
	qualifier string,
	name string,
	requests ...RootRequest,
) (NameReference, error) {
	switch {
	case qualifier == "":
		return NameReference{}, &NameError{
			Name:   name,
			Reason: "reference qualifier is empty",
		}
	case name == "":
		return NameReference{}, &NameError{
			Reason: "reference name is empty",
		}
	}
	return NameReference{
		qualifier: qualifier,
		name:      name,
		requests:  slices.Clone(requests),
	}, nil
}

func NewProviderQualifiedNameReference(
	qualifier string,
	name string,
	requests ...RootRequest,
) (NameReference, error) {
	reference, err := NewQualifiedNameReference(
		qualifier,
		name,
		requests...,
	)
	if err != nil {
		return NameReference{}, err
	}
	reference.provider = true
	return reference, nil
}

func (r NameReference) Name() string {
	return r.name
}

func (r NameReference) Qualifier() (string, bool) {
	return r.qualifier, r.qualifier != ""
}

func (r NameReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

func (r NameReference) ProviderBoundary() bool {
	return r.provider
}

func (r NameReference) WithRequests(
	requests ...RootRequest,
) (NameReference, error) {
	if r.name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	r.requests = slices.Clone(requests)
	return r, nil
}

func (r NameReference) Expression(factory tsgo.Factory) tsgo.Expression {
	if r.qualifier == "" {
		return factory.Identifier(r.name)
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(r.qualifier),
		nil,
		factory.Identifier(r.name),
		tsgo.NodeFlagsNone,
	)
}

func (r NameReference) EntityName(factory tsgo.Factory) tsgo.EntityName {
	if r.qualifier == "" {
		return factory.Identifier(r.name)
	}
	return factory.QualifiedName(
		factory.Identifier(r.qualifier),
		factory.Identifier(r.name),
	)
}

func (r NameReference) MemberExpression(
	factory tsgo.Factory,
	member string,
) (tsgo.PropertyAccessExpression, error) {
	if member == "" {
		return nil, &NameError{
			Name:   r.name,
			Reason: "reference member is empty",
		}
	}
	return factory.PropertyAccessExpression(
		r.Expression(factory),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	), nil
}

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
