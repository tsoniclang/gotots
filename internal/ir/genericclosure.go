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

// --- Per-declaration type-parameter requirements ---------------------------
//
// A generic declaration whose shape uses map[P]V requires every CONCRETE
// binding of P to be a SameValueZero key: the declaration then admits with
// the direct GoMap carrier, and each instantiation SITE binding a non-SVZ
// carrier fails closed (the guarded instantiation never exists in emitted
// code, so the carrier stays exact for everything that does). Requirements
// propagate backwards along free-parameter edges: an outer parameter
// forwarded into a required position inherits the requirement.

// RequireParamSVZKey records that the declaration's i-th type parameter
// must bind a SameValueZero key carrier.
func (s Scope) RequireParamSVZKey(obj types.Object, index int) {
	key := objKey(obj)
	reqs := s.paramKeyReqs[key]
	for len(reqs) <= index {
		reqs = append(reqs, false)
	}
	reqs[index] = true
	s.paramKeyReqs[key] = reqs
}

// ParamRequiresSVZKey reports whether the declaration's i-th type
// parameter requires a SameValueZero key binding.
func (s Scope) ParamRequiresSVZKey(obj types.Object, index int) bool {
	reqs := s.paramKeyReqs[objKey(obj)]
	return index < len(reqs) && reqs[index]
}

// RequireParamKeyCapture records the SOFT requirement: the declaration
// forwards key$P (a key-encodable class capture) without keying a map by
// it — any Go-legal binding admits.
func (s Scope) RequireParamKeyCapture(obj types.Object, index int) {
	key := objKey(obj)
	reqs := s.paramCaptureReqs[key]
	for len(reqs) <= index {
		reqs = append(reqs, false)
	}
	reqs[index] = true
	s.paramCaptureReqs[key] = reqs
}

// RequireParamPtr records the pointer-family requirement: the
// declaration's shape mentions *P, so its emission splits on the
// binding's pointer representation (identity vs cell).
func (s Scope) RequireParamPtr(obj types.Object, index int) {
	key := objKey(obj)
	reqs := s.paramPtrReqs[key]
	for len(reqs) <= index {
		reqs = append(reqs, false)
	}
	reqs[index] = true
	s.paramPtrReqs[key] = reqs
}

// ParamRequiresPtr reports the pointer-family requirement.
func (s Scope) ParamRequiresPtr(obj types.Object, index int) bool {
	reqs := s.paramPtrReqs[objKey(obj)]
	return index < len(reqs) && reqs[index]
}

// ParamRequiresKeyOp reports whether the declaration's i-th parameter
// takes key$P in the factory protocol — the union of the HARD map-key
// requirement and the SOFT capture requirement.
func (s Scope) ParamRequiresKeyOp(obj types.Object, index int) bool {
	if s.ParamRequiresSVZKey(obj, index) {
		return true
	}
	reqs := s.paramCaptureReqs[objKey(obj)]
	return index < len(reqs) && reqs[index]
}

