// Whole-package emission: the transactional per-body loop (typed
// outcomes, overlay commit-on-success) over structs, methods, carriers,
// package variables, functions, and the initialization sequence.
package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/implid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/plan"
)

// Package prints one translated package into a single TypeScript module:
// classes for named structs (each followed by its method functions and
// rtti constants), then carrier-type methods and rttis, then package
// variables, then functions, each in sorted name order. module carries
// the package identity, the language-ABI specifiers, and the
// co-generated import environment; the body is printed first so only
// referenced packages are imported.
// Body-emission outcomes form a CLOSED classification. Emitted bodies
// produce no outcome record. KnownUnsupported arises ONLY from a typed,
// producer-owned ir.Unsupported condition reaching emission: the body is
// reclassified (IR-admitted -> unimplemented) and materializes as a typed
// throwing placeholder. EmitterDefect is every other emission failure — an
// unknown error, a missing lowering, an invariant violation — a compiler
// defect that HARD-FAILS the acceptance gates; a diagnostic placeholder
// keeps the file analyzable, but the defect disposition is never counted
// as an ordinary placeholder and the gate stays failed until it is fixed.
// Classification is by typed error identity (ir.AsUnsupported), never by
// error-message text.
const (
	OutcomeKnownUnsupported = "known-unsupported"
	OutcomeEmitterDefect    = "emitter-defect"
)

// BodyOutcome records one body whose emission did not complete normally.
type BodyOutcome struct {
	ID   string
	Kind string // OutcomeKnownUnsupported | OutcomeEmitterDefect
	Err  string
	Site *ir.UnsupportedSite
}

// BodyArtifact is one body's canonical analysis-only serialization: the
// exact emitted fragment (spec 01) — never a module, never imported or
// executed — retained for EVERY lowered body, withheld packages included.
type BodyArtifact struct {
	ID string
	// ImplementationID = ID + "/" + specialization key (ADR-0010): the
	// artifact/hash owner. Two artifacts with one ImplementationID are
	// a collision and fail generation.
	ImplementationID string
	Text             string
}

// emitTransactionalBody prints one body through print into an overlay and
// commits only on success. On failure it classifies the outcome, re-emits
// the body as a typed throwing placeholder (signature-only — a fresh
// overlay, so nothing of the failed attempt leaks), and records the
// outcome. Only a placeholder that itself fails to emit aborts the
// package (its signature is a declaration-level defect).
func emitTransactionalBody(body *strings.Builder, module *Module, fn *ir.Func, specKey string,
	print func(*strings.Builder, *Module) error, outcomes *[]BodyOutcome, artifacts *[]BodyArtifact) error {
	fragment := module.Overlay()
	var buf strings.Builder
	before := len(module.emissions)
	err := print(&buf, fragment)
	if err == nil {
		fragment.Commit()
		if !module.emissionsSince(before, fn.ID, EmissionBodyPlaceholder) {
			module.recordEmission(fn.ID, EmissionBody, specKey)
		}
		body.WriteString(buf.String())
		*artifacts = append(*artifacts, BodyArtifact{ID: fn.ID, ImplementationID: implid.MustNew(fn.ID, specKey).String(), Text: buf.String()})
		return nil
	}
	outcome := BodyOutcome{ID: fn.ID, Kind: OutcomeEmitterDefect, Err: err.Error()}
	if unsupported, ok := ir.AsUnsupported(err); ok {
		site := ir.SiteOf(unsupported)
		outcome = BodyOutcome{ID: fn.ID, Kind: OutcomeKnownUnsupported, Err: err.Error(), Site: &site}
	}
	fn.Placeholder = true
	retry := module.Overlay()
	var placeholder strings.Builder
	if placeholderErr := print(&placeholder, retry); placeholderErr != nil {
		return fmt.Errorf("%s: placeholder emission failed (%v) after body failure: %w", fn.ID, placeholderErr, err)
	}
	retry.Commit()
	body.WriteString(placeholder.String())
	*artifacts = append(*artifacts, BodyArtifact{ID: fn.ID, ImplementationID: implid.MustNew(fn.ID, specKey).String(), Text: placeholder.String()})
	*outcomes = append(*outcomes, outcome)
	return nil
}

