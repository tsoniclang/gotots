package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// ProbeResult measures how much of the real corpus the current reviewed
// subset IR-admits (constructs typed IR for), and ranks exactly what
// blocks the rest. It is diagnostic evidence for roadmap ordering — never
// authoritative generation, which requires complete package coverage AND
// executed emission. IR admission never implies emission, typechecking, or
// usability.
type ProbeResult struct {
	// Diagnostic marks this report as a development probe, never
	// acceptance evidence: it carries no body ledger or manifest and is
	// not gate output.
	Diagnostic bool `json:"diagnostic"`
	// SourceRevision and ProfileHash attest the probed inputs.
	SourceRevision string `json:"sourceRevision,omitempty"`
	ProfileHash    string `json:"profileHash,omitempty"`
	Packages       int    `json:"packages"`
	Bodies         int    `json:"bodies"`
	// IRAdmitted counts bodies with complete typed semantic IR — the
	// ir-admitted evidence stage. It is NOT module-retained coverage: a
	// body can be ir-admitted yet removed from every runnable module by
	// package withholding. See ModuleRetainedPackages / WithheldPackages
	// and spec 00 Translation Evidence Stages.
	IRAdmitted       int            `json:"irAdmitted"`
	Blocked          int            `json:"blocked"`
	BlockerHistogram map[string]int `json:"blockerHistogram"`
	// ConstructHistogram counts the raw (unnormalized) construct
	// spellings, ranking the exact shapes inside each blocker class.
	ConstructHistogram map[string]int `json:"constructHistogram"`
	// PackagesIRDeclComplete lists packages that are IR/declaration-complete
	// CANDIDATES: every declaration (bodies plus declaration-level
	// initializers/types/vars) constructed its typed IR. This is an ANALYSIS
	// candidacy, NOT emission — the deferred emitters are not executed here,
	// so these packages are not proven emitted, typechecked, or usable.
	PackagesIRDeclComplete []string `json:"packagesIRDeclComplete"`
	// PackagesBodyOnly lists packages where every body builds but a
	// declaration-level construct (initializer, type, var) still blocks
	// IR/declaration completeness, with the blocking diagnostic.
	PackagesBodyOnly []string `json:"packagesBodyOnly"`
	// PerPackage maps package path -> ir-admitted/total bodies.
	PerPackage map[string]string `json:"perPackage"`
	// PackageBlockers lists, for packages close to full body coverage
	// (at most three blocked bodies), each blocked body's sites — the
	// highest-leverage targets for package-level completion.
	PackageBlockers map[string][]string `json:"packageBlockers"`
	// ExternalRefs counts blocked references per external function or
	// method (pkg.Name) — the evidence ranking emulation-layer priorities.
	ExternalRefs map[string]int `json:"externalRefs"`
	// PerBodyState maps each body's canonical identity to its probe
	// classification ("ir-admitted" or "unimplemented"), the join key for
	// reconciling probe results against corpus support ledgers.
	PerBodyState map[string]string `json:"perBodyState"`
	// ClassInventory summarizes every unsupported class blocking the
	// corpus: its disposition category (the kind of solution it needs),
	// the shared root abstraction that would clear the whole class, and
	// the exact unique-unit and raw-site counts — the reusable-solution
	// roadmap, so no repeated class is silently routed to manual bodies.
	ClassInventory []ClassRow `json:"classInventory"`
	// UnimplementedUnits is one machine-readable row per unimplemented
	// body, carrying EVERY unsupported site (not only the first) with its
	// classification — the exhaustive residual accounting.
	UnimplementedUnits []UnitInventory `json:"unimplementedUnits"`
	// kindByClass joins each class key (Kind.String()) back to its
	// producer-owned Kind, so the class inventory dispositions by Kind
	// rather than re-parsing the class string.
	kindByClass map[string]ir.UnsupportedKind
}

