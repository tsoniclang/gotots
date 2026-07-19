// Slice-region analysis: every slice-typed local variable joins a
// value-flow region; requirements accumulate from the operations the
// body performs, and the planner selects each region's representation.
// The region key is the variable's canonical unique binding name (the
// builder resolved shadows to distinct names, e.g. s and s$1), so two
// shadowing variables are DISTINCT regions and each keeps precise
// requirements — never conflated by a shared spelling. Any occurrence
// the walk cannot prove local — calls, returns, fields, containers,
// captures, boxed cells — escapes.
package ir

import (
	"github.com/tsoniclang/gotots/internal/plan"
)

// slicePlanner accumulates one function body's slice regions.
type slicePlanner struct {
	planner *plan.Planner
	regions map[string]int // canonical unique binding name -> region occurrence
}

// AnalyzeSlicePlans computes the representation of every slice-typed
// local variable in one generated body.
func AnalyzeSlicePlans(fn *Func) map[string]string {
	if fn.Body == nil {
		return nil
	}
	s := &slicePlanner{planner: plan.NewPlanner(), regions: map[string]int{}}
	// Parameters and results cross the function boundary.
	for _, param := range fn.Params {
		if param.Type.Kind == KindSlice {
			s.planner.Require(s.region(param.Name), plan.ReqEscape)
		}
	}
	for _, result := range fn.Results {
		if result.Name != "" && result.Type.Kind == KindSlice {
			s.planner.Require(s.region(result.Name), plan.ReqEscape)
		}
	}
	if fn.Receiver != nil && fn.Receiver.Type.Kind == KindSlice {
		s.planner.Require(s.region(fn.Receiver.Name), plan.ReqEscape)
	}
	s.walkBlock(fn.Body)
	out := map[string]string{}
	for name, id := range s.regions {
		out[name] = string(s.planner.SelectSlice(id))
	}
	return out
}

func (s *slicePlanner) region(name string) int {
	if id, has := s.regions[name]; has {
		return id
	}
	id := s.planner.NewRegion()
	s.regions[name] = id
	return id
}

// varRegion returns the region of a slice-typed local variable read,
// or -1 when the expression is not one.
func (s *slicePlanner) varRegion(e Expr) int {
	ref, isVar := e.(*VarRef)
	if !isVar || ref.Pkg != "" || ref.T.Kind != KindSlice {
		return -1
	}
	return s.region(ref.Name)
}

