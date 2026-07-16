package emit

import (
	"fmt"
	"sort"
	"strings"
)

// ModuleImport is one co-generated package importable from a module.
type ModuleImport struct {
	Alias     string
	Specifier string // ESM specifier with explicit .js
}

// ExternMethod is one recorded external method: its source name (the
// stub symbol derives from it) and its canonical dispatch identity.
type ExternMethod struct {
	Name string
	Key  string
	// Adapter is the pre-spelled exactly typed vtable arrow delegating
	// to the stub export (built where the obligation's signature is
	// known); AdapterType is its exact arrow TYPE for union-member
	// spelling.
	Adapter     string
	AdapterType string
}

// ABIImports carries the language-ABI module specifiers for one module.
type ABIImports struct {
	Ints    string
	Runtime string
	Slice   string
	Iface   string
	Extern  string
}

// Module is the emission context of one generated package module: its Go
// package path, the language-ABI specifiers, and deterministic aliases
// for every co-generated package it may reference. Imports of
// co-generated packages are recorded during printing and emitted only
// when referenced.
type Module struct {
	Pkg string
	// PkgName is the Go package name, qualifying type displays in
	// runtime messages.
	PkgName string
	ABI     ABIImports
	// ExternMethods maps each external named type ("pkg.Type") to its
	// unit-recorded methods (name plus canonical dispatch identity) in
	// sorted order: the static dispatch tables of external rttis.
	ExternMethods map[string][]ExternMethod
	imports       map[string]ModuleImport
	used          map[string]bool
	// typeUsed marks packages referenced only in type positions: their
	// imports emit as `import type` — erased, adding no runtime edge and
	// no initialization (ADR-0004).
	typeUsed map[string]bool
	// ifaceAliases collects the closed union aliases this module spells,
	// keyed by alias name, emitted after imports.
	ifaceAliases map[string]string
	aliasOrder   []string
	// Withheld reports packages whose modules are not in the bundle:
	// union aliases exclude their classes (nothing of theirs can box at
	// runtime — their code does not run) and never reference their
	// absent files.
	Withheld func(pkg string) bool
	// initEdges are co-generated packages this package imports whose
	// module must still be imported (evaluated) even when no symbol
	// reference survives — folded constants and type-only uses erase the
	// reference but not the initialization dependency.
	initEdges map[string]bool
}

// Symbol exposes the module's qualified-symbol spelling.
func (m *Module) Symbol(pkg, name string) (string, error) { return m.symbol(pkg, name) }

// typeSymbol spells a qualified symbol used only in a TYPE position:
// the package import may then be type-only (erased).
func (m *Module) typeSymbol(pkg, name string) (string, error) {
	if pkg == m.Pkg {
		return tsName(name), nil
	}
	imported, ok := m.imports[pkg]
	if !ok {
		return "", fmt.Errorf("package %s is not importable from %s", pkg, m.Pkg)
	}
	m.typeUsed[pkg] = true
	return imported.Alias + "." + tsName(name), nil
}

// RegisterIfaceAlias records one closed-union alias; returns whether it
// was newly added.
func (m *Module) RegisterIfaceAlias(name, declaration string) bool {
	if _, exists := m.ifaceAliases[name]; exists {
		return false
	}
	m.ifaceAliases[name] = declaration
	m.aliasOrder = append(m.aliasOrder, name)
	return true
}

// aliasLines renders the union aliases in first-registration order.
func (m *Module) aliasLines() string {
	if len(m.aliasOrder) == 0 {
		return ""
	}
	var out strings.Builder
	for _, name := range m.aliasOrder {
		out.WriteString(m.ifaceAliases[name])
		out.WriteString("\n")
	}
	return out.String()
}

