package emit

import (
	"fmt"
	"go/types"
	"slices"
	"sort"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RootKind uint8

const (
	RootInvalid RootKind = iota
	RootRepresentationDemand
	RootFileCoverage
	RootExportedAPI
	RootConstantProjection
)

func (k RootKind) Valid() bool {
	return k == RootRepresentationDemand ||
		k == RootFileCoverage ||
		k == RootExportedAPI ||
		k == RootConstantProjection
}

type Root struct {
	object     types.Object
	kind       RootKind
	projection types.BasicKind
}

type RootError struct {
	Object string
	Reason string
}

func (e *RootError) Error() string {
	if e.Object == "" {
		return "create emission root: " + e.Reason
	}
	return fmt.Sprintf("create emission root for %q: %s", e.Object, e.Reason)
}

func NewRoot(object types.Object) (Root, error) {
	if selected, ok := object.(*types.Const); ok &&
		constantbinding.IsUntyped(selected.Type()) {
		return Root{}, &RootError{
			Object: selected.Name(),
			Reason: "untyped constant root requires an explicit concrete projection",
		}
	}
	return newRoot(object, RootRepresentationDemand)
}

func NewConstantProjectionRoot(
	selected *types.Const,
	projection types.BasicKind,
) (Root, error) {
	if selected == nil {
		return Root{}, &RootError{Reason: "constant is nil"}
	}
	if !constantbinding.IsUntyped(selected.Type()) {
		return Root{}, &RootError{
			Object: selected.Name(),
			Reason: "typed constant has one declaration and does not accept a projection root",
		}
	}
	if _, ok := api.ConstantProjectionType(projection); !ok {
		return Root{}, &RootError{
			Object: selected.Name(),
			Reason: "constant projection is not a concrete basic representation",
		}
	}
	root, err := newRoot(selected, RootConstantProjection)
	if err != nil {
		return Root{}, err
	}
	root.projection = projection
	return root, nil
}

func newRoot(object types.Object, kind RootKind) (Root, error) {
	if object == nil {
		return Root{}, &RootError{Reason: "object is nil"}
	}
	if !kind.Valid() {
		return Root{}, &RootError{
			Object: object.Name(),
			Reason: "root kind is invalid",
		}
	}
	if object.Pkg() == nil ||
		(object.Parent() != object.Pkg().Scope() && !isDeclaredMethod(object)) {
		return Root{}, &RootError{
			Object: object.Name(),
			Reason: "object is not a package declaration",
		}
	}
	return Root{object: object, kind: kind}, nil
}

func isDeclaredMethod(object types.Object) bool {
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiverType := signature.Recv().Type()
	if pointer, ok := types.Unalias(receiverType).(*types.Pointer); ok {
		receiverType = pointer.Elem()
	}
	named, ok := types.Unalias(receiverType).(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Pkg() == object.Pkg()
}

func (r Root) Object() types.Object {
	return r.object
}

func (r Root) Kind() RootKind {
	return r.kind
}

func (r Root) Projection() (types.BasicKind, bool) {
	_, valid := api.ConstantProjectionType(r.projection)
	return r.projection, r.kind == RootConstantProjection && valid
}

func (r Root) valid() bool {
	if r.object == nil || !r.kind.Valid() {
		return false
	}
	if r.kind == RootConstantProjection {
		_, ok := api.ConstantProjectionType(r.projection)
		return ok
	}
	return r.projection == types.Invalid
}

func compareRoots(
	left Root,
	right Root,
	compareOwners func(api.ArtifactOwner, api.ArtifactOwner) int,
) int {
	if order := compareOwners(
		api.MustSourceArtifactOwner(left.object),
		api.MustSourceArtifactOwner(right.object),
	); order != 0 {
		return order
	}
	if left.kind < right.kind {
		return -1
	}
	if left.kind > right.kind {
		return 1
	}
	return emitordering.CompareBasicKinds(left.projection, right.projection)
}

func ExportedAPIRoots(source *load.Package) ([]Root, error) {
	if source == nil || source.Types() == nil {
		return nil, &RootError{Reason: "source package is nil"}
	}
	scope := source.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	roots := make([]Root, 0, len(names))
	for _, name := range names {
		object := scope.Lookup(name)
		if object == nil || !object.Exported() {
			continue
		}
		root, err := newRoot(object, RootExportedAPI)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
		typeName, ok := object.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		for index := range named.NumMethods() {
			method := named.Method(index)
			if !method.Exported() {
				continue
			}
			methodRoot, err := newRoot(method, RootExportedAPI)
			if err != nil {
				return nil, err
			}
			roots = append(roots, methodRoot)
		}
	}
	sort.Slice(roots, func(left, right int) bool {
		return emitordering.CompareObjects(
			roots[left].object,
			roots[right].object,
		) < 0
	})
	return roots, nil
}

func (s *programSession) requireRoot(root Root) error {
	if !root.valid() {
		return &ScheduleError{Reason: "emission root is invalid"}
	}
	selected, isUntypedConstant := root.object.(*types.Const)
	if !isUntypedConstant || !constantbinding.IsUntyped(selected.Type()) {
		if root.kind == RootConstantProjection {
			return &ScheduleError{
				Object: root.object.Name(),
				Reason: "constant projection root does not own an untyped constant",
			}
		}
		return s.RequireUse(
			root.object,
			rootUseDemand(root.object),
			gostdlib.NoUseSelection(),
		)
	}
	if root.kind == RootFileCoverage || root.kind == RootExportedAPI {
		return s.RequireUse(
			root.object,
			rootUseDemand(root.object),
			gostdlib.NoUseSelection(),
		)
	}
	if root.kind != RootConstantProjection {
		return &ScheduleError{
			Object: selected.Name(),
			Reason: "untyped constant root has no concrete projection",
		}
	}
	requirement, err := api.NewConstantProjectionRequirement(
		selected,
		root.projection,
	)
	if err != nil {
		return err
	}
	return s.scheduleDeclarationRequirement(requirement)
}

func (s *programSession) verifyRootObligations(
	roots []Root,
	files []TargetFile,
) error {
	for _, root := range roots {
		function, genericFunction := root.object.(*types.Func)
		if root.kind == RootExportedAPI &&
			genericFunction &&
			function.Signature().Recv() == nil &&
			len(api.GenericDeclarationParameters(function)) != 0 {
			binding, ok := s.registry.Target(root.object)
			if !ok {
				return &ScheduleError{
					Object: root.object.Name(),
					Reason: "export root has no target binding",
				}
			}
			exports, ok := s.artifacts.ExportedBindings(
				api.MustSourceArtifactOwner(root.object),
			)
			if !ok || !slices.Contains(exports, binding.Name) {
				return &ScheduleError{
					Object: root.object.Name(),
					Reason: "export root has no exact source-facing target binding",
				}
			}
		}
		selected, ok := root.object.(*types.Const)
		if !ok ||
			!constantbinding.IsUntyped(selected.Type()) ||
			root.kind != RootConstantProjection {
			continue
		}
		binding, ok := s.registry.Target(selected)
		if !ok {
			return &ScheduleError{
				Object: selected.Name(),
				Reason: "constant projection root has no reserved target binding",
			}
		}
		name, err := api.ConstantProjectionName(
			binding.Name,
			root.projection,
		)
		if err != nil {
			return err
		}
		count := 0
		for _, file := range files {
			if file.kind != TargetFileSource ||
				file.outputPath != binding.SourcePath {
				continue
			}
			for _, statement := range file.sourceFile.Statements() {
				declaration, ok := statement.(tsgo.VariableStatement)
				if !ok {
					continue
				}
				for _, variable := range declaration.
					DeclarationList().
					Declarations() {
					identifier, ok := variable.Name().(tsgo.Identifier)
					if ok && identifier.Text() == name {
						count++
					}
				}
			}
		}
		if count != 1 {
			return &ScheduleError{
				Object: selected.Name(),
				Reason: fmt.Sprintf(
					"constant projection root %q materialized %d declarations, want one",
					name,
					count,
				),
			}
		}
	}
	return nil
}

type IntegerRepresentation = api.IntegerRepresentation
type EvaluationOrder = api.EvaluationOrder
type ConcurrencySemantics = api.ConcurrencySemantics

const (
	IntegerRepresentationInvalid       = api.IntegerRepresentationInvalid
	IntegerRepresentationNumber        = api.IntegerRepresentationNumber
	IntegerRepresentationBigInt        = api.IntegerRepresentationBigInt
	IntegerRepresentationFixed64BigInt = api.IntegerRepresentationFixed64BigInt

	EvaluationOrderInvalid    = api.EvaluationOrderInvalid
	EvaluationOrderDirect     = api.EvaluationOrderDirect
	EvaluationOrderPreserveGo = api.EvaluationOrderPreserveGo

	ConcurrencySemanticsDisabled    = api.ConcurrencySemanticsDisabled
	ConcurrencySemanticsCooperative = api.ConcurrencySemanticsCooperative
	ConcurrencySemanticsInvalid     = api.ConcurrencySemanticsInvalid
)

type Options struct {
	IntegerRepresentation IntegerRepresentation
	EvaluationOrder       EvaluationOrder
	ConcurrencySemantics  ConcurrencySemantics
	StandardLibrary       *gostdlibcertify.Certificate
	ExternalProvider      *externalcertify.Certificate
	SourceImplementations *sourceimplementation.Certificate
}

func DefaultOptions() Options {
	return Options{
		IntegerRepresentation: IntegerRepresentationNumber,
		EvaluationOrder:       EvaluationOrderDirect,
		ConcurrencySemantics:  ConcurrencySemanticsDisabled,
	}
}

func ParseIntegerRepresentation(value string) (IntegerRepresentation, error) {
	switch value {
	case IntegerRepresentationNumber.String():
		return IntegerRepresentationNumber, nil
	case IntegerRepresentationBigInt.String():
		return IntegerRepresentationBigInt, nil
	case IntegerRepresentationFixed64BigInt.String():
		return IntegerRepresentationFixed64BigInt, nil
	default:
		return IntegerRepresentationInvalid, &OptionsError{
			Field: "integer representation",
			Reason: fmt.Sprintf(
				"%q is not number, fixed64-bigint, or bigint",
				value,
			),
		}
	}
}

func ParseEvaluationOrder(value string) (EvaluationOrder, error) {
	switch value {
	case EvaluationOrderDirect.String():
		return EvaluationOrderDirect, nil
	case EvaluationOrderPreserveGo.String():
		return EvaluationOrderPreserveGo, nil
	default:
		return EvaluationOrderInvalid, &OptionsError{
			Field:  "evaluation order",
			Reason: fmt.Sprintf("%q is not direct or preserve-go", value),
		}
	}
}

func ParseConcurrencySemantics(value string) (ConcurrencySemantics, error) {
	switch value {
	case ConcurrencySemanticsDisabled.String():
		return ConcurrencySemanticsDisabled, nil
	case ConcurrencySemanticsCooperative.String():
		return ConcurrencySemanticsCooperative, nil
	default:
		return ConcurrencySemanticsInvalid, &OptionsError{
			Field:  "concurrency semantics",
			Reason: fmt.Sprintf("%q is not disabled or cooperative", value),
		}
	}
}

func (o Options) validate() error {
	if !o.IntegerRepresentation.Valid() {
		return &OptionsError{
			Field:  "integer representation",
			Reason: "value is invalid",
		}
	}
	if !o.EvaluationOrder.Valid() {
		return &OptionsError{
			Field:  "evaluation order",
			Reason: "value is invalid",
		}
	}
	if !o.ConcurrencySemantics.Valid() {
		return &OptionsError{
			Field:  "concurrency semantics",
			Reason: "value is invalid",
		}
	}
	if o.StandardLibrary != nil && !o.StandardLibrary.Valid() {
		return &OptionsError{
			Field:  "standard library",
			Reason: "certificate is invalid",
		}
	}
	if o.ExternalProvider != nil && !o.ExternalProvider.Valid() {
		return &OptionsError{
			Field:  "external provider",
			Reason: "certificate is invalid",
		}
	}
	if o.SourceImplementations != nil && !o.SourceImplementations.Valid() {
		return &OptionsError{
			Field:  "source implementations",
			Reason: "certificate is invalid",
		}
	}
	if o.SourceImplementations != nil && !o.SourceImplementations.SupportsCompilation(
		o.IntegerRepresentation.String(),
		o.EvaluationOrder.String(),
		o.ConcurrencySemantics.String(),
	) {
		return &OptionsError{
			Field:  "source implementations",
			Reason: "certificate compilation profile differs",
		}
	}
	return nil
}

type OptionsError struct {
	Field  string
	Reason string
}

func (e *OptionsError) Error() string {
	if e.Field == "" {
		return "validate compilation options: " + e.Reason
	}
	return fmt.Sprintf("validate compilation option %q: %s", e.Field, e.Reason)
}

type TargetFile struct {
	outputPath  string
	packageName string
	sourceFile  tsgo.SourceFile
	kind        TargetFileKind
}

type TargetFileKind uint8

const (
	TargetFileInvalid TargetFileKind = iota
	TargetFileSource
	TargetFilePackageState
	TargetFilePackageAssembly
	TargetFileProgramInitialization
	TargetFileSupport
	TargetFileEnvironmentContract
	TargetFileStandardLibraryConstantProjection
	TargetFileSourceImplementation
)

// rootUseDemand classifies the closed use demand created by requesting one
// emission root from the referenced object kind.
func rootUseDemand(object types.Object) environmentidentity.UseDemand {
	switch object.(type) {
	case *types.Func:
		return environmentidentity.UseDemandCallable
	case *types.TypeName:
		return environmentidentity.UseDemandTypeContract
	default:
		return environmentidentity.UseDemandValue
	}
}

func (e ProgramEmission) Files() []TargetFile {
	return slices.Clone(e.files)
}

// EnvironmentProfile is the complete exact identity of the settled
// environment evidence: build, compilation, provider, and pinned-target
// profiles, each content-addressed by a deterministic fingerprint.
type EnvironmentProfile = environmentcontract.Profile

// EnvironmentProfile binds the settled environment evidence to the exact
// selected build, compilation, provider, and pinned-target profiles.
func (e ProgramEmission) EnvironmentProfile() EnvironmentProfile {
	return e.environmentProfile
}

func (e ProgramEmission) EnvironmentObligations() []EnvironmentObligation {
	return slices.Clone(e.environmentObligations)
}

func (e ProgramEmission) ExternalFunctionObligations() []ExternalFunctionObligation {
	return slices.Clone(e.externalFunctionObligations)
}

func (e ProgramEmission) RuntimePackage() (RuntimePackage, bool) {
	return e.runtimePackage, e.runtimePackage.assembled.Valid()
}

func (e ProgramEmission) PackageDependencies() []PackageDependency {
	return slices.Clone(e.packageDependencies)
}

func (d PackageDependency) Name() string    { return d.name }
func (d PackageDependency) Version() string { return d.version }

func (p RuntimePackage) Name() string {
	return p.assembled.Name()
}

func (p RuntimePackage) Version() string {
	return p.assembled.Version()
}

func (p RuntimePackage) RootPath() string {
	return p.assembled.RootPath()
}

func (p RuntimePackage) ManifestPath() string {
	return p.assembled.ManifestPath()
}

func (p RuntimePackage) IntegerRepresentation() IntegerRepresentation {
	return p.assembled.Profile()
}

func (p RuntimePackage) ConcurrencySemantics() ConcurrencySemantics {
	return p.assembled.Concurrency()
}

func (p RuntimePackage) Manifest() []byte {
	return p.assembled.Manifest()
}

func (p RuntimePackage) Fingerprint() string {
	return p.assembled.Fingerprint()
}

func (f TargetFile) OutputPath() string {
	return f.outputPath
}

func (f TargetFile) PackageName() string {
	return f.packageName
}

func (f TargetFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

func (f TargetFile) Kind() TargetFileKind {
	return f.kind
}
