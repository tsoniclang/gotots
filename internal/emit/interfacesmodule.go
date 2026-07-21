package emit

import "strings"

// InterfacesContent renders the canonical interface-artifacts module
// (ADR-0008): every registered union's discriminated-union TYPE, its
// equality function, and (where any consumer needs one) its map-key
// encoder — each defined exactly once. buildUnionAliasDefinition builds
// AND stores each union's eq/key; this reads the stored strings, so eq
// and key are built (and their symbols recorded) exactly once.
// Building one union def may register nested unions and mark earlier
// keys required, so emission runs to a fixed point over the growing
// registry. The module owns its artifacts, so internal references are
// unqualified.
func InterfacesContent(module *Module) (string, error) {
	if module.Interfaces == nil || !module.ownsInterfaces {
		return "", nil
	}
	p := &printer{out: &strings.Builder{}, module: module}
	aliasDefs := map[string]string{}

	for {
		progressed := false
		for _, name := range module.Interfaces.Names() {
			if _, done := aliasDefs[name]; done {
				continue
			}
			alias, err := p.buildUnionAliasDefinition(name, module.Interfaces.Type(name))
			if err != nil {
				return "", err
			}
			aliasDefs[name] = alias
			progressed = true
		}
		if !progressed {
			break
		}
	}

	var body strings.Builder
	for _, name := range module.Interfaces.Names() {
		body.WriteString(aliasDefs[name])
		body.WriteString("\n")
	}
	// Equality functions then key encoders, both stored by
	// buildUnionAliasDefinition, rendered in registry order (the module
	// bypasses RegisterIfaceAlias so aliasOrder is empty here).
	for _, name := range module.Interfaces.Names() {
		if def := module.ifaceEqFns[name]; def != "" {
			body.WriteString(def)
			body.WriteString("\n")
		}
	}
	for _, name := range module.Interfaces.Names() {
		if def := module.ifaceEqFns[name+"$key"]; def != "" {
			body.WriteString(def)
			body.WriteString("\n")
		}
	}
	return module.importLines() + body.String(), nil
}
