package providerboundary

import (
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
)

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
						return gostdlibsource.ProviderInterfaceMethod(method)
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
			return gostdlibsource.ProviderInterfaceMethod(method)
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
			effect = gostdlib.EffectAwaitable
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

func matchProfileInterface(
	node *profileBoundaryInterface,
	provider gostdlib.ProviderInterface,
) (
	bool,
	string,
	gostdlib.ProviderCallableProfileKeyInterface,
	error,
) {
	if node == nil {
		return false, "source interface evidence is absent",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil
	}
	if provider.Mode() != gostdlib.ProviderInterfaceModeBridge {
		return false, "provider interface is not a complete bridge",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil
	}
	if len(provider.Methods()) != len(node.methods) {
		return false, "method counts differ",
			gostdlib.ProviderCallableProfileKeyInterface{}, nil
	}
	keyMethods := make(
		[]gostdlib.ProviderCallableProfileKeyMethod,
		0,
		len(node.methods),
	)
	for _, method := range node.methods {
		selected, ok := provider.Method(method.identity)
		if !ok {
			return false, "method is absent: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil
		}
		if selected.Kind() != gostdlib.ProviderInterfaceMethodCallable {
			return false, "method is not callable: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil
		}
		if selected.SourceSignature() != method.signature {
			return false, "source signature differs: " + method.identity,
				gostdlib.ProviderCallableProfileKeyInterface{}, nil
		}
		if selected.Effect() != method.effect {
			return false, "effect differs for " + method.identity +
					" (source=" + string(method.effect) +
					", provider=" + string(selected.Effect()) + ")",
				gostdlib.ProviderCallableProfileKeyInterface{}, nil
		}
		keyMethods = append(keyMethods, gostdlib.ProviderCallableProfileKeyMethod{
			SourceIdentity: method.identity,
			Effect:         selected.Effect(),
		})
	}
	return true, "", gostdlib.ProviderCallableProfileKeyInterface{
		SourceIdentity: node.identity,
		Methods:        keyMethods,
	}, nil
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
