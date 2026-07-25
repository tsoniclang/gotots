package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

func (m *schemaModel) assignInlineAliases() error {
	for schemaName, members := range m.normalizedNodeMember {
		for _, value := range members {
			if value.List != "" || len(value.Type) < 2 || !m.memberIsChild(value) {
				continue
			}
			alias := inlineAliasName(value.Owner, value.Name)
			seen := make(map[int]struct{})
			for _, typeName := range value.Type {
				indices, err := m.resolveNodeSet(typeName, make(map[string]bool))
				if err != nil {
					return fmt.Errorf("%s.%s: %w", schemaName, value.Name, err)
				}
				for _, index := range indices {
					seen[index] = struct{}{}
				}
			}
			for index := range seen {
				m.nodes[index].InlineAliases = append(m.nodes[index].InlineAliases, alias)
			}
		}
	}
	for index := range m.nodes {
		sort.Strings(m.nodes[index].InlineAliases)
	}
	return nil
}

func (m *schemaModel) assignGenericAliases() {
	for alias, owner := range m.instantiationOwner {
		kindAlias, broad := m.kindAliases[m.instantiationKind[alias]]
		if !broad {
			continue
		}
		kindSet := make(map[string]struct{})
		for _, kind := range m.expandKindMembers(kindAlias) {
			kindSet[kind] = struct{}{}
		}
		for index := range m.nodes {
			node := &m.nodes[index]
			if node.SchemaName != owner {
				continue
			}
			if _, included := kindSet[node.Kind]; included {
				node.GenericAliases = append(node.GenericAliases, alias)
			}
		}
	}
	for index := range m.nodes {
		sort.Strings(m.nodes[index].GenericAliases)
	}
}

func (m *schemaModel) memberIsVisible(value member) bool {
	return !value.GoOnly && !value.NoTS && !value.NoGo
}

func (m *schemaModel) memberIsFactory(value member) bool {
	return m.memberIsVisible(value) && !value.NoFactory
}

func (m *schemaModel) memberIsEncoded(value member) bool {
	return !value.GoOnly && !value.NoTS && !value.NoGo
}

func (m *schemaModel) memberIsChild(value member) bool {
	if value.List != "" {
		return value.List != "raw" || m.typeNameIsNode(value.Type[0])
	}
	if len(value.Type) == 0 {
		return false
	}
	for _, name := range value.Type {
		if !m.typeNameIsNode(name) {
			return false
		}
	}
	return true
}

func (m *schemaModel) typeNameIsNode(name string) bool {
	if name == "Node" {
		return true
	}
	if _, exists := m.raw.Nodes.Definitions[name]; exists {
		return true
	}
	if _, exists := m.raw.Bases[name]; exists {
		return true
	}
	if _, exists := m.raw.Nodes.Aliases[name]; exists {
		return true
	}
	if _, exists := m.instantiationOwner[name]; exists {
		return true
	}
	return false
}

func (m *schemaModel) goMemberType(owner string, value member) (string, error) {
	if len(value.Type) == 0 {
		return "", fmt.Errorf("%s.%s has no type", owner, value.Name)
	}
	var result string
	if len(value.Type) > 1 {
		if m.memberIsChild(value) {
			result = inlineAliasName(value.Owner, value.Name)
		} else if allSyntaxKinds(value.Type) {
			result = inlineKindName(value.Owner, value.Name)
		} else {
			return "", fmt.Errorf("%s.%s has unsupported mixed union %q", owner, value.Name, value.Type)
		}
	} else {
		name := value.Type[0]
		switch name {
		case "bool", "boolean":
			result = "bool"
		case "int":
			result = "int"
		case "string":
			result = "string"
		case "NodeFlags", "TokenFlags":
			result = name
		case "any":
			return "", fmt.Errorf("%s.%s uses unsupported any", owner, value.Name)
		default:
			if strings.HasPrefix(name, "SyntaxKind.") {
				result = "SyntaxKind"
			} else {
				result = name
			}
		}
	}
	if value.List != "" {
		result = "[]" + result
	}
	return result, nil
}

func allSyntaxKinds(names []string) bool {
	for _, name := range names {
		if !strings.HasPrefix(name, "SyntaxKind.") {
			return false
		}
	}
	return true
}

func inlineAliasName(owner string, field string) string {
	return owner + exportedName(field)
}

func inlineKindName(owner string, field string) string {
	return owner + exportedName(field) + "Kind"
}

func exportedName(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "jsdoc") {
		return "JSDoc" + name[len("jsdoc"):]
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func parameterName(name string) string {
	value := strings.ToLower(name[:1]) + name[1:]
	switch value {
	case "type":
		return "typeNode"
	case "range", "map", "func", "var", "defer", "go", "select", "interface",
		"struct", "chan", "package", "import", "return", "fallthrough", "default",
		"case", "switch", "else", "if", "for", "const", "break", "continue":
		return value + "Value"
	default:
		return value
	}
}

func (m *schemaModel) factoryMembers(node concreteNode) []member {
	var result []member
	for _, value := range node.Members {
		if m.memberIsFactory(value) {
			result = append(result, value)
		}
	}
	return result
}

func (m *schemaModel) encodedMembers(node concreteNode) []member {
	var result []member
	for _, value := range node.Members {
		if m.memberIsEncoded(value) {
			result = append(result, value)
		}
	}
	return result
}

func uniqueStrings(values ...[]string) []string {
	seen := make(map[string]struct{})
	for _, group := range values {
		for _, value := range group {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsAny(values []member) bool {
	for _, value := range values {
		if slices.Contains(value.Type, "any") {
			return true
		}
	}
	return false
}
