package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

func (m *schemaModel) assignUnionAliases() error {
	for _, name := range sortedKeys(m.raw.Nodes.Aliases) {
		if _, isBaseAlias, err := m.nodeAlias(name); err != nil {
			return err
		} else if isBaseAlias {
			continue
		}
		m.unionAliases = append(m.unionAliases, name)
		indices, err := m.resolveNodeSet(name, make(map[string]bool))
		if err != nil {
			return err
		}
		for _, index := range indices {
			m.nodes[index].UnionAliases = append(m.nodes[index].UnionAliases, name)
		}
	}
	for index := range m.nodes {
		sort.Strings(m.nodes[index].UnionAliases)
	}
	return nil
}

func (m *schemaModel) nodeAlias(name string) ([]string, bool, error) {
	rawValue, exists := m.raw.Nodes.Aliases[name]
	if !exists {
		return nil, false, fmt.Errorf("unknown node alias %s", name)
	}
	var base struct {
		Base string `json:"base"`
	}
	if err := json.Unmarshal(rawValue, &base); err == nil && base.Base != "" {
		return []string{base.Base}, true, nil
	}
	var members []string
	if err := json.Unmarshal(rawValue, &members); err != nil {
		return nil, false, fmt.Errorf("decode node alias %s: %w", name, err)
	}
	return members, false, nil
}

func (m *schemaModel) resolveNodeSet(name string, visiting map[string]bool) ([]int, error) {
	if visiting[name] {
		return nil, fmt.Errorf("node alias cycle at %s", name)
	}
	visiting[name] = true
	defer delete(visiting, name)

	if indices, exists := m.nodesBySchema[name]; exists {
		return slices.Clone(indices), nil
	}
	if index, exists := m.concreteByName[name]; exists {
		return []int{index}, nil
	}
	if kind, exists := m.instantiationKind[name]; exists {
		if members, broad := m.kindAliases[kind]; broad {
			return m.resolveKindSet(members), nil
		}
		if index, exists := m.concreteByName[name]; exists {
			return []int{index}, nil
		}
		return nil, fmt.Errorf("instantiation alias %s has no concrete node", name)
	}
	if members, exists := m.kindAliases[name]; exists {
		return m.resolveKindSet(m.expandKindMembers(members)), nil
	}
	if _, exists := m.raw.Bases[name]; exists {
		var result []int
		for index, node := range m.nodes {
			definition := m.raw.Nodes.Definitions[node.SchemaName]
			if slices.Contains(m.allRawBases(definition.Extends), name) {
				result = append(result, index)
			}
		}
		return result, nil
	}
	if _, exists := m.raw.Nodes.Aliases[name]; exists {
		members, baseAlias, err := m.nodeAlias(name)
		if err != nil {
			return nil, err
		}
		if baseAlias {
			return m.resolveNodeSet(members[0], visiting)
		}
		seen := make(map[int]struct{})
		for _, memberName := range members {
			indices, err := m.resolveNodeSet(memberName, visiting)
			if err != nil {
				return nil, fmt.Errorf("%s member %s: %w", name, memberName, err)
			}
			for _, index := range indices {
				seen[index] = struct{}{}
			}
		}
		result := make([]int, 0, len(seen))
		for index := range seen {
			result = append(result, index)
		}
		sort.Ints(result)
		return result, nil
	}
	return nil, fmt.Errorf("cannot resolve node set %s", name)
}

func (m *schemaModel) resolveKindSet(kinds []string) []int {
	seen := make(map[int]struct{})
	for _, kind := range kinds {
		for _, index := range m.concreteByKind[kind] {
			seen[index] = struct{}{}
		}
	}
	result := make([]int, 0, len(seen))
	for index := range seen {
		result = append(result, index)
	}
	sort.Ints(result)
	return result
}

func (m *schemaModel) expandKindMembers(members []string) []string {
	var result []string
	for _, name := range members {
		if nested, exists := m.kindAliases[name]; exists {
			result = append(result, m.expandKindMembers(nested)...)
		} else {
			result = append(result, name)
		}
	}
	return result
}

func (m *schemaModel) allRawBases(names []string) []string {
	seen := make(map[string]struct{})
	var visit func(string)
	visit = func(name string) {
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		for _, parent := range m.raw.Bases[name].Extends {
			visit(parent)
		}
	}
	for _, name := range names {
		visit(name)
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
