package emit

import "strings"

// InterfacesContent renders the canonical interface-artifacts module
// (ADR-0008): every registered union's discriminated-union TYPE, its
// equality function, and (where any consumer needs one) its map-key
// encoder — each defined exactly once. Building one union def may
// reference nested unions, which register mid-build, so the emission
// runs to a fixed point over the growing registry. The module owns its
// artifacts, so references inside these defs are unqualified.
func InterfacesContent(module *Module) (string, error) {
	if module.Interfaces == nil || !module.ownsInterfaces {
		return "", nil
	}
	p := &printer{out: &strings.Builder{}, module: module}
	aliasDefs := map[string]string{}
	eqDefs := map[string]string{}
	keyDefs := map[string]string{}

	// Fixed point: building a union def can register nested unions and
	// mark earlier unions' keys as required, so iterate until every
	// registered union has its alias, equality, and (where required)
	// key emitted and nothing new appears.
	for {
		progressed := false
		for _, name := range module.Interfaces.Names() {
			t := module.Interfaces.Type(name)
			if _, done := aliasDefs[name]; !done {
				alias, err := p.buildUnionAliasDefinition(name, t)
				if err != nil {
					return "", err
				}
				aliasDefs[name] = alias
				eqDefs[name] = p.ifaceEqFn(t, name)
				progressed = true
			}
			if _, done := keyDefs[name]; !done && module.Interfaces.KeyRequired(name) {
				keyFn, err := p.ifaceKeyFn(t, name)
				if err != nil {
					return "", err
				}
				keyDefs[name] = keyFn
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}

	var body strings.Builder
	// Types first, then equality functions, then key encoders — every
	// reference resolves within the module regardless of order (function
	// and type references, not init-time use).
	for _, name := range module.Interfaces.Names() {
		body.WriteString(aliasDefs[name])
		body.WriteString("\n")
	}
	for _, name := range module.Interfaces.Names() {
		if def := eqDefs[name]; def != "" {
			body.WriteString(def)
			body.WriteString("\n")
		}
		if def := keyDefs[name]; def != "" {
			body.WriteString(def)
			body.WriteString("\n")
		}
	}
	return module.importLines() + body.String(), nil
}
