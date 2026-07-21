// Object-model family emission (ADR-0012): a struct in a recovered
// object-model family emits as a native TypeScript class on its proven
// inheritance spine — `abstract class Root` / `class Member extends
// Primary` — with own fields only, a super-chaining total constructor, a
// spine-aware value contract, synthesized abstract contract methods on
// the root, and members placed by their recovered disposition. Inherited
// members emit nothing, so the promoted forwarding chains and per-call
// dispatch switches disappear.
package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/objectmodel"
)

// familyClass is the resolved emission view of one family struct: its
// recovered class facts joined to the IR struct and the ancestor spine
// (root-first) whose own fields the constructor forwards through super.
type familyClass struct {
	structDecl *ir.Struct
	class      objectmodel.Class
	family     objectmodel.Family
	ancestors  []spineLevel
	// selfField is the family self-reference field (Node.data), dropped
	// from the root class: native inheritance collapses it.
	selfField string
	// structsByCanon resolves every family struct in the unit, so the root
	// can take a contract method's declared signature from an implementer.
	structsByCanon map[string]*ir.Struct
	// baseMembers is every member name (own field or declared method) present
	// on the primary-spine ancestors, plus the contract and value-contract
	// members every ancestor carries. A method whose name is here overrides an
	// inherited member and needs the `override` modifier (noImplicitOverride).
	baseMembers map[string]bool
}

// spineLevel is one ancestor class with the own fields it contributes to
// the flattened constructor (its primary-embedded field excluded).
type spineLevel struct {
	canon string
	own   []ir.Var
}

// familyMemberName renames a family method member whose name collides with a
// field somewhere in the family (the field shadows the method in Go); every
// reference — definition, call site, vtable adapter — uses the suffixed name.
// The collision set is unit-wide (sealed on the plan) so cross-package call
// sites rename identically.
func (m *Module) familyMemberName(name string) string {
	if m.ObjectModel != nil && m.ObjectModel.AccessorCollisions()[name] {
		return tsName(name) + "$"
	}
	return tsName(name)
}

// isFamilyClass reports whether a named type of this module is a placed
// object-model family class (so its dispatch, field access, construction,
// and vtable adapters take the native-inheritance form).
func (m *Module) isFamilyClass(name string) bool {
	if m.ObjectModel == nil {
		return false
	}
	_, ok := m.ObjectModel.Class(m.Pkg + "." + name)
	return ok
}

// familyNamedCanon returns the canonical name of a struct or
// pointer-to-struct type (Named carries the pointee name for a pointer).
func familyNamedCanon(t ir.Type) (string, bool) {
	if t.Named == "" || t.Pkg == "" {
		return "", false
	}
	return t.Pkg + "." + t.Named, true
}

// familyClassAndPrimary returns whether baseType is a placed family class
// and, if so, its recovered Class (for the primary-spine and family
// lookups the field-access and dispatch collapses need).
func (m *Module) familyClassAndFamily(baseType ir.Type) (objectmodel.Class, objectmodel.Family, bool) {
	if m.ObjectModel == nil {
		return objectmodel.Class{}, objectmodel.Family{}, false
	}
	canon, ok := familyNamedCanon(baseType)
	if !ok {
		return objectmodel.Class{}, objectmodel.Family{}, false
	}
	class, ok := m.ObjectModel.Class(canon)
	if !ok {
		return objectmodel.Class{}, objectmodel.Family{}, false
	}
	family, ok := m.ObjectModel.Family(class.Family)
	if !ok {
		return objectmodel.Class{}, objectmodel.Family{}, false
	}
	return class, family, true
}

// isFamilySelfRef reports whether X.field selects a family self-reference
// (X is a family class and field is its family's self-reference field).
// Such a selection collapses under native inheritance: X IS its own data.
func (m *Module) isFamilySelfRef(baseType ir.Type, field string) bool {
	_, family, ok := m.familyClassAndFamily(baseType)
	return ok && field == family.SelfField
}

// isFamilyPrimaryHop reports whether X.field selects the primary embedded
// base of a family class — an inherited-spine hop that collapses to X
// under native inheritance.
func (m *Module) isFamilyPrimaryHop(baseType ir.Type, field string) bool {
	class, _, ok := m.familyClassAndFamily(baseType)
	return ok && class.HasPrimary && field == class.Primary.Field
}