// CoGeneratedImports returns every co-generated package this module
// actually imports in its emitted output: symbol references (including
// interface-dispatch branch targets) and initialization edges. These
// are the real dependency edges for withholding.
func (m *Module) CoGeneratedImports() []string {
	seen := map[string]bool{}
	for path := range m.used {
		seen[path] = true
	}
	for path := range m.initEdges {
		seen[path] = true
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// RequireInitEdge records that this package imports a co-generated
// package for its initialization side effects; the import is emitted
// even without a surviving symbol reference.
func (m *Module) RequireInitEdge(pkg string) {
	if pkg == m.Pkg {
		return
	}
	if _, importable := m.imports[pkg]; importable {
		m.initEdges[pkg] = true
	}
}

// NewModule builds the emission context for one generated module.
// specifiers maps every co-generated package path to its ESM specifier
// from this module's directory (the module's own path is ignored).
// Aliases are package-path segments with a "$" suffix: Go identifiers can
// never contain "$", so an alias can never collide with any source-named
// symbol, parameter, or local, and never shadows or is shadowed.
func NewModule(pkg, pkgName string, abiImports ABIImports, specifiers map[string]string) *Module {
	var paths []string
	for path := range specifiers {
		if path != pkg {
			paths = append(paths, path)
		}
	}
	aliases := assignAliases(paths)
	imports := make(map[string]ModuleImport, len(paths))
	for _, path := range paths {
		imports[path] = ModuleImport{Alias: aliases[path], Specifier: specifiers[path]}
	}
	return &Module{Pkg: pkg, PkgName: pkgName, ABI: abiImports, imports: imports,
		used: map[string]bool{}, initEdges: map[string]bool{},
		typeUsed: map[string]bool{}, ifaceAliases: map[string]string{}}
}

// symbol spells a reference to a package-level symbol: unqualified within
// the module's own package, alias-qualified (recording the import) for
// every other co-generated package.
func (m *Module) symbol(pkg, name string) (string, error) {
	if pkg == m.Pkg {
		return tsName(name), nil
	}
	imported, ok := m.imports[pkg]
	if !ok {
		return "", fmt.Errorf("reference to package %q which is not part of the translated unit", pkg)
	}
	m.used[pkg] = true
	return imported.Alias + "." + tsName(name), nil
}

// importLines renders the deterministic import block: the language-ABI
// modules, then every referenced co-generated package in sorted path
// order.
func (m *Module) importLines() string {
	var out strings.Builder
	fmt.Fprintf(&out, "import * as goabi$ from %q;\n", m.ABI.Ints)
	fmt.Fprintf(&out, "import * as gort$ from %q;\n", m.ABI.Runtime)
	fmt.Fprintf(&out, "import * as gosl$ from %q;\n", m.ABI.Slice)
	fmt.Fprintf(&out, "import * as goif$ from %q;\n", m.ABI.Iface)
	fmt.Fprintf(&out, "import * as goext$ from %q;\n", m.ABI.Extern)
	emit := map[string]bool{}
	for path := range m.used {
		emit[path] = true
	}
	for path := range m.initEdges {
		emit[path] = true
	}
	typeOnly := make([]string, 0, len(m.typeUsed))
	for path := range m.typeUsed {
		if !emit[path] {
			typeOnly = append(typeOnly, path)
		}
	}
	usedPaths := make([]string, 0, len(emit))
	for path := range emit {
		usedPaths = append(usedPaths, path)
	}
	sort.Strings(usedPaths)
	sort.Strings(typeOnly)
	for _, path := range usedPaths {
		imported := m.imports[path]
		fmt.Fprintf(&out, "import * as %s from %q;\n", imported.Alias, imported.Specifier)
	}
	for _, path := range typeOnly {
		imported := m.imports[path]
		// Erased: no runtime module edge, no initialization (ADR-0004).
		fmt.Fprintf(&out, "import type * as %s from %q;\n", imported.Alias, imported.Specifier)
	}
	return out.String()
}

// assignAliases derives one deterministic alias per package path: the
// sanitized last path segment plus "$", widened leftward segment by
// segment while any two paths collide, with a sorted-order index as the
// final tiebreak for paths whose full sanitized spellings coincide.
func assignAliases(paths []string) map[string]string {
	sorted := append([]string{}, paths...)
	sort.Strings(sorted)
	segments := make([][]string, len(sorted))
	depth := make([]int, len(sorted))
	for i, path := range sorted {
		segments[i] = strings.Split(path, "/")
		depth[i] = 1
	}

	alias := func(i int) string {
		parts := segments[i][len(segments[i])-depth[i]:]
		sanitized := make([]string, len(parts))
		for j, part := range parts {
			sanitized[j] = sanitizeIdent(part)
		}
		return strings.Join(sanitized, "$") + "$"
	}

	for {
		widened := false
		byAlias := map[string][]int{}
		for i := range sorted {
			byAlias[alias(i)] = append(byAlias[alias(i)], i)
		}
		for _, group := range byAlias {
			if len(group) < 2 {
				continue
			}
			for _, i := range group {
				if depth[i] < len(segments[i]) {
					depth[i]++
					widened = true
				}
			}
		}
		if !widened {
			break
		}
	}

	out := make(map[string]string, len(sorted))
	byAlias := map[string][]int{}
	for i := range sorted {
		byAlias[alias(i)] = append(byAlias[alias(i)], i)
	}
	for _, group := range byAlias {
		for position, i := range group {
			spelled := alias(i)
			if len(group) > 1 {
				spelled = fmt.Sprintf("%s%d$", spelled, position)
			}
			out[sorted[i]] = spelled
		}
	}
	return out
}

// sanitizeIdent maps one path segment onto identifier characters.
func sanitizeIdent(segment string) string {
	var out strings.Builder
	for i, r := range segment {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if isDigit && i == 0 {
			out.WriteByte('_')
		}
		if isLetter || isDigit {
			out.WriteRune(r)
		} else {
			out.WriteByte('_')
		}
	}
	return out.String()
}
