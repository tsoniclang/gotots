package providerboundary

import (
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type CallableProfileSelection struct {
	reference api.ProviderCallableProfileReference
	requests  []api.RootRequest
}

func InterfaceABIExact(
	context api.Context,
	source *types.Named,
) (bool, []api.RootRequest, error) {
	if source == nil || source.Obj() == nil {
		return false, nil, boundaryInvariant(
			context,
			"provider interface source is invalid",
		)
	}
	if _, ok := source.Underlying().(*types.Interface); !ok {
		return false, nil, boundaryInvariant(
			context,
			"provider interface source is not an interface",
		)
	}
	analyzer := &profileBoundaryAnalyzer{
		context:  context,
		nodes:    make(map[string]*profileBoundaryInterface),
		building: make(map[string]bool),
		visited:  make(map[profileTypeVisit]struct{}),
	}
	if err := analyzer.collectRootType(
		source,
		make(map[string]struct{}),
	); err != nil {
		return false, nil, err
	}
	return len(analyzer.affectedInterfaces()) == 0,
		api.CombineRequests(analyzer.requests), nil
}

func ResolveCallableProfile(
	context api.Context,
	owner *types.Func,
	signature *types.Signature,
) (CallableProfileSelection, bool, error) {
	if owner == nil || signature == nil {
		return CallableProfileSelection{}, false, boundaryInvariant(
			context,
			"provider callable-profile source is invalid",
		)
	}
	candidates, providerOwned, err :=
		context.Names().ProviderCallableProfileCandidates(owner.Origin())
	if err != nil || !providerOwned {
		return CallableProfileSelection{}, false, err
	}
	base, err := analyzeCallableProfileBoundary(context, signature)
	if err != nil {
		return CallableProfileSelection{}, false, err
	}
	canonicalParameters := affectedRoots(base.parameterInterfaces, base.affected)
	var callableEvidence []gostdlib.ProviderCallableParameterDocument
	if len(profileCallableParameterRoots(signature)) != 0 {
		var callableOwned bool
		callableEvidence, callableOwned, err =
			context.Names().ProviderCallableParameters(owner.Origin())
		if err != nil || !callableOwned {
			return CallableProfileSelection{}, false, err
		}
	}
	callableParameters, mismatchedCallables, err :=
		profileCallableParameterBoundary(
			context,
			signature,
			callableEvidence,
		)
	if err != nil {
		return CallableProfileSelection{}, false, err
	}
	canonicalParameters = mergeIndexes(canonicalParameters, mismatchedCallables)
	canonicalResults := affectedRoots(base.resultInterfaces, base.affected)
	required := false
	for _, candidate := range candidates {
		required = required || candidate.Profile().Required()
	}
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsDisabled &&
		!required {
		return CallableProfileSelection{
			requests: api.CombineRequests(base.analyzer.requests),
		}, false, nil
	}
	if len(canonicalParameters) == 0 && !required {
		return CallableProfileSelection{
			requests: api.CombineRequests(base.analyzer.requests),
		}, false, nil
	}
	var matches []matchedProfile
	var mismatches []string
	for _, candidate := range candidates {
		if required && !candidate.Profile().Required() {
			continue
		}
		expectedParameters := slices.Clone(canonicalParameters)
		if candidate.Profile().Required() {
			semanticParameters, parameterErr := requiredProfileParameters(
				candidate.Profile(),
			)
			if parameterErr != nil {
				return CallableProfileSelection{}, false, parameterErr
			}
			expectedParameters = mergeIndexes(expectedParameters, semanticParameters)
		}
		selected, matchesCurrent, mismatch, matchErr := matchCallableProfileCandidate(
			context,
			signature,
			candidate,
			expectedParameters,
			canonicalResults,
		)
		if matchErr != nil {
			return CallableProfileSelection{}, false, matchErr
		}
		if matchesCurrent {
			matches = append(matches, selected)
		} else {
			mismatches = append(
				mismatches,
				candidate.Profile().ProfileKey()+": "+mismatch,
			)
		}
	}
	if len(matches) != 1 {
		reason := "provider callable has no exact certified boundary profile"
		if len(matches) > 1 {
			reason = "provider callable has multiple exact certified boundary profiles"
		}
		identity, identityErr := sourceObjectIdentity(owner.Origin())
		if identityErr != nil {
			return CallableProfileSelection{}, false, identityErr
		}
		reason += " for " + identity
		reason += " with affected interfaces [" +
			strings.Join(sortedIdentitySet(base.affected), ", ") + "]"
		reason += " and canonical parameters [" +
			joinIndexes(canonicalParameters) + "]"
		reason += " and callable parameters [" +
			joinIndexes(callableParameters) + "]"
		reason += " and canonical results [" +
			joinIndexes(canonicalResults) + "]"
		if len(mismatches) != 0 {
			reason += " (" + strings.Join(mismatches, "; ") + ")"
		}
		return CallableProfileSelection{}, false, boundaryInvariant(context, reason)
	}
	selected := matches[0]
	reference, referenced, err := context.Names().ProviderCallableProfile(
		owner.Origin(),
		selected.profile.ProfileKey(),
	)
	if err != nil || !referenced {
		return CallableProfileSelection{}, false, err
	}
	if reference.Profile().ProfileKey() != selected.profile.ProfileKey() {
		return CallableProfileSelection{}, false, boundaryInvariant(
			context,
			"provider callable-profile reference diverged from its selected candidate",
		)
	}
	if err := RequireSynchronousEffect(
		context,
		selected.profile.ProfileKey(),
		selected.profile.Effect(),
	); err != nil {
		return CallableProfileSelection{}, false, err
	}
	return CallableProfileSelection{
		reference: reference,
		requests: api.CombineRequests(
			base.analyzer.requests,
			selected.requests,
		),
	}, true, nil
}

func requiredProfileParameters(
	profile gostdlib.ProviderCallableProfile,
) ([]int, error) {
	var result []int
	for _, selected := range profile.Interfaces() {
		protocol, ok := selected.Protocol()
		if !ok {
			continue
		}
		parameters, err := gostdlib.ProviderProtocolCallableParameters(protocol)
		if err != nil {
			return nil, err
		}
		result = mergeIndexes(result, parameters)
	}
	return result, nil
}

func mergeIndexes(left []int, right []int) []int {
	seen := make(map[int]struct{}, len(left)+len(right))
	for _, index := range left {
		seen[index] = struct{}{}
	}
	for _, index := range right {
		seen[index] = struct{}{}
	}
	result := make([]int, 0, len(seen))
	for index := range seen {
		result = append(result, index)
	}
	sort.Ints(result)
	return result
}

func sortedIdentitySet(source map[string]struct{}) []string {
	result := make([]string, 0, len(source))
	for identity := range source {
		result = append(result, identity)
	}
	sort.Strings(result)
	return result
}

func joinIndexes(source []int) string {
	result := make([]string, len(source))
	for index, value := range source {
		result[index] = strconv.Itoa(value)
	}
	return strings.Join(result, ", ")
}

type callableProfileBoundary struct {
	analyzer            *profileBoundaryAnalyzer
	parameterInterfaces []map[string]struct{}
	resultInterfaces    []map[string]struct{}
	affected            map[string]struct{}
}

type matchedProfile struct {
	profile  gostdlib.ProviderCallableProfile
	requests []api.RootRequest
}

func analyzeCallableProfileBoundary(
	context api.Context,
	signature *types.Signature,
) (callableProfileBoundary, error) {
	analyzer := &profileBoundaryAnalyzer{
		context:  context,
		nodes:    make(map[string]*profileBoundaryInterface),
		building: make(map[string]bool),
		visited:  make(map[profileTypeVisit]struct{}),
	}
	parameterInterfaces := make([]map[string]struct{}, signature.Params().Len())
	for index := range signature.Params().Len() {
		parameterInterfaces[index] = make(map[string]struct{})
		if err := analyzer.collectRootType(
			signature.Params().At(index).Type(),
			parameterInterfaces[index],
		); err != nil {
			return callableProfileBoundary{}, err
		}
	}
	resultLength := 0
	if signature.Results() != nil {
		resultLength = signature.Results().Len()
	}
	resultInterfaces := make([]map[string]struct{}, resultLength)
	for index := range resultLength {
		resultInterfaces[index] = make(map[string]struct{})
		if err := analyzer.collectRootType(
			signature.Results().At(index).Type(),
			resultInterfaces[index],
		); err != nil {
			return callableProfileBoundary{}, err
		}
	}
	return callableProfileBoundary{
		analyzer:            analyzer,
		parameterInterfaces: parameterInterfaces,
		resultInterfaces:    resultInterfaces,
		affected:            analyzer.affectedInterfaces(),
	}, nil
}

func matchCallableProfileCandidate(
	context api.Context,
	signature *types.Signature,
	candidate api.ProviderCallableProfileCandidate,
	canonicalParameters []int,
	canonicalResults []int,
) (matchedProfile, bool, string, error) {
	profile := candidate.Profile()
	if profile.Receiver() != (signature.Recv() != nil) ||
		!slices.Equal(profile.CanonicalParameters(), canonicalParameters) {
		return matchedProfile{}, false, "receiver or canonical roots differ", nil
	}
	selected, err := analyzeCallableProfileBoundary(context, signature)
	if err != nil {
		return matchedProfile{}, false, "", err
	}
	implemented := make(
		map[string]struct{},
		len(profile.ImplementedResultInterfaces()),
	)
	for _, identity := range profile.ImplementedResultInterfaces() {
		implemented[identity] = struct{}{}
	}
	implementedResults := rootsContainingIdentities(
		selected.resultInterfaces,
		implemented,
	)
	requiredResults := mergeIndexes(
		canonicalResults,
		implementedResults,
	)
	if !slices.Equal(profile.CanonicalResults(), requiredResults) {
		return matchedProfile{}, false, "receiver or canonical roots differ", nil
	}
	profileCallables := profile.CallableParameters()
	if !profileCallablesAccept(
		context,
		signature,
		canonicalParameters,
		profileCallables,
	) {
		return matchedProfile{}, false, "transported callable effects differ", nil
	}
	keyCallables := profileKeyCallables(profileCallables)
	guardIdentities := make(map[string]struct{}, len(profile.GuardInterfaces()))
	for _, identity := range profile.GuardInterfaces() {
		guardIdentities[identity] = struct{}{}
	}
	viewDocuments := profile.CapabilityViews()
	viewTypes := candidate.CapabilityViews()
	if len(viewDocuments) != len(viewTypes) {
		return matchedProfile{}, false, "capability-view count differs", nil
	}
	viewTargetIdentities := make(map[string]struct{}, len(viewDocuments))
	for index, view := range viewDocuments {
		baseIdentity, err := sourceObjectIdentity(viewTypes[index].Base().Obj())
		if err != nil {
			return matchedProfile{}, false, "", err
		}
		targetIdentity, err := sourceObjectIdentity(viewTypes[index].Target().Obj())
		if err != nil {
			return matchedProfile{}, false, "", err
		}
		if baseIdentity != view.BaseSourceIdentity ||
			targetIdentity != view.TargetSourceIdentity {
			return matchedProfile{}, false, "capability-view identity differs", nil
		}
		if selected.analyzer.nodes[baseIdentity] == nil {
			return matchedProfile{}, false, "capability-view base is not transported", nil
		}
		viewTargetIdentities[targetIdentity] = struct{}{}
	}
	profileInterfaces := profile.Interfaces()
	boundaryIdentities := make(map[string]struct{}, len(profileInterfaces))
	for _, selectedInterface := range profileInterfaces {
		identity := selectedInterface.SourceIdentity()
		if _, guard := guardIdentities[identity]; guard {
			continue
		}
		if _, capabilityTarget := viewTargetIdentities[identity]; capabilityTarget {
			continue
		}
		boundaryIdentities[identity] = struct{}{}
	}
	expectedBoundaryIdentities := cloneIdentitySet(selected.affected)
	for identity := range implemented {
		expectedBoundaryIdentities[identity] = struct{}{}
	}
	if !sameIdentitySet(boundaryIdentities, expectedBoundaryIdentities) {
		return matchedProfile{}, false, "affected interface set differs", nil
	}
	guardTypes := candidate.Guards()
	guardIdentityList := profile.GuardInterfaces()
	for index, guard := range guardTypes {
		certificate, ok := profile.Interface(guardIdentityList[index])
		if !ok {
			return matchedProfile{}, false, "guard certificate is absent", nil
		}
		if err := selected.analyzer.collectProfileInterface(
			guard,
			certificate,
		); err != nil {
			return matchedProfile{}, false, "", err
		}
	}
	for index, view := range viewTypes {
		certificate, ok := profile.Interface(
			viewDocuments[index].TargetSourceIdentity,
		)
		if !ok {
			return matchedProfile{}, false,
				"capability-view target certificate is absent", nil
		}
		if err := selected.analyzer.collectProfileInterface(
			view.Target(),
			certificate,
		); err != nil {
			return matchedProfile{}, false, "", err
		}
	}
	keyInterfaces := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(profileInterfaces),
	)
	for _, selectedInterface := range profileInterfaces {
		node := selected.analyzer.nodes[selectedInterface.SourceIdentity()]
		matchesInterface, mismatch, keyInterface, matchErr :=
			matchProfileInterface(
				node,
				selectedInterface.ProviderInterface(),
			)
		if matchErr != nil {
			return matchedProfile{}, false, "", matchErr
		}
		if !matchesInterface {
			return matchedProfile{}, false,
				"interface ABI differs for " + selectedInterface.SourceIdentity() +
					": " + mismatch, nil
		}
		keyInterfaces = append(keyInterfaces, keyInterface)
	}
	profileKey, err := gostdlib.BuildImplementedResultProfileKey(
		keyInterfaces,
		keyCallables,
		profile.ImplementedResultInterfaces(),
	)
	if err != nil {
		return matchedProfile{}, false, "", err
	}
	if profileKey != profile.ProfileKey() {
		return matchedProfile{}, false, "profile key differs", nil
	}
	return matchedProfile{
		profile:  profile,
		requests: selected.analyzer.requests,
	}, true, "", nil
}