// familyClassOf resolves the family-class emission view for one of this
// module's own named types, using the module's struct index.
func (m *Module) familyClassOf(name string) (*familyClass, bool) {
	if m.ObjectModel == nil {
		return nil, false
	}
	structDecl, ok := m.StructsByCanon[m.Pkg+"."+name]
	if !ok {
		return nil, false
	}
	return resolveFamilyClass(m.ObjectModel, m.Pkg+"."+name, structDecl, m.StructsByCanon)
}

// resolveFamilyClass joins the sealed plan to the IR struct for canon,
// returning nil when canon is not a placed family class. structsByCanon
// resolves ancestor structs on the spine.
func resolveFamilyClass(plan *objectmodel.Plan, canon string, structDecl *ir.Struct,
	structsByCanon map[string]*ir.Struct) (*familyClass, bool) {
	if plan == nil {
		return nil, false
	}
	class, ok := plan.Class(canon)
	if !ok {
		return nil, false
	}
	family, ok := plan.Family(class.Family)
	if !ok {
		return nil, false
	}
	fc := &familyClass{structDecl: structDecl, class: class, family: family, structsByCanon: structsByCanon}
	if class.Root {
		fc.selfField = family.SelfField
	}
	var chain []string
	for cur := class; cur.HasPrimary; {
		chain = append(chain, cur.Primary.BaseCanon)
		next, ok := plan.Class(cur.Primary.BaseCanon)
		if !ok {
			return nil, false
		}
		cur = next
	}
	for i := len(chain) - 1; i >= 0; i-- {
		ancestorCanon := chain[i]
		ancestorStruct, ok := structsByCanon[ancestorCanon]
		if !ok {
			return nil, false
		}
		ancestorClass, _ := plan.Class(ancestorCanon)
		self := ""
		if ancestorClass.Root {
			self = family.SelfField
		}
		fc.ancestors = append(fc.ancestors, spineLevel{
			canon: ancestorCanon,
			own:   familyOwnFields(ancestorStruct, ancestorClass, self),
		})
	}
	fc.baseMembers = computeBaseMembers(fc)
	return fc, true
}

// computeBaseMembers collects every member name reachable on the primary
// spine ancestors — their own fields and declared methods, plus the
// contract methods and the value-contract members every ancestor carries —
// so an override can be detected by name.
func computeBaseMembers(fc *familyClass) map[string]bool {
	names := map[string]bool{}
	for _, level := range fc.ancestors {
		for _, field := range level.own {
			names[field.Name] = true
		}
		if s, ok := fc.structsByCanon[level.canon]; ok {
			for _, m := range s.Methods {
				names[m.Name] = true
			}
		}
	}
	// The contract methods are declared (abstract or default) on the root, so
	// every non-root class overrides them when it redefines one.
	for _, name := range fc.family.ContractMethods {
		names[name] = true
	}
	return names
}

// familyOwnFields returns the fields a family class declares itself:
// every IR field except the primary-embedded base (inherited via
// extends) and, for the root, the collapsed self-reference field.
// Secondary embedded components stay as owned fields.
func familyOwnFields(structDecl *ir.Struct, class objectmodel.Class, selfField string) []ir.Var {
	primaryField := ""
	if class.HasPrimary {
		primaryField = class.Primary.Field
	}
	out := make([]ir.Var, 0, len(structDecl.Fields))
	for _, field := range structDecl.Fields {
		if class.HasPrimary && field.Name == primaryField {
			continue
		}
		if selfField != "" && field.Name == selfField {
			continue
		}
		out = append(out, field)
	}
	return out
}

func (fc *familyClass) ownFields() []ir.Var {
	return familyOwnFields(fc.structDecl, fc.class, fc.selfField)
}

// transitiveOwnFields is every own field on the spine root-first then
// this class's own — the flattened total constructor parameter order.
func (fc *familyClass) transitiveOwnFields() []ir.Var {
	var out []ir.Var
	for _, level := range fc.ancestors {
		out = append(out, level.own...)
	}
	return append(out, fc.ownFields()...)
}

// ancestorConstructorArgs is the argument list forwarded to super: every
// ancestor level's own fields, root-first.
func (fc *familyClass) ancestorConstructorArgs() []ir.Var {
	var out []ir.Var
	for _, level := range fc.ancestors {
		out = append(out, level.own...)
	}
	return out
}

// isInheritedPromotion reports whether a promoted method is inherited
// through the primary spine (emit nothing) rather than delegated through
// a secondary component.
func (fc *familyClass) isInheritedPromotion(name string) bool {
	return fc.class.Methods[name] == objectmodel.MethodInherited
}