func Package(module *Module, decls Decls) (string, []BodyOutcome, []BodyArtifact, error) {
	sortedStructs := append([]*ir.Struct{}, decls.Structs...)
	sort.Slice(sortedStructs, func(i, j int) bool { return sortedStructs[i].Name < sortedStructs[j].Name })
	sortedMethodDecls := append([]Method{}, decls.Methods...)
	sort.Slice(sortedMethodDecls, func(i, j int) bool {
		if sortedMethodDecls[i].TypeName != sortedMethodDecls[j].TypeName {
			return sortedMethodDecls[i].TypeName < sortedMethodDecls[j].TypeName
		}
		return sortedMethodDecls[i].Fn.Name < sortedMethodDecls[j].Fn.Name
	})
	sorted := append([]*ir.Func{}, decls.Functions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var body strings.Builder
	var outcomes []BodyOutcome
	var artifacts []BodyArtifact
	for _, structDecl := range sortedStructs {
		body.WriteString("\n")
		sortedMethods := append([]*ir.Func{}, structDecl.Methods...)
		sort.Slice(sortedMethods, func(i, j int) bool { return sortedMethods[i].Name < sortedMethods[j].Name })
		var memberMethods, freeMethods, delegateMethods []*ir.Func
		for _, method := range sortedMethods {
			if module.MethodEmissionFor(methodPlanKey(method)) == plan.MethodOrdinaryNilChecked {
				memberMethods = append(memberMethods, method)
			} else {
				freeMethods = append(freeMethods, method)
				if len(structDecl.TypeParams) == 0 && len(method.TypeParams) == 0 &&
					!structDecl.FamilyEnc && !structDecl.FamilyPtrCell && !method.Placeholder {
					delegateMethods = append(delegateMethods, method)
				}
			}
		}
		if err := printStruct(&body, module, structDecl, memberMethods, delegateMethods, &outcomes, &artifacts); err != nil {
			return "", nil, nil, err
		}
		for _, method := range freeMethods {
			body.WriteString("\n")
			className := familyName(structDecl)
			familyEnc := structDecl.FamilyEnc
			familyPtr := structDecl.FamilyPtrCell
			err := emitTransactionalBody(&body, module, method, specializationKey(familyEnc || method.FamilyEnc, familyPtr || method.FamilyPtrCell), func(out *strings.Builder, frag *Module) error {
				return printMethodFunctionVariant(out, frag, className, method, familyEnc, familyPtr)
			}, &outcomes, &artifacts)
			if err != nil {
				return "", nil, nil, err
			}
		}
		for _, delegate := range structDecl.Promoted {
			if !delegate.IfaceField {
				continue
			}
			body.WriteString("\n")
			structT := ir.Type{Kind: ir.KindStruct, Go: structDecl.ID, Named: structDecl.Name, Pkg: module.Pkg}
			if err := printIfaceDelegate(&body, module, structDecl.Name, structT, delegate); err != nil {
				return "", nil, nil, err
			}
		}
		if len(structDecl.TypeParams) == 0 {
			body.WriteString("\n")
			info := RttiInfo{
				TypeName: structDecl.Name, Exported: structDecl.Exported,
				Pointer: true, Comparable: structDecl.Comparable, HasEq: structDecl.Comparable,
				Methods: structDecl.Methods, Promoted: structDecl.Promoted,
			}
			if err := printRtti(&body, module, info); err != nil {
				return "", nil, nil, err
			}
			if err := printVtables(&body, module, info); err != nil {
				return "", nil, nil, err
			}
		}
	}
	for _, method := range sortedMethodDecls {
		body.WriteString("\n")
		typeName, fn := method.TypeName, method.Fn
		err := emitTransactionalBody(&body, module, fn, specializationKey(fn.FamilyEnc, fn.FamilyPtrCell), func(out *strings.Builder, frag *Module) error {
			return printMethodFunction(out, frag, typeName, fn)
		}, &outcomes, &artifacts)
		if err != nil {
			return "", nil, nil, err
		}
	}
	sortedCarriers := append([]CarrierType{}, decls.CarrierTypes...)
	sort.Slice(sortedCarriers, func(i, j int) bool { return sortedCarriers[i].Name < sortedCarriers[j].Name })
	if len(sortedCarriers) > 0 {
		body.WriteString("\n")
	}
	for _, carrier := range sortedCarriers {
		var carrierMethods []*ir.Func
		for _, method := range sortedMethodDecls {
			if method.TypeName == carrier.Name {
				carrierMethods = append(carrierMethods, method.Fn)
			}
		}
		// The carrier's named alias: union members and signatures can
		// reference the Go type name directly.
		underlyingSpelled, err := (&printer{out: &body, module: module}).tsType(carrier.Underlying)
		if err != nil {
			return "", nil, nil, err
		}
		carrierGenerics := ""
		if len(carrier.TypeParams) > 0 {
			carrierGenerics = "<" + strings.Join(carrier.TypeParams, ", ") + ">"
		}
		fmt.Fprintf(&body, "export type %s%s = %s;\n", tsName(carrier.Name), carrierGenerics, underlyingSpelled)
		// A named carrier's payload is a primitive: comparable directly,
		// and its pointer type (a cell) boxes with its own rtti.
		carrierInfo := RttiInfo{
			TypeName: carrier.Name, Exported: carrier.Exported,
			Pointer: true, Comparable: true,
			Methods: carrierMethods,
		}
		if err := printRtti(&body, module, carrierInfo); err != nil {
			return "", nil, nil, err
		}
		if err := printVtables(&body, module, carrierInfo); err != nil {
			return "", nil, nil, err
		}
	}
	sortedVars := append([]PackageVar{}, decls.Vars...)
	sort.Slice(sortedVars, func(i, j int) bool { return sortedVars[i].Name < sortedVars[j].Name })
	if len(sortedVars) > 0 {
		body.WriteString("\n")
	}
	// Every variable declares with its zero value; initializers run below
	// in the type checker's dependency order — exactly Go's sequence.
	ordered := map[int]PackageVar{}
	maxOrder := -1
	for _, packageVar := range sortedVars {
		if packageVar.Order >= 0 {
			ordered[packageVar.Order] = packageVar
			if packageVar.Order > maxOrder {
				maxOrder = packageVar.Order
			}
		}
		if packageVar.Blank {
			continue
		}
		p := &printer{out: &body, module: module}
		spelled, err := p.tsType(packageVar.Type)
		if err != nil {
			return "", nil, nil, err
		}
		zero, err := p.zeroLiteral(packageVar.Type)
		if err != nil {
			return "", nil, nil, err
		}
		export := "export "
		p.line("%slet %s: %s = %s;", export, tsName(packageVar.Name), spelled, zero)
	}
	for _, function := range sorted {
		body.WriteString("\n")
		fn := function
		err := emitTransactionalBody(&body, module, fn, specializationKey(fn.FamilyEnc, fn.FamilyPtrCell), func(out *strings.Builder, frag *Module) error {
			return printFunc(out, frag, fn)
		}, &outcomes, &artifacts)
		if err != nil {
			return "", nil, nil, err
		}
	}
	if maxOrder >= 0 || len(decls.InitCalls) > 0 {
		body.WriteString("\n")
		p := &printer{out: &body, module: module}
		for order := 0; order <= maxOrder; order++ {
			packageVar, has := ordered[order]
			if !has {
				continue
			}
			if packageVar.Placeholder {
				// The initializer is a recorded unsupported construct: its
				// order slot fails closed with the exact body identity.
				module.recordEmission(packageVar.ID, EmissionInitializerPlaceholder, "default")
				p.line("gort$.goBodyUnimplemented(%q);", packageVar.ID)
				continue
			}
			module.recordEmission(packageVar.ID, EmissionInitializer, "default")
			init, err := p.printExpr(packageVar.Init)
			if err != nil {
				return "", nil, nil, err
			}
			var target ir.Target = ir.VarTarget{Name: packageVar.Name, Pkg: module.Pkg, T: packageVar.Type}
			if packageVar.Blank {
				target = ir.BlankTarget{}
			}
			if err := p.printStore(target, init); err != nil {
				return "", nil, nil, err
			}
		}
		for _, initCall := range decls.InitCalls {
			p.line("%s();", initCall)
		}
	}
	// Build every reserved union alias's deferred definition now: the body is
	// fully spelled (all aliases reserved) and newModule has populated every
	// external adapter type, so external members carry their exact vtable
	// type rather than an incomplete Record<never,never>.
	if err := finalizeUnionAliases(module); err != nil {
		return "", nil, nil, err
	}
	return module.importLines() + module.aliasLines() + module.eqFnLines() + body.String(), outcomes, artifacts, nil
}