// walkExpr accumulates requirements from one expression tree. escaped
// marks contexts whose slice operands leave local control.
func (s *slicePlanner) walkExpr(e Expr) {
	switch n := e.(type) {
	case nil:
		return
	case *VarRef:
		// A bare read forwards nothing by itself; forwarding contexts
		// mark escapes explicitly.
	case *IsNil:
		if id := s.varRegion(n.X); id >= 0 {
			s.planner.Require(id, plan.ReqNilability)
		}
		s.walkExpr(n.X)
	case *SliceCap:
		if id := s.varRegion(n.X); id >= 0 {
			s.planner.Require(id, plan.ReqCapacity)
		}
		s.walkExpr(n.X)
	case *SliceReslice:
		// A reslice shares the operand's storage as a new first-class
		// value.
		if id := s.varRegion(n.X); id >= 0 {
			s.planner.Require(id, plan.ReqSharedView)
		}
		s.walkExpr(n.X)
		s.walkExpr(n.Low)
		s.walkExpr(n.High)
	case *SliceCopy:
		if id := s.varRegion(n.Dst); id >= 0 {
			s.planner.Require(id, plan.ReqSharedView)
		}
		if id := s.varRegion(n.Src); id >= 0 {
			s.planner.Require(id, plan.ReqSharedView)
		}
		s.walkExpr(n.Dst)
		s.walkExpr(n.Src)
	case *SliceAppend:
		// Appends are exact for a single owner; the assignment walk
		// erases this requirement again for x = append(x, ...) shapes.
		if id := s.varRegion(n.X); id >= 0 {
			s.planner.Require(id, plan.ReqSharedView)
		}
		s.walkExpr(n.X)
		for _, value := range n.Values {
			s.escapeIfSlice(value)
		}
	case *SliceAppendSlice:
		if id := s.varRegion(n.X); id >= 0 {
			s.planner.Require(id, plan.ReqSharedView)
		}
		s.walkExpr(n.X)
		s.escapeIfSlice(n.Source)
	case *SliceGet:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *SliceLen:
		s.walkExpr(n.X)
	case *SliceLit:
		for _, value := range n.Values {
			s.escapeIfSlice(value)
		}
	case *SliceMake:
		s.walkExpr(n.Length)
		s.walkExpr(n.Capacity)
	case *Closure:
		// The closure shares the enclosing regions: captures see the same
		// binding, so its own operations contribute their requirements
		// directly.
		s.walkBlock(n.Body)
	case *Binary:
		s.walkExpr(n.L)
		s.walkExpr(n.R)
	case *Unary:
		s.walkExpr(n.X)
	case *Convert:
		s.walkExpr(n.X)
	case *StringConvert:
		// The string-conversion ABI reads the operand's carrier shape, so
		// a []byte/[]rune source must not be a native array.
		s.escapeIfSlice(n.X)
	case *StringIndex:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *StringSlice:
		s.walkExpr(n.X)
		s.walkExpr(n.Low)
		s.walkExpr(n.High)
	case *StringLen:
		s.walkExpr(n.X)
	case *FieldLoad:
		s.walkExpr(n.X)
	case *FieldCellRef:
		s.walkExpr(n.Base)
	case *MapGet:
		s.walkExpr(n.Map)
		s.walkExpr(n.Key)
	case *MapLookup:
		s.walkExpr(n.Map)
		s.walkExpr(n.Key)
	case *MapLen:
		s.walkExpr(n.X)
	case *MapFrom:
		for i := range n.Keys {
			s.escapeIfSlice(n.Keys[i])
			s.escapeIfSlice(n.Values[i])
		}
	case *ArrayGet:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *ArrayLit:
		for _, value := range n.Values {
			s.escapeIfSlice(value)
		}
	case *ArrayLenExpr:
		s.walkExpr(n.X)
	case *ArrayEqual:
		s.walkExpr(n.L)
		s.walkExpr(n.R)
	case *ArraySliceView:
		s.walkExpr(n.X)
		s.walkExpr(n.Low)
		s.walkExpr(n.High)
	case *StructCopy:
		s.walkExpr(n.X)
	case *ParamCopy:
		s.walkExpr(n.X)
	case *Deref:
		s.walkExpr(n.X)
	case *AddrOf:
		s.walkExpr(n.X)
	case *MinMax:
		for _, argument := range n.Args {
			s.walkExpr(argument)
		}
	case *TupleSpread:
		s.walkExpr(n.X)
	case *TupleVariadicSpread:
		s.walkExpr(n.X)
	case *IfaceBox:
		s.escapeIfSlice(n.X)
	case *IfaceEqual:
		s.walkExpr(n.L)
		s.walkExpr(n.R)
	case *IfaceConcreteEqual:
		s.walkExpr(n.Iface)
		s.escapeIfSlice(n.Concrete)
	case *TypeAssert:
		s.walkExpr(n.X)
	case *IfaceAssert:
		s.walkExpr(n.X)
	case *StructNew:
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *Call:
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *ExternalCall:
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *MethodCall:
		s.escapeIfSlice(n.Recv)
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *ExternalMethodCall:
		s.escapeIfSlice(n.Recv)
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *IfaceCall:
		s.walkExpr(n.Recv)
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *DynCall:
		s.walkExpr(n.Fun)
		for _, argument := range n.Args {
			s.escapeIfSlice(argument)
		}
	case *ParamEqual:
		s.walkExpr(n.L)
		s.walkExpr(n.R)
	case *MethodValue:
		s.escapeIfSlice(n.Recv)
	case *ExternFieldRead:
		s.escapeIfSlice(n.X)
	case *ExternToOwned:
		s.escapeIfSlice(n.X)
	case *ExternLit:
		// Constructor arguments escape into the reviewed stub.
		for _, value := range n.Values {
			s.escapeIfSlice(value)
		}
	case *CellNew:
		s.escapeIfSlice(n.Zero)
	case *Const, *NilConst, *FuncRef, *StructZero, *ExternZero, *ExternVar,
		*ParamZero, *MapMake, *BoxedLoad, *BoxedRef:
		// Leaves contribute nothing.
	}
}