// methodModifier returns "override " when a method redefines a member
// present on a primary-spine ancestor (TypeScript's noImplicitOverride
// requires the modifier); the root never overrides, so it returns "".
func (fc *familyClass) methodModifier(name string) string {
	if fc.class.Root {
		return ""
	}
	if fc.baseMembers[name] {
		return "override "
	}
	return ""
}

// isTrampolineRemoved reports whether a root method is a self-reference
// trampoline removed to the abstract contract (no concrete body emitted).
func (fc *familyClass) isTrampolineRemoved(name string) bool {
	return fc.class.Methods[name] == objectmodel.MethodTrampolineRemoved
}

// rootDeclaresConcrete reports whether the root provides a concrete body
// for a method (a default the subclasses inherit), so the abstract
// contract synthesis skips it.
func (fc *familyClass) rootDeclaresConcrete(name string) bool {
	return fc.class.Methods[name] == objectmodel.MethodDeclared
}

// depth is the number of ancestors on the primary spine (0 for the root),
// so family classes can be emitted base-before-derived (TS classes are not
// hoisted for `extends`).
func (fc *familyClass) depth() int { return len(fc.ancestors) }

// familyDepth returns a named type's spine depth and whether it is a
// family class, for the base-before-derived emission order.
func familyDepth(m *Module, name string) (int, bool) {
	fc, ok := m.familyClassOf(name)
	if !ok {
		return 0, false
	}
	return fc.depth(), true
}

// printFamilyClass emits one family struct as its native class: header,
// own fields, super-chaining constructor, spine-aware value contract,
// synthesized abstract contract methods (root), declared method members,
// and secondary-component delegates. Inherited promotions are omitted.
func printFamilyClass(out *strings.Builder, module *Module, fc *familyClass,
	memberMethods, delegateMethods []*ir.Func, outcomes *[]BodyOutcome, artifacts *[]BodyArtifact) error {
	p := &printer{out: out, module: module}
	// A family class is emitted concrete with an exact zero-state: Go structs
	// are constructed and zeroed, so the class must be instantiable. Contract
	// methods the class does not provide dispatch to the zero value's nil
	// interface (a panic), exactly as Go's `n.data.M()` on a zero node does.
	header := "export class " + tsName(fc.structDecl.Name)
	if !fc.class.Root {
		header += " extends " + tsName(fc.class.Primary.BaseType)
	}
	p.line("%s {", header)
	p.indent++

	for _, field := range fc.ownFields() {
		spelled, err := p.tsType(field.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", fc.structDecl.ID, err)
		}
		if field.Cell {
			p.line("%s: gort$.GoCell<%s>;", field.Name, spelled)
		} else {
			p.line("%s: %s;", field.Name, spelled)
		}
	}

	if err := printFamilyConstructor(p, fc); err != nil {
		return err
	}
	if err := printFamilyValueContract(p, fc); err != nil {
		return err
	}
	if fc.class.Root {
		if err := printFamilyContractZeroState(p, fc, memberMethods); err != nil {
			return err
		}
	}

	// Declared methods that the planner keeps as ordinary members bind to
	// `this`; a family override reuses the leaf member emitter, prefixed
	// with `override` when it redefines an inherited member. A trampoline
	// removed to the abstract contract emits no concrete body (it would
	// recurse on itself and collide with the abstract declaration).
	for _, method := range memberMethods {
		if fc.isTrampolineRemoved(method.Name) {
			continue
		}
		out.WriteString("\n")
		m := method
		modifiers := fc.methodModifier(m.Name)
		err := emitTransactionalBody(out, module, m, "default", func(o *strings.Builder, frag *Module) error {
			return printMethodMember(o, frag, m, modifiers)
		}, outcomes, artifacts)
		if err != nil {
			return err
		}
	}
	// Exception-lowered declared methods keep their free-function body; the
	// class carries a one-line delegate so proven-receiver call sites stay
	// source-shaped. Trampolines removed to the abstract contract emit no
	// delegate.
	for _, method := range delegateMethods {
		if fc.isTrampolineRemoved(method.Name) {
			continue
		}
		if err := printMethodDelegate(p, fc.structDecl, method, fc.methodModifier(method.Name)); err != nil {
			return err
		}
	}
	// Secondary-component promotions delegate through the owned component
	// field; primary-spine promotions are inherited natively (omitted).
	for _, delegate := range fc.structDecl.Promoted {
		if fc.isInheritedPromotion(delegate.Name) {
			continue
		}
		if delegate.IfaceField {
			continue
		}
		if err := printPromotedMemberDelegate(p, delegate, fc.methodModifier(delegate.Name)); err != nil {
			return err
		}
	}

	p.indent--
	p.line("}")
	return nil
}

