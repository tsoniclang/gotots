// Substitution closure of the generic-instantiation evidence: an
// instantiation recorded INSIDE another generic declaration binds type
// arguments that mention the outer declaration's type parameters (a
// free-parameter vector — e.g. CopyOnWriteSet[K] instantiating
// CopyOnWriteMap[K, struct{}]). Such a vector is not itself evidence of a
// concrete carrier; its concretizations are. The closure substitutes every
// CONCRETE vector of the outer declaration through each free-parameter
// edge to a fixed point, so downstream admission decisions (map-key
// families, value-copy factories, pointer carriers) see the complete
// concrete evidence and can soundly SKIP free-parameter vectors.
package ir

import (
	"go/types"
	"sort"
)

// genericEdge is one instantiation recorded inside a generic declaration
// whose type arguments mention the outer declaration's parameters.
type genericEdge struct {
	// outer identifies the enclosing generic declaration whose parameters
	// the arguments mention: exactly one of outerFunc/outerType is set.
	outerFunc *types.Func
	outerType *types.TypeName
	// inner identifies the instantiated declaration: exactly one of
	// innerFunc/innerType is set.
	innerFunc *types.Func
	innerType *types.TypeName
	args      []types.Type
}

// AddGenericEdge records a free-parameter instantiation edge discovered by
// the prepass. outer and inner are the enclosing and instantiated objects
// (*types.Func or *types.TypeName).
func (s Scope) AddGenericEdge(outer, inner types.Object, args []types.Type) {
	edge := genericEdge{args: args}
	switch o := outer.(type) {
	case *types.Func:
		edge.outerFunc = o
	case *types.TypeName:
		edge.outerType = o
	default:
		return
	}
	switch i := inner.(type) {
	case *types.Func:
		edge.innerFunc = i
	case *types.TypeName:
		edge.innerType = i
	default:
		return
	}
	*s.genericEdges = append(*s.genericEdges, edge)
}

// typeParamsOf returns the type-parameter list an edge's outer
// declaration binds (a method's receiver parameters included).
func (e genericEdge) outerParams() *types.TypeParamList {
	if e.outerType != nil {
		if named, ok := e.outerType.Type().(*types.Named); ok {
			return named.TypeParams()
		}
		return nil
	}
	signature := e.outerFunc.Type().(*types.Signature)
	if recv := signature.RecvTypeParams(); recv != nil && recv.Len() > 0 {
		return recv
	}
	return signature.TypeParams()
}

// outerConcreteVectors returns the outer declaration's recorded CONCRETE
// instantiation vectors (free-parameter vectors excluded).
func (s Scope) outerConcreteVectors(e genericEdge) [][]types.Type {
	var all [][]types.Type
	if e.outerFunc != nil {
		all = s.generics[e.outerFunc]
	} else {
		all = s.typeGenerics[e.outerType]
	}
	out := make([][]types.Type, 0, len(all))
	for _, vector := range all {
		concrete := true
		for _, arg := range vector {
			if mentionsTypeParamType(arg) {
				concrete = false
				break
			}
		}
		if concrete {
			out = append(out, vector)
		}
	}
	return out
}

// substituteParams structurally replaces type parameters (by object
// identity) with their bound arguments. Named instantiated types
// re-instantiate through go/types so their underlying structure and
// identity stay exact.
func substituteParams(t types.Type, binding map[*types.TypeParam]types.Type) types.Type {
	switch u := t.(type) {
	case *types.TypeParam:
		if bound, ok := binding[u]; ok {
			return bound
		}
		return u
	case *types.Pointer:
		return types.NewPointer(substituteParams(u.Elem(), binding))
	case *types.Slice:
		return types.NewSlice(substituteParams(u.Elem(), binding))
	case *types.Array:
		return types.NewArray(substituteParams(u.Elem(), binding), u.Len())
	case *types.Chan:
		return types.NewChan(u.Dir(), substituteParams(u.Elem(), binding))
	case *types.Map:
		return types.NewMap(substituteParams(u.Key(), binding), substituteParams(u.Elem(), binding))
	case *types.Named:
		args := u.TypeArgs()
		if args == nil || args.Len() == 0 {
			return u
		}
		newArgs := make([]types.Type, args.Len())
		changed := false
		for i := range args.Len() {
			newArgs[i] = substituteParams(args.At(i), binding)
			if newArgs[i] != args.At(i) {
				changed = true
			}
		}
		if !changed {
			return u
		}
		instantiated, err := types.Instantiate(nil, u.Origin(), newArgs, false)
		if err != nil {
			return u // fail conservative: the unsubstituted form stays free
		}
		return instantiated
	}
	return t
}

// CloseGenericEvidence substitutes the outer declarations' concrete
// vectors through every free-parameter edge to a fixed point, appending
// newly derived concrete vectors to the instantiation evidence. Iteration
// is sorted by canonical spelling so the closure is deterministic.
func (s Scope) CloseGenericEvidence() {
	edges := *s.genericEdges
	sort.SliceStable(edges, func(i, j int) bool { return edgeKey(edges[i]) < edgeKey(edges[j]) })
	seen := map[string]bool{}
	record := func(e genericEdge, vector []types.Type) bool {
		key := edgeTargetKey(e) + vectorKey(vector)
		if seen[key] {
			return false
		}
		seen[key] = true
		if e.innerFunc != nil {
			s.generics[e.innerFunc] = append(s.generics[e.innerFunc], vector)
		} else {
			s.typeGenerics[e.innerType] = append(s.typeGenerics[e.innerType], vector)
		}
		return true
	}
	// Seed the dedup set with every already-recorded vector.
	for fn, vectors := range s.generics {
		for _, vector := range vectors {
			seen[objKey(fn)+vectorKey(vector)] = true
		}
	}
	for name, vectors := range s.typeGenerics {
		for _, vector := range vectors {
			seen[objKey(name)+vectorKey(vector)] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, edge := range edges {
			params := edge.outerParams()
			if params == nil || params.Len() == 0 {
				continue
			}
			for _, outerVector := range s.outerConcreteVectors(edge) {
				if len(outerVector) != params.Len() {
					continue
				}
				binding := make(map[*types.TypeParam]types.Type, params.Len())
				for i := range params.Len() {
					binding[params.At(i)] = outerVector[i]
				}
				derived := make([]types.Type, len(edge.args))
				concrete := true
				for i, arg := range edge.args {
					derived[i] = substituteParams(arg, binding)
					if mentionsTypeParamType(derived[i]) {
						concrete = false
						break
					}
				}
				if !concrete {
					continue
				}
				if record(edge, derived) {
					changed = true
				}
			}
		}
	}
}

func edgeKey(e genericEdge) string {
	return objKey(edgeOuter(e)) + "->" + edgeTargetKey(e) + vectorKey(e.args)
}

func edgeOuter(e genericEdge) types.Object {
	if e.outerFunc != nil {
		return e.outerFunc
	}
	return e.outerType
}

func edgeTargetKey(e genericEdge) string {
	if e.innerFunc != nil {
		return objKey(e.innerFunc)
	}
	return objKey(e.innerType)
}

func objKey(obj types.Object) string {
	pkg := ""
	if obj.Pkg() != nil {
		pkg = obj.Pkg().Path()
	}
	return pkg + "." + obj.Name()
}

func vectorKey(vector []types.Type) string {
	key := "["
	for _, t := range vector {
		key += types.TypeString(t, nil) + ";"
	}
	return key + "]"
}

// MentionsTypeParam reports whether a type's structure references any
// type parameter (the free-parameter test the prepass and closure share).
func MentionsTypeParam(t types.Type) bool { return mentionsTypeParamType(t) }