// escapeIfSlice marks slice-valued operands escaping and keeps walking.
func (s *slicePlanner) escapeIfSlice(e Expr) {
	if e == nil {
		return
	}
	if id := s.varRegion(e); id >= 0 {
		s.planner.Require(id, plan.ReqEscape)
	}
	s.walkExpr(e)
}

// walkStmt accumulates requirements from one statement.
func (s *slicePlanner) walkStmt(stmt Stmt) {
	switch n := stmt.(type) {
	case nil:
		return
	case *DeclStmt:
		for i, name := range n.Names {
			if name == "_" || i >= len(n.Types) {
				continue
			}
			if n.Types[i].Kind != KindSlice {
				if i < len(n.Values) {
					s.walkExpr(n.Values[i])
				}
				continue
			}
			if i < len(n.Values) {
				s.connectNamedValue(name, n.Values[i])
			}
		}
		if n.Tuple != nil {
			s.walkExpr(n.Tuple)
			for i, name := range n.Names {
				if name != "_" && i < len(n.Types) && n.Types[i].Kind == KindSlice {
					// Tuple slots come from calls: already escaped.
					s.planner.Require(s.region(name), plan.ReqEscape)
				}
			}
		}
	case *AssignStmt:
		for i, target := range n.Targets {
			variable, isVar := target.(VarTarget)
			if isVar && variable.Pkg == "" && variable.T.Kind == KindSlice && i < len(n.Values) {
				s.connectNamedValue(variable.Name, n.Values[i])
				continue
			}
			if isVar && variable.Pkg != "" && i < len(n.Values) {
				// A package variable escapes (assigned from any function).
				s.escapeIfSlice(n.Values[i])
				continue
			}
			if i < len(n.Values) {
				// Stores into fields, elements, maps, or cells escape.
				s.escapeIfSlice(n.Values[i])
			}
			s.walkTarget(target)
		}
		if n.Tuple != nil {
			s.walkExpr(n.Tuple)
		}
	case *CompoundStmt:
		s.walkTarget(n.Target)
		s.walkExpr(n.Rhs)
	case *ReturnStmt:
		for _, value := range n.Values {
			s.escapeIfSlice(value)
		}
		s.walkExpr(n.CallValue)
	case *IfStmt:
		s.walkStmt(n.Init)
		s.walkExpr(n.Cond)
		s.walkBlock(n.Then)
		s.walkStmt(n.Else)
	case *ForStmt:
		s.walkStmt(n.Init)
		s.walkExpr(n.Cond)
		s.walkStmt(n.Post)
		s.walkBlock(n.Body)
	case *Block:
		s.walkBlock(n)
	case *StmtSeq:
		for _, inner := range n.Stmts {
			s.walkStmt(inner)
		}
	case *RangeSlice:
		s.walkExpr(n.X)
		if n.Value != "" && n.VarT.Kind == KindSlice {
			// The element binding aliases container storage.
			s.planner.Require(s.region(n.Value), plan.ReqEscape)
		}
		s.walkBlock(n.Body)
	case *RangeInt:
		s.walkExpr(n.N)
		s.walkBlock(n.Body)
	case *RangeString:
		s.walkExpr(n.X)
		s.walkBlock(n.Body)
	case *RangeMap:
		s.walkExpr(n.X)
		if n.Value != "" && n.ValT.Kind == KindSlice {
			s.planner.Require(s.region(n.Value), plan.ReqEscape)
		}
		s.walkBlock(n.Body)
	case *RangeFunc:
		s.walkExpr(n.Seq)
		if n.Key != "" && n.KeyT.Kind == KindSlice {
			s.planner.Require(s.region(n.Key), plan.ReqEscape)
		}
		if n.Value != "" && n.ValT.Kind == KindSlice {
			s.planner.Require(s.region(n.Value), plan.ReqEscape)
		}
		s.walkBlock(n.Body)
	case *SwitchStmt:
		s.walkStmt(n.Init)
		s.walkExpr(n.Tag)
		for i := range n.Clauses {
			for _, value := range n.Clauses[i].Values {
				s.walkExpr(value)
			}
			s.walkBlock(n.Clauses[i].Body)
		}
	case *TypeSwitchStmt:
		s.walkStmt(n.Init)
		s.walkExpr(n.X)
		for i := range n.Clauses {
			s.walkBlock(n.Clauses[i].Body)
		}
	case *TryFinally:
		s.walkBlock(n.Body)
		s.walkBlock(n.Finally)
	case *LabeledStmt:
		s.walkStmt(n.Stmt)
	case *BoxedDecl:
		// The cell aliases the value as a first-class location.
		s.escapeIfSlice(n.Init)
	case *DeferPush:
		s.walkExpr(n.Call)
	case *ExprStmt:
		s.walkExpr(n.Call)
	case *MapDeleteStmt:
		s.walkExpr(n.Map)
		s.walkExpr(n.Key)
	case *MapClearStmt:
		s.walkExpr(n.Map)
	case *PanicStmt:
		s.escapeIfSlice(n.Value)
	}
}