// printFamilyConstructor emits the flattened total constructor: every
// transitive own field root-first, forwarding the ancestor portion
// through super, then assigning this class's own fields.
func printFamilyConstructor(p *printer, fc *familyClass) error {
	params := make([]string, 0)
	for _, field := range fc.transitiveOwnFields() {
		spelled, err := p.tsType(field.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", fc.structDecl.ID, err)
		}
		params = append(params, tsName(field.Name)+": "+spelled)
	}
	p.line("constructor(%s) {", strings.Join(params, ", "))
	p.indent++
	if !fc.class.Root {
		superArgs := make([]string, 0)
		for _, field := range fc.ancestorConstructorArgs() {
			superArgs = append(superArgs, tsName(field.Name))
		}
		p.line("super(%s);", strings.Join(superArgs, ", "))
	}
	for _, field := range fc.ownFields() {
		if field.Cell {
			p.line("this.%s = { v: %s };", field.Name, tsName(field.Name))
		} else {
			p.line("this.%s = %s;", field.Name, tsName(field.Name))
		}
	}
	p.indent--
	p.line("}")
	return nil
}

// printFamilyValueContract emits the spine-aware value semantics over the
// transitive own fields with a covariant self return: goClone$ / goSet$ and a
// static goZero$. Every family class is concrete, so a non-root class always
// overrides the inherited contract. goEq$/goKey$ are emitted only when the
// concrete type is comparable / key-encodable (below).
func printFamilyValueContract(p *printer, fc *familyClass) error {
	self := tsName(fc.structDecl.Name)
	override := ""
	if !fc.class.Root {
		override = "override "
	}
	fields := fc.transitiveOwnFields()

	clone := make([]string, 0, len(fields))
	for _, field := range fields {
		expr, err := p.familyFieldClone(field)
		if err != nil {
			return err
		}
		clone = append(clone, expr)
	}
	p.line("%sgoClone$(): %s {", override, self)
	p.indent++
	p.line("return new %s(%s);", self, strings.Join(clone, ", "))
	p.indent--
	p.line("}")

	p.line("%sgoSet$(other: %s): void {", override, self)
	p.indent++
	for _, field := range fields {
		if err := p.familyFieldSet(field); err != nil {
			return err
		}
	}
	p.indent--
	p.line("}")

	zeros := make([]string, 0, len(fields))
	for _, field := range fields {
		zero, err := p.zeroLiteral(field.Type)
		if err != nil {
			return err
		}
		zeros = append(zeros, zero)
	}
	p.line("static %sgoZero$(): %s {", override, self)
	p.indent++
	p.line("return new %s(%s);", self, strings.Join(zeros, ", "))
	p.indent--
	p.line("}")

	// A comparable / key-encodable family type carries goEq$ / goKey$ over
	// its transitive own fields, so it can compare and key exactly like the
	// flat value struct. Comparability is monotone down the spine, so a
	// non-root class overrides the inherited operation.
	if fc.structDecl.Comparable {
		comparisons := make([]string, 0, len(fields))
		for _, field := range fields {
			this, other := "this."+field.Name, "other."+field.Name
			if field.Cell {
				this += ".v"
				other += ".v"
			}
			cmp, err := p.eqComponent(this, other, field.Type)
			if err != nil {
				return err
			}
			comparisons = append(comparisons, cmp)
		}
		if len(comparisons) == 0 {
			comparisons = append(comparisons, "true")
		}
		p.line("%sgoEq$(other: %s): boolean {", override, self)
		p.indent++
		p.line("return %s;", strings.Join(comparisons, " && "))
		p.indent--
		p.line("}")
	}
	if fc.structDecl.KeyEncodable {
		components := make([]string, 0, len(fields))
		for _, field := range fields {
			access := "this." + field.Name
			if field.Cell {
				access += ".v"
			}
			comp, err := p.keyComponent(access, field.Type)
			if err != nil {
				return err
			}
			components = append(components, comp)
		}
		if len(components) == 0 {
			components = append(components, `"z"`)
		}
		p.line("%sgoKey$(): string {", override)
		p.indent++
		p.line("return %s;", strings.Join(components, ` + "|" + `))
		p.indent--
		p.line("}")
	}
	return nil
}

