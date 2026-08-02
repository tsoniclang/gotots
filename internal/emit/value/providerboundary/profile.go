package providerboundary

import (
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
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
	canonicalResults := affectedRoots(base.resultInterfaces, base.affected)
	required := false
	for _, candidate := range candidates {
		required = required || candidate.Profile().Required()
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
	var exactMatches []matchedProfile
	for _, selected := range matches {
		if !selected.elevated {
			exactMatches = append(exactMatches, selected)
		}
	}
	if len(exactMatches) != 0 {
		matches = exactMatches
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
	elevated bool
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
	allowedResults := mergeIndexes(
		canonicalResults,
		implementedResults,
	)
	if !indexesSubset(profile.CanonicalResults(), allowedResults) ||
		!indexesSubset(implementedResults, profile.CanonicalResults()) {
		return matchedProfile{}, false, "receiver or canonical roots differ", nil
	}
	guardIdentities := make(map[string]struct{}, len(profile.GuardInterfaces()))
	for _, identity := range profile.GuardInterfaces() {
		guardIdentities[identity] = struct{}{}
	}
	profileInterfaces := profile.Interfaces()
	boundaryIdentities := make(map[string]struct{}, len(profileInterfaces))
	for _, selectedInterface := range profileInterfaces {
		if _, guard := guardIdentities[selectedInterface.SourceIdentity()]; !guard {
			boundaryIdentities[selectedInterface.SourceIdentity()] = struct{}{}
		}
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
	canonicalIdentities := identitiesAtRoots(
		selected.parameterInterfaces,
		profile.CanonicalParameters(),
	)
	for identity := range identitiesAtRoots(
		selected.resultInterfaces,
		profile.CanonicalResults(),
	) {
		canonicalIdentities[identity] = struct{}{}
	}
	for identity := range guardIdentities {
		canonicalIdentities[identity] = struct{}{}
	}
	providerResultIdentities := identitiesAtRoots(
		selected.resultInterfaces,
		indexesDifference(canonicalResults, profile.CanonicalResults()),
	)
	keyInterfaces := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(profileInterfaces),
	)
	elevatedProfile := false
	for _, selectedInterface := range profileInterfaces {
		node := selected.analyzer.nodes[selectedInterface.SourceIdentity()]
		identity := selectedInterface.SourceIdentity()
		_, mayElevate := implemented[identity]
		_, canonical := canonicalIdentities[identity]
		_, providerResult := providerResultIdentities[identity]
		var elevation profileEffectElevation
		if mayElevate {
			elevation = func(
				method profileBoundaryMethod,
			) ([]api.RootRequest, error) {
				return cooperativecall.ElevateInterfaceMethodContract(
					context,
					method.reference,
				)
			}
		}
		matchesInterface, mismatch, keyInterface, requests, elevated, matchErr :=
			matchProfileInterface(
				node,
				selectedInterface.ProviderInterface(),
				elevation,
				providerResult && !canonical && !mayElevate,
			)
		if matchErr != nil {
			return matchedProfile{}, false, "", matchErr
		}
		if !matchesInterface {
			return matchedProfile{}, false,
				"interface ABI differs for " + selectedInterface.SourceIdentity() +
					": " + mismatch, nil
		}
		selected.analyzer.requests = append(
			selected.analyzer.requests,
			requests...,
		)
		elevatedProfile = elevatedProfile || elevated
		keyInterfaces = append(keyInterfaces, keyInterface)
	}
	profileKey, err := gostdlib.BuildImplementedResultProfileKey(
		keyInterfaces,
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
		elevated: elevatedProfile,
	}, true, "", nil
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

func identitiesAtRoots(
	roots []map[string]struct{},
	indexes []int,
) map[string]struct{} {
	result := make(map[string]struct{})
	for _, index := range indexes {
		if index < 0 || index >= len(roots) {
			continue
		}
		for identity := range roots[index] {
			result[identity] = struct{}{}
		}
	}
	return result
}

func indexesSubset(subset []int, superset []int) bool {
	for _, index := range subset {
		if _, found := slices.BinarySearch(superset, index); !found {
			return false
		}
	}
	return true
}

func indexesDifference(left []int, right []int) []int {
	result := make([]int, 0, len(left))
	for _, index := range left {
		if _, found := slices.BinarySearch(right, index); !found {
			result = append(result, index)
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

type profileBoundaryAnalyzer struct {
	context  api.Context
	nodes    map[string]*profileBoundaryInterface
	building map[string]bool
	visited  map[profileTypeVisit]struct{}
	requests []api.RootRequest
}

type profileTypeVisit struct {
	source types.Type
	parent string
}

type profileBoundaryInterface struct {
	identity       string
	methods        []profileBoundaryMethod
	children       map[string]struct{}
	directMismatch bool
}

func (a *profileBoundaryAnalyzer) collectRootType(
	source types.Type,
	root map[string]struct{},
) error {
	a.visited = make(map[profileTypeVisit]struct{})
	return a.collectType(source, "", root)
}

type profileBoundaryMethod struct {
	identity  string
	signature string
	effect    gostdlib.EffectKind
	reference api.InterfaceMethodCallableReference
}

func (a *profileBoundaryAnalyzer) collectType(
	source types.Type,
	parent string,
	root map[string]struct{},
) error {
	if source == nil {
		return nil
	}
	visit := profileTypeVisit{source: source, parent: parent}
	if _, seen := a.visited[visit]; seen {
		return nil
	}
	a.visited[visit] = struct{}{}
	source = types.Unalias(source)
	switch selected := source.(type) {
	case *types.Named:
		if _, ok := selected.Underlying().(*types.Interface); ok {
			provider, providerOwned, err := a.context.Names().ProviderInterface(selected)
			if err != nil {
				return err
			}
			if providerOwned {
				if provider.Mode() == gostdlib.ProviderInterfaceModeSealedNative {
					return nil
				}
				identity, err := sourceObjectIdentity(selected.Obj())
				if err != nil {
					return err
				}
				if root != nil {
					root[identity] = struct{}{}
				}
				if parent != "" {
					a.nodes[parent].children[identity] = struct{}{}
				}
				interfaceType := selected.Underlying().(*types.Interface).Complete()
				return a.ensureInterface(
					interfaceType,
					identity,
					provider,
					func(method *types.Func) (string, string, error) {
						return gostdlib.ProviderInterfaceMethodSource(method)
					},
				)
			}
		}
		return a.collectType(selected.Underlying(), parent, root)
	case *types.Struct:
		for index := range selected.NumFields() {
			if err := a.collectType(
				selected.Field(index).Type(),
				parent,
				root,
			); err != nil {
				return err
			}
		}
	case *types.Interface:
		selected = selected.Complete()
		for index := range selected.NumMethods() {
			signature, ok := selected.Method(index).Type().(*types.Signature)
			if !ok {
				return boundaryInvariant(a.context, "interface method signature is invalid")
			}
			if err := a.collectSignature(signature, parent, root); err != nil {
				return err
			}
		}
	case *types.Pointer:
		return a.collectType(selected.Elem(), parent, root)
	case *types.Slice:
		return a.collectType(selected.Elem(), parent, root)
	case *types.Array:
		return a.collectType(selected.Elem(), parent, root)
	case *types.Map:
		if err := a.collectType(selected.Key(), parent, root); err != nil {
			return err
		}
		return a.collectType(selected.Elem(), parent, root)
	case *types.Chan:
		return a.collectType(selected.Elem(), parent, root)
	case *types.Signature:
		return a.collectSignature(selected, parent, root)
	case *types.Tuple:
		for index := range selected.Len() {
			if err := a.collectType(selected.At(index).Type(), parent, root); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *profileBoundaryAnalyzer) collectSignature(
	signature *types.Signature,
	parent string,
	root map[string]struct{},
) error {
	if signature == nil {
		return nil
	}
	for _, tuple := range []*types.Tuple{signature.Params(), signature.Results()} {
		if tuple == nil {
			continue
		}
		for index := range tuple.Len() {
			if err := a.collectType(tuple.At(index).Type(), parent, root); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *profileBoundaryAnalyzer) collectProfileInterface(
	source types.Type,
	certificate gostdlib.ProviderCallableProfileInterface,
) error {
	if protocol, synthetic := certificate.Protocol(); synthetic {
		interfaceType, ok := types.Unalias(source).(*types.Interface)
		if !ok {
			return boundaryInvariant(
				a.context,
				"provider callable-profile protocol is not an anonymous interface",
			)
		}
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(protocol)
		if err != nil {
			return err
		}
		if identity != certificate.SourceIdentity() {
			return boundaryInvariant(
				a.context,
				"provider callable-profile protocol identity diverged from its certificate",
			)
		}
		return a.ensureInterface(
			interfaceType.Complete(),
			identity,
			certificate.ProviderInterface(),
			func(method *types.Func) (string, string, error) {
				document, ok := gostdlib.ProviderProtocolMethod(
					protocol,
					method.Name(),
				)
				if !ok {
					return "", "", boundaryInvariant(
						a.context,
						"provider protocol method is absent from its certificate",
					)
				}
				return gostdlib.ProviderProtocolMethodSource(identity, document)
			},
		)
	}
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return boundaryInvariant(
			a.context,
			"provider callable-profile guard is not a named interface",
		)
	}
	if _, ok := named.Underlying().(*types.Interface); !ok {
		return boundaryInvariant(
			a.context,
			"provider callable-profile guard is not an interface",
		)
	}
	identity, err := sourceObjectIdentity(named.Obj())
	if err != nil {
		return err
	}
	if identity != certificate.SourceIdentity() {
		return boundaryInvariant(
			a.context,
			"provider callable-profile guard identity diverged from its certificate",
		)
	}
	return a.ensureInterface(
		named.Underlying().(*types.Interface).Complete(),
		identity,
		certificate.ProviderInterface(),
		func(method *types.Func) (string, string, error) {
			return gostdlib.ProviderInterfaceMethodSource(method)
		},
	)
}

type profileMethodSource func(*types.Func) (string, string, error)

func (a *profileBoundaryAnalyzer) ensureInterface(
	interfaceType *types.Interface,
	interfaceIdentity string,
	provider gostdlib.ProviderInterface,
	methodSource profileMethodSource,
) error {
	if a.nodes[interfaceIdentity] != nil {
		return nil
	}
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return boundaryInvariant(
			a.context,
			"sealed provider interface cannot define a callable boundary profile",
		)
	}
	node := &profileBoundaryInterface{
		identity: interfaceIdentity,
		children: make(map[string]struct{}),
	}
	a.nodes[interfaceIdentity] = node
	if a.building[interfaceIdentity] {
		return nil
	}
	a.building[interfaceIdentity] = true
	defer delete(a.building, interfaceIdentity)
	if interfaceType == nil || methodSource == nil {
		return boundaryInvariant(
			a.context,
			"provider callable-profile interface evidence is incomplete",
		)
	}
	interfaceType = interfaceType.Complete()
	if len(provider.Methods()) != interfaceType.NumMethods() {
		return boundaryInvariant(
			a.context,
			"provider interface method certificate is incomplete",
		)
	}
	for index := range interfaceType.NumMethods() {
		method := interfaceType.Method(index).Origin()
		methodIdentity, sourceSignature, err := methodSource(method)
		if err != nil {
			return err
		}
		certificate, ok := provider.Method(methodIdentity)
		if !ok || certificate.SourceSignature() != sourceSignature ||
			certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable {
			return boundaryInvariant(
				a.context,
				"provider callable-profile method certificate is incomplete",
			)
		}
		callableReference, err := a.context.Names().InterfaceMethodCallable(method)
		if err != nil {
			return err
		}
		cooperative, requests, err := cooperativecall.InterfaceMethodContract(
			a.context,
			callableReference,
		)
		if err != nil {
			return err
		}
		a.requests = append(a.requests, requests...)
		effect := gostdlib.EffectSynchronous
		if cooperative {
			effect = gostdlib.EffectAsynchronous
		}
		node.directMismatch = node.directMismatch ||
			certificate.Effect() != effect
		node.methods = append(node.methods, profileBoundaryMethod{
			identity:  methodIdentity,
			signature: sourceSignature,
			effect:    effect,
			reference: callableReference,
		})
		methodSignature, ok := method.Type().(*types.Signature)
		if !ok {
			return boundaryInvariant(a.context, "provider interface method type is invalid")
		}
		if err := a.collectSignature(
			methodSignature,
			interfaceIdentity,
			nil,
		); err != nil {
			return err
		}
	}
	sort.Slice(node.methods, func(left, right int) bool {
		return node.methods[left].identity < node.methods[right].identity
	})
	return nil
}

func (a *profileBoundaryAnalyzer) affectedInterfaces() map[string]struct{} {
	affected := make(map[string]struct{})
	for identity, node := range a.nodes {
		if node.directMismatch {
			affected[identity] = struct{}{}
		}
	}
	for changed := true; changed; {
		changed = false
		for identity, node := range a.nodes {
			if _, selected := affected[identity]; selected {
				continue
			}
			for child := range node.children {
				if _, selected := affected[child]; selected {
					affected[identity] = struct{}{}
					changed = true
					break
				}
			}
		}
	}
	return affected
}

func affectedRoots(
	roots []map[string]struct{},
	affected map[string]struct{},
) []int {
	result := make([]int, 0, len(roots))
	for index, identities := range roots {
		for identity := range identities {
			if _, selected := affected[identity]; selected {
				result = append(result, index)
				break
			}
		}
	}
	return result
}

type profileEffectElevation func(
	profileBoundaryMethod,
) ([]api.RootRequest, error)

func matchProfileInterface(
	node *profileBoundaryInterface,
	provider gostdlib.ProviderInterface,
	elevate profileEffectElevation,
	providerResult bool,
) (
	bool,
	string,
	gostdlib.ProviderCallableProfileKeyInterface,
	[]api.RootRequest,
	bool,
	error,
) {
	if node == nil {
		return false, "source interface evidence is absent",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
	}
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return false, "provider interface is not a complete bridge",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
	}
	if len(provider.Methods()) != len(node.methods) {
		return false, "method counts differ",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
	}
	keyMethods := make(
		[]gostdlib.ProviderCallableProfileKeyMethod,
		0,
		len(node.methods),
	)
	var requests []api.RootRequest
	introducedElevation := false
	for _, method := range node.methods {
		selected, ok := provider.Method(method.identity)
		if !ok {
			return false, "method is absent: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
		}
		if selected.Kind() != gostdlib.ProviderInterfaceMethodCallable {
			return false, "method is not callable: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
		}
		if selected.SourceSignature() != method.signature {
			return false, "source signature differs: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
		}
		if selected.Effect() != method.effect {
			implementedElevation := elevate != nil &&
				method.effect == gostdlib.EffectSynchronous &&
				selected.Effect() == gostdlib.EffectAsynchronous
			providerBridge := providerResult &&
				method.effect == gostdlib.EffectAsynchronous &&
				selected.Effect() == gostdlib.EffectSynchronous
			if !implementedElevation && !providerBridge {
				return false, "effect differs for " + method.identity +
						" (source=" + string(method.effect) +
						", provider=" + string(selected.Effect()) + ")",
					gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
			}
			introducedElevation = introducedElevation || implementedElevation
		}
		if elevate != nil &&
			selected.Effect() == gostdlib.EffectAsynchronous {
			selectedRequests, err := elevate(method)
			if err != nil {
				return false, "", gostdlib.ProviderCallableProfileKeyInterface{}, nil,
					false, err
			}
			if len(selectedRequests) == 0 {
				return false, "effect elevation has no owning request: " + method.identity,
					gostdlib.ProviderCallableProfileKeyInterface{}, nil, false, nil
			}
			requests = append(requests, selectedRequests...)
		}
		keyMethods = append(keyMethods, gostdlib.ProviderCallableProfileKeyMethod{
			SourceIdentity: method.identity,
			Effect:         selected.Effect(),
		})
	}
	return true, "", gostdlib.ProviderCallableProfileKeyInterface{
		SourceIdentity: node.identity,
		Methods:        keyMethods,
	}, api.CombineRequests(requests), introducedElevation, nil
}

func sameIdentitySet(left map[string]struct{}, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for identity := range left {
		if _, ok := right[identity]; !ok {
			return false
		}
	}
	return true
}

func sourceObjectIdentity(object types.Object) (string, error) {
	if object == types.Universe.Lookup("error") {
		return gostdlib.LanguageErrorInterfaceIdentity, nil
	}
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return "", err
	}
	return contract.Identity(), nil
}
