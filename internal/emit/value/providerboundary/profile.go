package providerboundary

import (
	"go/types"
	"slices"
	"sort"
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
	if len(canonicalParameters) == 0 {
		return CallableProfileSelection{
			requests: api.CombineRequests(base.analyzer.requests),
		}, false, nil
	}
	var matches []matchedProfile
	var mismatches []string
	for _, candidate := range candidates {
		selected, matchesCurrent, mismatch, matchErr := matchCallableProfileCandidate(
			context,
			signature,
			candidate,
			canonicalParameters,
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
		if err := analyzer.collectType(
			signature.Params().At(index).Type(),
			"",
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
		if err := analyzer.collectType(
			signature.Results().At(index).Type(),
			"",
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
		!slices.Equal(profile.CanonicalParameters(), canonicalParameters) ||
		!slices.Equal(profile.CanonicalResults(), canonicalResults) {
		return matchedProfile{}, false, "receiver or canonical roots differ", nil
	}
	selected, err := analyzeCallableProfileBoundary(context, signature)
	if err != nil {
		return matchedProfile{}, false, "", err
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
	if !sameIdentitySet(boundaryIdentities, selected.affected) {
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
	keyInterfaces := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(profileInterfaces),
	)
	for _, selectedInterface := range profileInterfaces {
		node := selected.analyzer.nodes[selectedInterface.SourceIdentity()]
		matchesInterface, mismatch := profileInterfaceMatches(
			node,
			selectedInterface.ProviderInterface(),
		)
		if !matchesInterface {
			return matchedProfile{}, false,
				"interface ABI differs for " + selectedInterface.SourceIdentity() +
					": " + mismatch, nil
		}
		keyInterfaces = append(keyInterfaces, node.keyInterface())
	}
	profileKey, err := gostdlib.BuildProviderCallableProfileKey(keyInterfaces)
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

type profileBoundaryMethod struct {
	identity  string
	signature string
	effect    gostdlib.EffectKind
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
				return a.ensureInterface(selected, identity, provider)
			}
		}
		return a.collectType(selected.Underlying(), parent, root)
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
		named,
		identity,
		certificate.ProviderInterface(),
	)
}

func (a *profileBoundaryAnalyzer) ensureInterface(
	named *types.Named,
	identity string,
	provider gostdlib.ProviderInterface,
) error {
	if a.nodes[identity] != nil {
		return nil
	}
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return boundaryInvariant(
			a.context,
			"sealed provider interface cannot define a callable boundary profile",
		)
	}
	node := &profileBoundaryInterface{
		identity: identity,
		children: make(map[string]struct{}),
	}
	a.nodes[identity] = node
	if a.building[identity] {
		return nil
	}
	a.building[identity] = true
	defer delete(a.building, identity)
	interfaceType := named.Underlying().(*types.Interface).Complete()
	if len(provider.Methods()) != interfaceType.NumMethods() {
		return boundaryInvariant(
			a.context,
			"provider interface method certificate is incomplete",
		)
	}
	for index := range interfaceType.NumMethods() {
		method := interfaceType.Method(index).Origin()
		contract, err := environmentcontract.Describe(method)
		if err != nil {
			return err
		}
		certificate, ok := provider.Method(contract.Identity())
		if !ok || certificate.SourceSignature() != contract.Signature() ||
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
			identity:  contract.Identity(),
			signature: contract.Signature(),
			effect:    effect,
		})
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			return boundaryInvariant(a.context, "provider interface method type is invalid")
		}
		if err := a.collectSignature(signature, identity, nil); err != nil {
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

func (i *profileBoundaryInterface) keyInterface() gostdlib.ProviderCallableProfileKeyInterface {
	methods := make([]gostdlib.ProviderCallableProfileKeyMethod, len(i.methods))
	for index, method := range i.methods {
		methods[index] = gostdlib.ProviderCallableProfileKeyMethod{
			SourceIdentity: method.identity,
			Effect:         method.effect,
		}
	}
	return gostdlib.ProviderCallableProfileKeyInterface{
		SourceIdentity: i.identity,
		Methods:        methods,
	}
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

func profileInterfaceMatches(
	node *profileBoundaryInterface,
	provider gostdlib.ProviderInterface,
) (bool, string) {
	if node == nil {
		return false, "source interface evidence is absent"
	}
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return false, "provider interface is not a complete bridge"
	}
	if len(provider.Methods()) != len(node.methods) {
		return false, "method counts differ"
	}
	for _, method := range node.methods {
		selected, ok := provider.Method(method.identity)
		if !ok {
			return false, "method is absent: " + method.identity
		}
		if selected.Kind() != gostdlib.ProviderInterfaceMethodCallable {
			return false, "method is not callable: " + method.identity
		}
		if selected.SourceSignature() != method.signature {
			return false, "source signature differs: " + method.identity
		}
		if selected.Effect() != method.effect {
			return false, "effect differs for " + method.identity +
				" (source=" + string(method.effect) +
				", provider=" + string(selected.Effect()) + ")"
		}
	}
	return true, ""
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
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return "", err
	}
	return contract.Identity(), nil
}