// familyFieldClone spells one field's clone expression accessing the
// field directly on `this` (inherited or own).
func (p *printer) familyFieldClone(field ir.Var) (string, error) {
	access := "this." + field.Name
	if field.Cell {
		return access + ".v", nil
	}
	if field.Type.Kind == ir.KindIface && field.Type.TypeParamName != "" {
		return "this.clone$" + field.Type.TypeParamName + "(" + access + ")", nil
	}
	switch field.Type.Kind {
	case ir.KindStruct:
		return access + ".goClone$()", nil
	case ir.KindArray:
		cloneElem, err := p.arrayElemClone(*field.Type.Elem)
		if err != nil {
			return "", err
		}
		return "gosl$.goArrayClone(" + access + ", " + cloneElem + ")", nil
	case ir.KindExternal:
		callee, err := p.module.symbol(field.Type.Pkg, externCloneSymbol(field.Type.Named))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", callee, access), nil
	}
	return access, nil
}

// familyFieldSet emits one field's in-place overwrite statement.
func (p *printer) familyFieldSet(field ir.Var) error {
	this, other := "this."+field.Name, "other."+field.Name
	if field.Cell {
		p.line("%s.v = %s.v;", this, other)
		return nil
	}
	switch field.Type.Kind {
	case ir.KindStruct:
		p.line("%s.goSet$(%s);", this, other)
	case ir.KindArray:
		setElem, err := p.arrayElemSet(*field.Type.Elem)
		if err != nil {
			return err
		}
		p.line("gosl$.goArraySetAll(%s, %s, %s);", this, other, setElem)
	case ir.KindExternal:
		callee, err := p.module.symbol(field.Type.Pkg, externSetSymbol(field.Type.Named))
		if err != nil {
			return err
		}
		p.line("%s(%s, %s);", callee, this, other)
	default:
		p.line("%s = %s;", this, other)
	}
	return nil
}

// printFamilyContractZeroState emits, on the root, a concrete zero-state
// body for every virtual-contract method the root does not itself provide.
// On a zero value the self-interface is nil, so `n.data.M()` panics; the
// root class reproduces that exactly (concrete subclasses override with the
// real behavior). This keeps the root instantiable — Go structs are zeroed
// and constructed — instead of an abstract class.
func printFamilyContractZeroState(p *printer, fc *familyClass, memberMethods []*ir.Func) error {
	for _, name := range fc.family.ContractMethods {
		if fc.rootDeclaresConcrete(name) {
			continue
		}
		sig := findImplementerSignature(fc, name, memberMethods)
		if sig == nil {
			return fmt.Errorf("%s: no implementer signature for contract method %q", fc.structDecl.ID, name)
		}
		params := make([]string, 0, len(sig.Params))
		for _, param := range sig.Params {
			spelled, err := p.tsType(param.Type)
			if err != nil {
				return fmt.Errorf("%s: %w", fc.structDecl.ID, err)
			}
			params = append(params, tsName(param.Name)+": "+spelled)
		}
		result, err := p.tsResultType(sig.Results)
		if err != nil {
			return fmt.Errorf("%s: %w", fc.structDecl.ID, err)
		}
		p.line("%s(%s): %s {", tsName(name), strings.Join(params, ", "), result)
		p.indent++
		p.line("gort$.goPanicNil();")
		p.indent--
		p.line("}")
	}
	return nil
}

// findImplementerSignature resolves a contract method's signature. The
// root does not declare it, so it is taken from an implementer — the
// contract signature is uniform across implementers by the interface, so
// any implementing struct's method serves.
func findImplementerSignature(fc *familyClass, name string, memberMethods []*ir.Func) *ir.Func {
	for _, m := range memberMethods {
		if m.Name == name {
			return m
		}
	}
	for _, member := range fc.family.Members {
		canon := memberCanon(fc, member)
		s, ok := fc.structsByCanon[canon]
		if !ok {
			continue
		}
		for _, m := range s.Methods {
			if m.Name == name {
				return m
			}
		}
	}
	return nil
}