// CollectParamKeyRequirements scans one generic declaration's shape for
// map keys typed by its own parameters (fields, embedded types, method
// signatures for a type; the full signature for a function) and records
// the SVZ-key requirement per parameter. Local map declarations inside
// generic function bodies are covered by the same walk over the
// declaration's syntax types recorded by the caller.
func (s Scope) CollectParamKeyRequirements(obj types.Object, params *types.TypeParamList, shape []types.Type) {
	if params == nil || params.Len() == 0 {
		return
	}
	// A generic STRUCT whose origin is structurally key-encodable carries
	// goKey$ and captures key$P for every parameter: record the
	// requirement on all of them, so the requirement store is the ONE
	// source both the class capture mask and the call-side masks read.
	if typeName, isType := obj.(*types.TypeName); isType {
		if named, isNamed := typeName.Type().(*types.Named); isNamed && originStructKeyCapturing(named.Origin()) {
			for i := range params.Len() {
				s.RequireParamKeyCapture(obj, i)
			}
		}
	}
	index := map[*types.TypeParam]int{}
	nameIndex := map[string]int{}
	for i := range params.Len() {
		index[params.At(i)] = i
		nameIndex[params.At(i).Obj().Name()] = i
	}
	indexOf := func(p *types.TypeParam) (int, bool) {
		if i, mine := index[p]; mine {
			return i, true
		}
		// A method signature's receiver parameters are DISTINCT objects
		// from the type's own — same declaration, matched by name.
		i, mine := nameIndex[p.Obj().Name()]
		return i, mine
	}
	seen := map[types.Type]bool{}
	var walk func(t types.Type)
	walk = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch u := types.Unalias(t).(type) {
		case *types.Map:
			if p, ok := types.Unalias(u.Key()).(*types.TypeParam); ok {
				if i, mine := indexOf(p); mine {
					s.RequireParamSVZKey(obj, i)
				}
			}
			walk(u.Key())
			walk(u.Elem())
		case *types.Pointer:
			if p, ok := types.Unalias(u.Elem()).(*types.TypeParam); ok {
				if i, mine := indexOf(p); mine {
					s.RequireParamPtr(obj, i)
				}
			}
			walk(u.Elem())
		case *types.Slice:
			walk(u.Elem())
		case *types.Array:
			walk(u.Elem())
		case *types.Chan:
			walk(u.Elem())
		case *types.Struct:
			for i := range u.NumFields() {
				walk(u.Field(i).Type())
			}
		case *types.Tuple:
			for i := range u.Len() {
				walk(u.At(i).Type())
			}
		case *types.Signature:
			walk(u.Params())
			walk(u.Results())
		case *types.Named:
			if args := u.TypeArgs(); args != nil {
				if u.TypeParams() != nil && u.TypeParams().Len() > 0 {
					// A generic instantiation binding one of OUR parameters:
					// record the edge so the requirement fixed point flows
					// the callee's key requirements (its own map keys AND
					// its capture mask) back to the binding parameter.
					mentionsMine := false
					argList := make([]types.Type, args.Len())
					for i := range args.Len() {
						argList[i] = args.At(i)
						if p, ok := types.Unalias(args.At(i)).(*types.TypeParam); ok {
							if _, mine := index[p]; mine {
								mentionsMine = true
							}
						}
					}
					if mentionsMine {
						s.AddGenericEdge(obj, u.Origin().Obj(), argList)
					}
				}
				for i := range args.Len() {
					walk(args.At(i))
				}
			}
			if u.TypeParams() == nil || u.TypeParams().Len() == 0 {
				walk(u.Underlying())
			}
		}
	}
	for _, t := range shape {
		walk(t)
	}
}

// PropagateParamRequirements closes the requirements backwards over the
// free-parameter edges to a fixed point: an outer parameter bound into a
// required inner position is itself required.
func (s Scope) PropagateParamRequirements() {
	edges := *s.genericEdges
	sort.SliceStable(edges, func(i, j int) bool { return edgeKey(edges[i]) < edgeKey(edges[j]) })
	for changed := true; changed; {
		changed = false
		for _, edge := range edges {
			inner := edgeTarget(edge)
			outer := edgeOuter(edge)
			params := edge.outerParams()
			if params == nil {
				continue
			}
			for j, arg := range edge.args {
				hard := s.ParamRequiresSVZKey(inner, j)
				soft := s.ParamRequiresKeyOp(inner, j)
				ptr := s.ParamRequiresPtr(inner, j)
				if !hard && !soft && !ptr {
					continue
				}
				p, ok := types.Unalias(arg).(*types.TypeParam)
				if !ok {
					continue
				}
				for i := range params.Len() {
					if params.At(i) != p {
						continue
					}
					if hard && !s.ParamRequiresSVZKey(outer, i) {
						s.RequireParamSVZKey(outer, i)
						changed = true
					}
					if soft && !s.ParamRequiresKeyOp(outer, i) {
						s.RequireParamKeyCapture(outer, i)
						changed = true
					}
					if ptr && !s.ParamRequiresPtr(outer, i) {
						s.RequireParamPtr(outer, i)
						changed = true
					}
				}
			}
		}
	}
}

func edgeTarget(e genericEdge) types.Object {
	if e.innerFunc != nil {
		return e.innerFunc
	}
	return e.innerType
}

// originStructKeyCapturing is a STRUCTURAL key-encodability walk of a
// generic origin struct (no builder context: bare parameters and
// pointers admit; scalars admit; anything else conservatively requires
// the key — over-requiring is safe, the operation derivation is total
// over admitted bindings).
func originStructKeyCapturing(origin *types.Named) bool {
	structType, ok := origin.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := range structType.NumFields() {
		ft := structType.Field(i).Type()
		if _, isParam := types.Unalias(ft).(*types.TypeParam); isParam {
			continue
		}
		if _, isPtr := types.Unalias(ft).Underlying().(*types.Pointer); isPtr {
			continue
		}
		if basic, isBasic := types.Unalias(ft).Underlying().(*types.Basic); isBasic {
			_ = basic
			continue
		}
		return false
	}
	return true
}