func profileCallableParameterRoots(signature *types.Signature) []int {
	if signature == nil || signature.Params() == nil {
		return nil
	}
	var result []int
	for index := range signature.Params().Len() {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(index).Type(),
		); callable {
			result = append(result, index)
		}
	}
	return result
}

func profileCallableParameterBoundary(
	context api.Context,
	signature *types.Signature,
	evidence []gostdlib.ProviderCallableParameterDocument,
) ([]int, []int, error) {
	roots := profileCallableParameterRoots(signature)
	if len(roots) != len(evidence) {
		return nil, nil, boundaryInvariant(
			context,
			"provider callable-parameter evidence does not exact-join the source signature",
		)
	}
	expected := gostdlib.EffectSynchronous
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsCooperative {
		expected = gostdlib.EffectAwaitable
	}
	var mismatched []int
	for index, root := range roots {
		selected := evidence[index]
		if selected.Parameter != root || !selected.Effect.Valid() {
			return nil, nil, boundaryInvariant(
				context,
				"provider callable-parameter evidence diverged from the source signature",
			)
		}
		if !providerCallableEffectAccepts(selected.Effect, expected) {
			mismatched = append(mismatched, root)
		}
	}
	return roots, mismatched, nil
}

func profileCallablesAccept(
	context api.Context,
	signature *types.Signature,
	canonicalParameters []int,
	callables []gostdlib.ProviderCallableParameterDocument,
) bool {
	var roots []int
	for _, parameter := range canonicalParameters {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(parameter).Type(),
		); callable {
			roots = append(roots, parameter)
		}
	}
	if len(roots) != len(callables) {
		return false
	}
	expected := gostdlib.EffectSynchronous
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsCooperative {
		expected = gostdlib.EffectAwaitable
	}
	for index, root := range roots {
		if callables[index].Parameter != root ||
			!providerCallableEffectAccepts(callables[index].Effect, expected) {
			return false
		}
	}
	return true
}