// printFamilyStructNew reshapes a composite literal's field-ordered
// constructor args into the family class's flattened transitive-own-field
// order: each transitive-own field takes the literal's provided value
// (matched by name against this struct's own fields) or a zero for
// inherited/omitted fields — matching Go's zero-valued embedded base.
// Keyed-reordered literals stage their provided values in source order
// first (side-effect order), exactly as the flat emitter does.
func (p *printer) printFamilyStructNew(fc *familyClass, n *ir.StructNew, class string) (string, error) {
	argIndexByName := make(map[string]int, len(fc.structDecl.Fields))
	for i, field := range fc.structDecl.Fields {
		argIndexByName[field.Name] = i
	}
	fields := fc.transitiveOwnFields()

	if len(n.EvalOrder) == 0 {
		ctorArgs := make([]string, 0, len(fields))
		for _, field := range fields {
			arg, err := p.familyCtorArg(argIndexByName, n, field, nil)
			if err != nil {
				return "", err
			}
			ctorArgs = append(ctorArgs, arg)
		}
		return fmt.Sprintf("new %s(%s)", class, strings.Join(ctorArgs, ", ")), nil
	}

	stagedParam := map[int]string{}
	var params, values []string
	for k, argIndex := range n.EvalOrder {
		spelled, err := p.tsType(n.Args[argIndex].Type())
		if err != nil {
			return "", err
		}
		name := fmt.Sprintf("$v%d", k)
		stagedParam[argIndex] = name
		params = append(params, name+": "+spelled)
		printed, err := p.printExpr(n.Args[argIndex])
		if err != nil {
			return "", err
		}
		values = append(values, printed)
	}
	ctorArgs := make([]string, 0, len(fields))
	for _, field := range fields {
		arg, err := p.familyCtorArg(argIndexByName, n, field, stagedParam)
		if err != nil {
			return "", err
		}
		ctorArgs = append(ctorArgs, arg)
	}
	return fmt.Sprintf("((%s) => new %s(%s))(%s)",
		strings.Join(params, ", "), class, strings.Join(ctorArgs, ", "), strings.Join(values, ", ")), nil
}

// familyCtorArg spells one transitive-own field's constructor argument:
// the literal's provided value (a staged parameter name when reordered),
// or a zero for a field the literal omits (including every inherited
// field, which Go zero-values through the embedded base).
func (p *printer) familyCtorArg(argIndexByName map[string]int, n *ir.StructNew, field ir.Var, stagedParam map[int]string) (string, error) {
	idx, ok := argIndexByName[field.Name]
	if !ok {
		return p.zeroLiteral(field.Type)
	}
	if stagedParam != nil {
		if name, staged := stagedParam[idx]; staged {
			return name, nil
		}
	}
	return p.printExpr(n.Args[idx])
}

// memberCanon spells a family member's canonical key in the emitting
// unit's struct map (pkg-qualified type name).
func memberCanon(fc *familyClass, member string) string {
	prefix := strings.TrimSuffix(fc.class.Canon, fc.structDecl.Name)
	return prefix + member
}

// familyVtableEntries builds the boxed-dispatch adapters for a family
// class. Under native inheritance every method — declared, overridden,
// inherited, or component-delegated — is a class member reachable as
// `$r.<name>(...)`, so each vtable entry is a direct member call keyed by
// the method's canonical slot, with no promoted forwarding chain and no
// displaced free-function call. The box machinery is preserved (a node
// still boxes into interfaces other than its own contract); only the
// adapter shape becomes native. Parameters type through the class member
// itself (`Parameters<Self["member"]>`), which resolves inherited members.
func (p *printer) familyVtableEntries(info RttiInfo) ([]string, []string, error) {
	self := tsName(info.TypeName)
	var valueSet, pointerSet []string
	add := func(slot, member string, valueReceiver bool) {
		value := fmt.Sprintf("%s: ($r: %s, ...$a: Parameters<%s[%q]>) => $r.%s(...$a)",
			slot, self, self, member, member)
		pointer := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: Parameters<%s[%q]>) => gort$.goNilCheck<%s>($r).%s(...$a)",
			slot, self, self, member, self, member)
		if valueReceiver {
			valueSet = append(valueSet, value)
		}
		pointerSet = append(pointerSet, pointer)
	}

	methods := append([]*ir.Func{}, info.Methods...)
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	for _, method := range methods {
		slot := requireIdentity(method.Slot, "vtable slot for method "+method.Name)
		add(slot, p.module.familyMemberName(method.Name), !method.PointerReceiver)
	}
	promoted := append([]ir.PromotedDelegate{}, info.Promoted...)
	sort.Slice(promoted, func(i, j int) bool { return promoted[i].Name < promoted[j].Name })
	for _, delegate := range promoted {
		slot := requireIdentity(delegate.Slot, "vtable slot for promoted method "+delegate.Name)
		add(slot, p.module.familyMemberName(delegate.Name), delegate.ValueReceiver)
	}
	return valueSet, pointerSet, nil
}
