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
	imports map[string]ModuleImport
	used    map[string]bool
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
	return &Module{Pkg: pkg, PkgName: pkgName, ABI: abiImports, imports: imports, used: map[string]bool{}}
}

// symbol spells a reference to a package-level symbol: unqualified within
// the module's own package, alias-qualified (recording the import) for
// every other co-generated package.
func (m *Module) symbol(pkg, name string) (string, error) {
	if pkg == m.Pkg {
		return name, nil
	}
	imported, ok := m.imports[pkg]
	if !ok {
		return "", fmt.Errorf("reference to package %q which is not part of the translated unit", pkg)
	}
	m.used[pkg] = true
	return imported.Alias + "." + name, nil
}

// importLines renders the deterministic import block: the language-ABI
// modules, then every referenced co-generated package in sorted path
// order.
func (m *Module) importLines() string {
	var out strings.Builder
	fmt.Fprintf(&out, "import * as goabi from %q;\n", m.ABI.Ints)
	fmt.Fprintf(&out, "import * as gort from %q;\n", m.ABI.Runtime)
	fmt.Fprintf(&out, "import * as gosl from %q;\n", m.ABI.Slice)
	fmt.Fprintf(&out, "import * as goif from %q;\n", m.ABI.Iface)
	fmt.Fprintf(&out, "import * as goext from %q;\n", m.ABI.Extern)
	usedPaths := make([]string, 0, len(m.used))
	for path := range m.used {
		usedPaths = append(usedPaths, path)
	}
	sort.Strings(usedPaths)
	for _, path := range usedPaths {
		imported := m.imports[path]
		fmt.Fprintf(&out, "import * as %s from %q;\n", imported.Alias, imported.Specifier)
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