// connectNamedValue joins a slice variable's region with its assigned
// value's region: direct variable reads connect; x = append(x, ...)
// keeps single ownership; appends into a different name alias
// reallocation-dependently, so their shared region takes the exact
// carrier; fresh literals and makes add nothing; unknown sources
// (calls, element reads, assertions) escape conservatively.
func (s *slicePlanner) connectNamedValue(targetName string, value Expr) {
	target := s.region(targetName)
	switch n := value.(type) {
	case *VarRef:
		if id := s.varRegion(n); id >= 0 {
			s.planner.Connect(target, id)
			if n.Name != targetName {
				// Two live names over one backing: aliasing observable.
				s.planner.Require(target, plan.ReqSharedView)
			}
			return
		}
	case *SliceAppend:
		if ref, isVar := n.X.(*VarRef); isVar && ref.Pkg == "" && ref.T.Kind == KindSlice {
			s.planner.Connect(target, s.region(ref.Name))
			if ref.Name != targetName {
				s.planner.Require(target, plan.ReqSharedView)
			}
			for _, appended := range n.Values {
				s.escapeIfSlice(appended)
			}
			return
		}
	case *SliceLit:
		for _, element := range n.Values {
			s.escapeIfSlice(element)
		}
		return
	case *SliceMake:
		s.walkExpr(n.Length)
		s.walkExpr(n.Capacity)
		return
	}
	// Unknown sources escape conservatively.
	s.planner.Require(target, plan.ReqEscape)
	s.walkExpr(value)
}

func (s *slicePlanner) walkTarget(target Target) {
	switch n := target.(type) {
	case *FieldTarget:
		s.walkExpr(n.X)
	case *MapTarget:
		s.walkExpr(n.Map)
		s.walkExpr(n.Key)
	case *SliceTarget:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *ArrayTarget:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *PointeeTarget:
		s.walkExpr(n.X)
	}
}

func (s *slicePlanner) walkBlock(block *Block) {
	if block == nil {
		return
	}
	for _, stmt := range block.Stmts {
		s.walkStmt(stmt)
	}
}