// ClassRow is one unsupported class's residual-inventory summary.
type ClassRow struct {
	Class    string `json:"class"`
	Category string `json:"category"`
	Root     string `json:"rootAbstraction"`
	Units    int    `json:"units"`
	Sites    int    `json:"sites"`
}

// UnitInventory is one unimplemented body with every unsupported site.
type UnitInventory struct {
	ID    string          `json:"id"`
	Sites []InventorySite `json:"sites"`
}

// InventorySite is one classified unsupported site.
type InventorySite struct {
	Class     string `json:"class"`
	Category  string `json:"category"`
	Construct string `json:"construct"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

// disposition names the KIND of solution a class of unsupported site
// requires and the shared root abstraction that would clear it.
type disposition struct{ category, root string }

// Disposition categories — the closed set the inventory reports.
const (
	catLanguageLowering  = "language-lowering"
	catRepresentation    = "representation-planning"
	catExternalContract  = "external-typed-contract"
	catProductPolicy     = "product-policy"
	catUnclassified      = "unclassified"
	rootUnclassifiedNote = "UNCLASSIFIED: this construct has no reviewed disposition; classify it explicitly rather than defaulting it"
)

// Shared root abstractions each category's clearing work threads through.
const (
	rootLangType     = "type universe: reviewed lowering for the remaining named-type forms"
	rootLangGenerics = "generic value semantics: per-instantiation copy operation threading"
	rootLangConvert  = "type conversion: same-underlying and generic-parameter conversion lowering"
	rootLangPointer  = "pointer cells: stable storage for the remaining address-taken locations"
	rootLangDispatch = "interface dispatch: closed-union lowering for the remaining boxing/assertion forms"
	rootLangDecl     = "local declarations: on-the-fly type-universe registration"
	rootLangControl  = "control flow: reviewed lowering for the remaining statement forms"
	rootLangBuiltin  = "builtin lowering: reviewed carrier operations for the remaining builtins"
	rootRepresent    = "keyed-map: per-instantiation static key encoding (uniform keyed carrier)"
	rootExternal     = "external contract surface: typed obligation/stub, never a manual caller body"
	rootConcurrency  = "concurrency: one explicit TS-Go concurrency product policy"
)

// dispositionByKind is the TOTAL classification of the closed
// ir.UnsupportedKind enum. Every producer-owned kind maps to an
// explicitly reviewed category and root abstraction — there is no
// substring matching and no ordered-prefix shadow. A kind absent here
// is caught by TestDispositionByKindTotal, never silently defaulted.
var dispositionByKind = map[ir.UnsupportedKind]disposition{
	ir.KindAddressOf:                               {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfALoopClauseVariable:            {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfANamedResult:                   {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfANonAddressableExpression:      {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfARangeVariable:                 {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfATupleBoundVariable:            {catLanguageLowering, rootLangPointer},
	ir.KindAddressOfATypeSwitchVariable:            {catLanguageLowering, rootLangPointer},
	ir.KindAliasDeclaration:                        {catLanguageLowering, rootLangType},
	ir.KindAppendOfFixedArrayElements:              {catLanguageLowering, rootLangBuiltin},
	ir.KindAppendTo:                                {catLanguageLowering, rootLangBuiltin},
	ir.KindAssignmentArityMismatch:                 {catLanguageLowering, rootLangControl},
	ir.KindAssignmentThrough:                       {catLanguageLowering, rootLangControl},
	ir.KindAssignmentTo:                            {catLanguageLowering, rootLangControl},
	ir.KindAssignmentToNonFieldSelector:            {catLanguageLowering, rootLangControl},
	ir.KindAssignmentToNonVariable:                 {catLanguageLowering, rootLangControl},
	ir.KindAssignmentToken:                         {catLanguageLowering, rootLangControl},
	ir.KindBasicType:                               {catLanguageLowering, rootLangType},
	ir.KindBlankImportInitSideEffects:              {catLanguageLowering, rootLangType},
	ir.KindBlankNamedResult:                        {catLanguageLowering, rootLangType},
	ir.KindBlankSlotArity:                          {catLanguageLowering, rootLangType},
	ir.KindBlankSlotInUnrecognizedTupleForm:        {catLanguageLowering, rootLangType},
	ir.KindBlankStructField:                        {catLanguageLowering, rootLangType},
	ir.KindBlankVariableDeclaration:                {catLanguageLowering, rootLangType},
	ir.KindBodylessFunction:                        {catLanguageLowering, rootLangType},
	ir.KindBranch:                                  {catLanguageLowering, rootLangControl},
	ir.KindBreakInsideATypeSwitchClause:            {catLanguageLowering, rootLangControl},
	ir.KindBuiltin:                                 {catLanguageLowering, rootLangBuiltin},
	ir.KindBuiltinStatement:                        {catLanguageLowering, rootLangBuiltin},
	ir.KindCallOf:                                  {catLanguageLowering, rootLangDispatch},
	ir.KindCallOutsideTheTranslatedUnit:            {catExternalContract, rootExternal},
	ir.KindCallOutsideTheTranslatedUnitUnqualified: {catExternalContract, rootExternal},
	ir.KindCallToAGenericExternalMethod:            {catExternalContract, rootExternal},
	ir.KindCallWithoutSignatureEvidence:            {catLanguageLowering, rootLangType},
	ir.KindCapOf:                                   {catLanguageLowering, rootLangBuiltin},
	ir.KindChannelSendStatement:                    {catProductPolicy, rootConcurrency},
	ir.KindChannelType:                             {catProductPolicy, rootConcurrency},
	ir.KindClearOf:                                 {catLanguageLowering, rootLangBuiltin},
	ir.KindCompositeLiteralOf:                      {catExternalContract, rootExternal},
	ir.KindCompoundAssignmentArity:                 {catLanguageLowering, rootLangControl},
	ir.KindCompoundAssignmentOn:                    {catLanguageLowering, rootLangControl},
	ir.KindCompoundAssignmentToTheBlankIdentifier:  {catLanguageLowering, rootLangControl},
	ir.KindConstantOfType:                          {catLanguageLowering, rootLangType},
	ir.KindConversionAsStatement:                   {catLanguageLowering, rootLangControl},
	ir.KindConversionFrom:                          {catLanguageLowering, rootLangConvert},
	ir.KindConversionFromUntypedNilTo:              {catLanguageLowering, rootLangConvert},
	ir.KindCopyBetween:                             {catLanguageLowering, rootLangBuiltin},
	ir.KindCopyOfFixedArrayElements:                {catLanguageLowering, rootLangBuiltin},
	ir.KindDeferBelowTheFunctionSTopLevelBlockRunsAtFunctionExitNeedsTheDeferStackLowering: {catLanguageLowering, rootLangControl},
	ir.KindDeferInAFunctionWithNamedResultsDeferredResultMutation:                          {catLanguageLowering, rootLangControl},
	ir.KindDeferredNonCallExpression:                                                       {catLanguageLowering, rootLangControl},
	ir.KindDereferenceOf:                                                                   {catLanguageLowering, rootLangPointer},
	ir.KindEqualityBetweenAnInterfaceAnd:                                                   {catLanguageLowering, rootLangDispatch},
	ir.KindEqualityOn:                                                                      {catLanguageLowering, rootLangDispatch},
	ir.KindEqualityOnArrayOf:                                                               {catLanguageLowering, rootLangDispatch},
	ir.KindEqualityPlanFor:                                                                 {catLanguageLowering, rootLangDispatch},
	ir.KindEqualityPlanForExternal:                                                         {catLanguageLowering, rootLangDispatch},
	ir.KindExpressionStatement:                                                             {catLanguageLowering, rootLangControl},
	ir.KindExpressionWithoutTypeEvidence:                                                   {catLanguageLowering, rootLangType},
	ir.KindFieldAccessOn:                                                                   {catExternalContract, rootExternal},
	ir.KindFieldAssignmentOn:                                                               {catLanguageLowering, rootLangControl},
	ir.KindFloat32Arithmetic:                                                               {catLanguageLowering, rootLangType},
	ir.KindFullSliceExpressionOn:                                                           {catLanguageLowering, rootLangBuiltin},
	ir.KindFunctionWithoutTypedDefinition:                                                  {catLanguageLowering, rootLangType},
	ir.KindGenericCall:                                                                     {catLanguageLowering, rootLangGenerics},
	ir.KindGenericCallInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation: {catLanguageLowering, rootLangGenerics},
	ir.KindGenericCallWithoutInstantiationEvidence:                                       {catLanguageLowering, rootLangGenerics},
	ir.KindGenericExternalMethodCall:                                                     {catExternalContract, rootExternal},
	ir.KindGenericFunctionInstantiatedWithAStructValueCopySemanticsVaryPerInstantiation:  {catLanguageLowering, rootLangGenerics},
	ir.KindGenericFunctionInstantiatedWithAnUnreviewedTypeArgument:                       {catLanguageLowering, rootLangGenerics},
	ir.KindGenericFunctionType:                                                           {catLanguageLowering, rootLangGenerics},
	ir.KindGenericMethodCall:                                                             {catLanguageLowering, rootLangGenerics},
	ir.KindGenericMethodExpression:                                                       {catLanguageLowering, rootLangGenerics},
	ir.KindGenericMethodValue:                                                            {catLanguageLowering, rootLangGenerics},
	ir.KindGenericTypeInstantiatedWithAValueCopyCarrierCopySemanticsVaryPerInstantiation: {catLanguageLowering, rootLangGenerics},
	ir.KindGenericTypeInstantiatedWithAnUnreviewedTypeArgument:                           {catExternalContract, rootExternal},
	ir.KindGoroutineStatement:                                                            {catProductPolicy, rootConcurrency},
	ir.KindIdentifier:                                                                    {catLanguageLowering, rootLangType},
	ir.KindIncDecOf:                                                                      {catLanguageLowering, rootLangType},
	ir.KindIndexOn:                                                                       {catLanguageLowering, rootLangType},
	ir.KindIndexedAssignmentOn:                                                           {catLanguageLowering, rootLangType},
	ir.KindInterfaceMethodExpression:                                                     {catLanguageLowering, rootLangDispatch},
	ir.KindInterfaceValueOfAnInstantiatedGenericType:                                     {catLanguageLowering, rootLangType},
	ir.KindInterfaceValueOfType:                                                          {catLanguageLowering, rootLangDispatch},
	ir.KindKeyedArrayLiteral:                                                             {catLanguageLowering, rootLangType},
	ir.KindKeyedSliceLiteral:                                                             {catLanguageLowering, rootLangType},
	ir.KindLabelOnANonLoopStatement:                                                      {catLanguageLowering, rootLangControl},
	ir.KindLabelOnARangeOverFuncLoop:                                                     {catLanguageLowering, rootLangControl},
	ir.KindLabeledBranchInsideARangeOverFuncBody:                                         {catLanguageLowering, rootLangControl},
	ir.KindLenOf:                 {catLanguageLowering, rootLangBuiltin},
	ir.KindMakeOf:                {catLanguageLowering, rootLangBuiltin},
	ir.KindMapKeyType:            {catRepresentation, rootRepresent},
	ir.KindMapLiteralWithoutKeys: {catLanguageLowering, rootLangType},
	ir.KindMethodCallOn:          {catLanguageLowering, rootLangDispatch},
	ir.KindMethodCallOutsideTheTranslatedUnitUnqualified:                {catExternalContract, rootExternal},
	ir.KindMethodCallWithoutSignatureEvidence:                           {catLanguageLowering, rootLangDispatch},
	ir.KindMethodExpressionOnAnUnnamedReceiverType:                      {catLanguageLowering, rootLangDispatch},
	ir.KindMethodExpressionOutsideTheTranslatedUnit:                     {catExternalContract, rootExternal},
	ir.KindMethodOnUnnamedReceiverType:                                  {catLanguageLowering, rootLangType},
	ir.KindMethodValueBindTimeReceiverCapture:                           {catLanguageLowering, rootLangDispatch},
	ir.KindMethodValueOn:                                                {catLanguageLowering, rootLangDispatch},
	ir.KindMethodValueOnAnUnnamedReceiverType:                           {catLanguageLowering, rootLangDispatch},
	ir.KindMethodValueOutsideTheTranslatedUnit:                          {catExternalContract, rootExternal},
	ir.KindMethodWithoutCanonicalIdentity:                               {catLanguageLowering, rootLangType},
	ir.KindMethodWithoutCanonicalSlot:                                   {catLanguageLowering, rootLangType},
	ir.KindMixedKeyedAndPositionalLiteral:                               {catLanguageLowering, rootLangType},
	ir.KindMultiResultCallInExpressionPosition:                          {catLanguageLowering, rootLangControl},
	ir.KindMultiResultForwardingIntoAVariadicCall:                       {catLanguageLowering, rootLangControl},
	ir.KindMultiValueVarInitializer:                                     {catLanguageLowering, rootLangType},
	ir.KindNestedError:                                                  {catLanguageLowering, rootLangType},
	ir.KindNewOf:                                                        {catLanguageLowering, rootLangBuiltin},
	ir.KindNilComparisonOn:                                              {catLanguageLowering, rootLangDispatch},
	ir.KindNilOfType:                                                    {catLanguageLowering, rootLangType},
	ir.KindNonFieldSelector:                                             {catLanguageLowering, rootLangType},
	ir.KindNonIntegralIntegerConstant:                                   {catLanguageLowering, rootLangType},
	ir.KindNonStructNamedType:                                           {catLanguageLowering, rootLangType},
	ir.KindNonValueVarSpec:                                              {catLanguageLowering, rootLangType},
	ir.KindNonVarDeclarationStatement:                                   {catLanguageLowering, rootLangControl},
	ir.KindOperator:                                                     {catLanguageLowering, rootLangType},
	ir.KindOrderingOn:                                                   {catLanguageLowering, rootLangType},
	ir.KindPackageLevel:                                                 {catLanguageLowering, rootLangDecl},
	ir.KindPackageLevelMultiValueVarInitializer:                         {catLanguageLowering, rootLangDecl},
	ir.KindPackageLevelMultiVariableInitializer:                         {catLanguageLowering, rootLangDecl},
	ir.KindPanicWith:                                                    {catLanguageLowering, rootLangControl},
	ir.KindPointerReceiverMethodCallOn:                                  {catLanguageLowering, rootLangDispatch},
	ir.KindPointerReceiverMethodValueOn:                                 {catExternalContract, rootExternal},
	ir.KindPointerToNonNamedType:                                        {catLanguageLowering, rootLangPointer},
	ir.KindPointerToNonStructType:                                       {catLanguageLowering, rootLangPointer},
	ir.KindPointerToTypeOutsideTheTranslatedUnit:                        {catExternalContract, rootExternal},
	ir.KindPromotedGenericMethod:                                        {catLanguageLowering, rootLangDispatch},
	ir.KindPromotedMethodFromATypeOutsideTheTranslatedUnit:              {catExternalContract, rootExternal},
	ir.KindPromotedMethodWithoutCanonicalIdentity:                       {catLanguageLowering, rootLangType},
	ir.KindPromotedSelectionThrough:                                     {catLanguageLowering, rootLangDispatch},
	ir.KindPromotionThroughANonStructEmbedding:                          {catLanguageLowering, rootLangDispatch},
	ir.KindPromotionThroughAnEmbeddedPointer:                            {catLanguageLowering, rootLangDispatch},
	ir.KindPromotionThroughAnUnnamedEmbedding:                           {catLanguageLowering, rootLangDispatch},
	ir.KindRangeOver:                                                    {catLanguageLowering, rootLangControl},
	ir.KindRangeOverAnIntegerWithASecondVariable:                        {catLanguageLowering, rootLangControl},
	ir.KindRangeVariableIsNotAnIdentifier:                               {catLanguageLowering, rootLangControl},
	ir.KindRangeWithAssignmentForm:                                      {catLanguageLowering, rootLangControl},
	ir.KindReferenceToAFunctionOutsideTheTranslatedUnit:                 {catExternalContract, rootExternal},
	ir.KindResliceOf:                                                    {catLanguageLowering, rootLangType},
	ir.KindReturnArityMismatch:                                          {catLanguageLowering, rootLangControl},
	ir.KindRuntimeTypeIdentityOf:                                        {catExternalContract, rootExternal},
	ir.KindSelectStatement:                                              {catProductPolicy, rootConcurrency},
	ir.KindShortDeclarationArityMismatch:                                {catLanguageLowering, rootLangControl},
	ir.KindShortDeclarationOfNonIdentifier:                              {catLanguageLowering, rootLangControl},
	ir.KindShortDeclarationReusingANonVariable:                          {catLanguageLowering, rootLangControl},
	ir.KindShortDeclarationReusingAnExistingVariableWithoutATupleSource: {catLanguageLowering, rootLangControl},
	ir.KindStoreIntoASliceOfExternalValues:                              {catExternalContract, rootExternal},
	ir.KindStoreIntoAnArrayOfExternalValues:                             {catExternalContract, rootExternal},
	ir.KindStructType:                                                   {catLanguageLowering, rootLangType},
	ir.KindSwitchCaseOf:                                                 {catLanguageLowering, rootLangControl},
	ir.KindSwitchTagOf:                                                  {catLanguageLowering, rootLangControl},
	ir.KindTwoRangeVariablesOverAOneValueSequence:                       {catLanguageLowering, rootLangControl},
	ir.KindType:               {catLanguageLowering, rootLangType},
	ir.KindTypeAssertionOn:    {catLanguageLowering, rootLangDispatch},
	ir.KindTypeInCallPosition: {catLanguageLowering, rootLangType},
	ir.KindTypeSwitchClauseWithAnInterfaceTypeMethodSetTest: {catLanguageLowering, rootLangType},
	ir.KindTypeSwitchGuardForm:                              {catLanguageLowering, rootLangType},
	ir.KindTypeSwitchOn:                                     {catLanguageLowering, rootLangDispatch},
	ir.KindTypeWithoutTypedDefinition:                       {catLanguageLowering, rootLangType},
	ir.KindUnaryOperator:                                    {catLanguageLowering, rootLangType},
	ir.KindUnrecognizedExpression:                           {catLanguageLowering, rootLangType},
	ir.KindUnrecognizedStatement:                            {catLanguageLowering, rootLangType},
	ir.KindUntypedNilOutsideATypedContext:                   {catLanguageLowering, rootLangType},
	ir.KindVarWithoutTypedDefinition:                        {catLanguageLowering, rootLangType},
	ir.KindVariadicParameterIsNotASlice:                     {catLanguageLowering, rootLangType},
	ir.KindZeroValueOf:                                      {catLanguageLowering, rootLangType},
}

// classifySite maps one producer-owned kind to its reviewed disposition
// via an exhaustive Kind-keyed table — never substring or ordered-prefix
// matching on the diagnostic text (so "type switch on ..." can never be
// read as the "type" family). A kind absent from the table classifies as
// "unclassified", surfaced honestly rather than defaulted to a lowering;
// TestDispositionByKindTotal proves the table is total over the enum.
func classifySite(kind ir.UnsupportedKind) (category, root string) {
	if d, ok := dispositionByKind[kind]; ok {
		return d.category, d.root
	}
	return catUnclassified, rootUnclassifiedNote
}

// Probe loads the owned corpus under the profile and attempts IR building
// for every production function body.
func Probe(prof *profile.Profile, env []string, sourceDir string) (*ProbeResult, error) {
	loaded, err := typedload.Load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}

	result := &ProbeResult{
		Diagnostic:         true,
		PerBodyState:       map[string]string{},
		SourceRevision:     prof.Pin.Revision,
		ProfileHash:        prof.Hash,
		BlockerHistogram:   map[string]int{},
		ConstructHistogram: map[string]int{},
		PerPackage:         map[string]string{},
		PackageBlockers:    map[string][]string{},
		ExternalRefs:       map[string]int{},
		kindByClass:        map[string]ir.UnsupportedKind{},
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID < loaded[j].ID })

	// The unit is every owned production package: cross-package references
	// among them resolve to co-generated modules.
	owned := func(p *packages.Package) bool {
		if typedload.RoleOf(p, sourceDir) != typedload.RoleProduction {
			return false
		}
		class, _ := prof.Classify(p.PkgPath)
		return class == profile.ClassOwned
	}
	var ownedPaths []string
	var ownedPackages []*packages.Package
	for _, p := range loaded {
		if owned(p) {
			ownedPaths = append(ownedPaths, p.PkgPath)
			ownedPackages = append(ownedPackages, p)
		}
	}
	unit := ir.NewScope(ownedPaths...)
	if err := collectGenericInstances(unit, ownedPackages); err != nil {
		return nil, err
	}

	for _, p := range loaded {
		if !owned(p) {
			continue
		}
		result.Packages++
		packageBodies, packageIRAdmitted := 0, 0
		var packageSites []string

		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			relative, relErr := filepath.Rel(sourceDir, filename)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				continue
			}
			source, readErr := os.ReadFile(filename)
			if readErr != nil {
				return nil, readErr
			}
			for _, decl := range file.Decls {
				funcDecl, isFunc := decl.(*ast.FuncDecl)
				if !isFunc || funcDecl.Body == nil {
					continue
				}
				result.Bodies++
				packageBodies++
				function, err := probeFunc(p, sourceDir, unit, source, funcDecl)
				if err != nil {
					return nil, err
				}
				result.PerBodyState[function.ID] = string(function.Support)
				if function.Support == ir.SupportUnimplemented {
					result.Blocked++
					unitInv := UnitInventory{ID: function.ID}
					for _, site := range function.Sites {
						result.BlockerHistogram[site.Class]++
						result.ConstructHistogram[site.Construct]++
						result.kindByClass[site.Class] = site.Kind
						category, _ := classifySite(site.Kind)
						unitInv.Sites = append(unitInv.Sites, InventorySite{
							Class: site.Class, Category: category, Construct: site.Construct,
							File: site.Span.File, Line: site.Span.Line})
						packageSites = append(packageSites,
							fmt.Sprintf("%s: %s (%s:%d)", function.ID, site.Construct, site.Span.File, site.Span.Line))
						if ref, isExternal := externalRefOfSite(site); isExternal {
							result.ExternalRefs[ref]++
						}
					}
					result.UnimplementedUnits = append(result.UnimplementedUnits, unitInv)
				} else {
					result.IRAdmitted++
					packageIRAdmitted++
				}
			}
		}
		if packageBodies > 0 {
			result.PerPackage[p.PkgPath] = fmt.Sprintf("%d/%d", packageIRAdmitted, packageBodies)
			if packageIRAdmitted == packageBodies {
				result.PackagesIRDeclComplete = append(result.PackagesIRDeclComplete, p.PkgPath)
			}
			if blocked := packageBodies - packageIRAdmitted; blocked > 0 && blocked <= 3 {
				result.PackageBlockers[p.PkgPath] = packageSites
			}
		}
	}
	sort.Strings(result.PackagesIRDeclComplete)

	// Body IR alone is not package translatability: initializers, type
	// declarations, and package variables all count. Every candidate is
	// verified by actually translating the package.
	byPath := map[string]*packages.Package{}
	for _, p := range loaded {
		if owned(p) {
			byPath[p.PkgPath] = p
		}
	}
	verified := make([]string, 0, len(result.PackagesIRDeclComplete))
	for _, candidate := range result.PackagesIRDeclComplete {
		throwaway := &Generated{Files: map[string]string{}, Ownership: map[string]string{}, Withheld: map[string]string{}}
		var emitters []func() error
		if err := translatePackage(throwaway, byPath[candidate], sourceDir, unit, Options{}, &emitters); err != nil {
			result.PackagesBodyOnly = append(result.PackagesBodyOnly, candidate+": "+firstLine(err.Error()))
			continue
		}
		if reason, withheld := throwaway.Withheld[candidate]; withheld {
			blocking := reason
			for _, support := range throwaway.Support {
				if support.State == ir.SupportUnimplemented && len(support.Sites) > 0 {
					blocking = support.Sites[0].Class
					break
				}
			}
			result.PackagesBodyOnly = append(result.PackagesBodyOnly, candidate+": "+blocking)
			continue
		}
		verified = append(verified, candidate)
	}
	result.PackagesIRDeclComplete = verified
	result.ClassInventory = buildClassInventory(result)
	return result, nil
}

// buildClassInventory summarizes every unsupported class: its disposition
// category and root abstraction, its raw site count, and the number of
// DISTINCT units it blocks — sorted most-blocking first.
func buildClassInventory(result *ProbeResult) []ClassRow {
	unitsByClass := map[string]map[string]bool{}
	for _, unitInv := range result.UnimplementedUnits {
		for _, site := range unitInv.Sites {
			if unitsByClass[site.Class] == nil {
				unitsByClass[site.Class] = map[string]bool{}
			}
			unitsByClass[site.Class][unitInv.ID] = true
		}
	}
	rows := make([]ClassRow, 0, len(result.BlockerHistogram))
	for class, sites := range result.BlockerHistogram {
		category, root := classifySite(result.kindByClass[class])
		rows = append(rows, ClassRow{
			Class: class, Category: category, Root: root,
			Units: len(unitsByClass[class]), Sites: sites})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Sites != rows[j].Sites {
			return rows[i].Sites > rows[j].Sites
		}
		return rows[i].Class < rows[j].Class
	})
	return rows
}

func probeFunc(p *packages.Package, sourceDir string, unit ir.Scope, source []byte, decl *ast.FuncDecl) (*ir.Func, error) {
	start := p.Fset.Position(decl.Body.Pos()).Offset
	end := p.Fset.Position(decl.Body.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return nil, fmt.Errorf("invalid body span")
	}
	digest := sha256.Sum256(source[start:end])
	id := goid.Func(p.PkgPath, decl.Name.Name)
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), decl.Name.Name)
	} else if goid.IsRepeatable("func", decl.Name.Name) {
		// init and blank functions repeat legally: their identities are
		// position-qualified, exactly like the census and the corpus, so
		// probe and corpus classifications join one-to-one.
		filename := p.Fset.Position(decl.Pos()).Filename
		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil {
			return nil, err
		}
		position := p.Fset.Position(decl.Name.Pos())
		id = goid.Repeatable(p.PkgPath, "func", decl.Name.Name, filepath.ToSlash(relative), position.Line, position.Column)
	}
	return ir.BuildFunc(p, sourceDir, unit, decl, id, hex.EncodeToString(digest[:]))
}

// externalRefOfSite extracts the qualified callee of an external-
// reference site, when the blocker is one.
func externalRefOfSite(site ir.UnsupportedSite) (string, bool) {
	for _, prefix := range []string{"call outside the translated unit (", "method call outside the translated unit ("} {
		if strings.HasPrefix(site.Construct, prefix) && strings.HasSuffix(site.Construct, ")") {
			return strings.TrimSuffix(strings.TrimPrefix(site.Construct, prefix), ")"), true
		}
	}
	return "", false
}

func firstLine(s string) string {
	if index := strings.IndexByte(s, '\n'); index >= 0 {
		return s[:index]
	}
	return s
}