func profileKeyCallables(
	source []gostdlib.ProviderCallableParameterDocument,
) []gostdlib.ProviderCallableProfileKeyCallable {
	result := make(
		[]gostdlib.ProviderCallableProfileKeyCallable,
		len(source),
	)
	for index, selected := range source {
		result[index] = gostdlib.ProviderCallableProfileKeyCallable{
			Parameter: selected.Parameter,
			Effect:    selected.Effect,
		}
	}
	return result
}

func providerCallableEffectAccepts(
	provider gostdlib.EffectKind,
	generated gostdlib.EffectKind,
) bool {
	return provider == generated ||
		provider == gostdlib.EffectAwaitable &&
			generated == gostdlib.EffectSynchronous
}

func rootsContainingIdentities(
	roots []map[string]struct{},
	identities map[string]struct{},
) []int {
	result := make([]int, 0, len(roots))
	for index, root := range roots {
		for identity := range identities {
			if _, found := root[identity]; found {
				result = append(result, index)
				break
			}
		}
	}
	return result
}

func cloneIdentitySet(source map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(source))
	for identity := range source {
		result[identity] = struct{}{}
	}
	return result
}

func (s CallableProfileSelection) Reference() api.ProviderCallableProfileReference {
	return s.reference
}

func (s CallableProfileSelection) Requests() []api.RootRequest {
	return slices.Clone(s.requests)
}
