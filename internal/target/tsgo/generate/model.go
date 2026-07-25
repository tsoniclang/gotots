package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type rawSchema struct {
	Schema string             `json:"$schema"`
	Bases  map[string]rawBase `json:"bases"`
	Nodes  rawNodes           `json:"nodes"`
	Kinds  rawKinds           `json:"kinds"`
}

type rawNodes struct {
	Definitions map[string]rawNode         `json:"definitions"`
	Aliases     map[string]json.RawMessage `json:"aliases"`
	ListAliases map[string]string          `json:"listAliases"`
}

type rawKinds struct {
	Elements []json.RawMessage          `json:"elements"`
	Markers  []rawKindMarker            `json:"markers"`
	Aliases  map[string]json.RawMessage `json:"aliases"`
}

type rawKindMarker struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type rawBase struct {
	TS      string               `json:"ts"`
	TSName  string               `json:"tsName"`
	Brand   string               `json:"brand"`
	Extends []string             `json:"extends"`
	Fields  map[string]rawMember `json:"fields"`
}

type rawNode struct {
	Kind                 stringList         `json:"kind"`
	Extends              []string           `json:"extends"`
	Members              []rawMember        `json:"members"`
	TypeParameters       []rawTypeParameter `json:"typeParameters"`
	InstantiationAliases map[string]string  `json:"instantiationAliases"`
	HandWritten          bool               `json:"handWritten"`
	HandWrittenVisitor   bool               `json:"handWrittenVisitor"`
	GenerateSubtreeFacts bool               `json:"generateSubtreeFacts"`
	Arena                bool               `json:"arena"`
	TSName               string             `json:"tsName"`
}

type rawTypeParameter struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
	Default    string `json:"default"`
}

type rawMember struct {
	Name      string     `json:"name"`
	Inherited bool       `json:"inherited"`
	Type      stringList `json:"type"`
	Optional  *bool      `json:"optional"`
	List      string     `json:"list"`
	Visit     string     `json:"visit"`
	TypeGuard string     `json:"typeGuard"`
	Private   *bool      `json:"private"`
	GoOnly    *bool      `json:"goOnly"`
	NoTS      *bool      `json:"noTS"`
	NoGo      *bool      `json:"noGo"`
	NoFactory *bool      `json:"noFactory"`
	Bitmask   string     `json:"bitmask"`
}

type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*s = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

type member struct {
	Owner     string
	Name      string
	Type      []string
	Optional  bool
	List      string
	Private   bool
	GoOnly    bool
	NoTS      bool
	NoGo      bool
	NoFactory bool
	Bitmask   string
}

type enumValue struct {
	Name  string
	Value uint32
	Alias bool
}

type concreteNode struct {
	Name           string
	SchemaName     string
	Kind           string
	Bases          []string
	Members        []member
	UnionAliases   []string
	InlineAliases  []string
	GenericAliases []string
	VariantOf      string
	Constructible  bool
	UnsupportedAny bool
}

type schemaModel struct {
	directory            string
	raw                  rawSchema
	syntaxKinds          []enumValue
	syntaxKindByName     map[string]uint32
	canonicalKinds       []string
	kindAliases          map[string][]string
	nodes                []concreteNode
	nodesBySchema        map[string][]int
	concreteByName       map[string]int
	concreteByKind       map[string][]int
	instantiationOwner   map[string]string
	instantiationKind    map[string]string
	unionAliases         []string
	baseGoOnly           map[string]bool
	normalizedNodeMember map[string][]member
}

func loadModel(directory string) (*schemaModel, error) {
	data, err := os.ReadFile(filepath.Join(directory, "upstream", "ast.json"))
	if err != nil {
		return nil, err
	}
	var schema rawSchema
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&schema); err != nil {
		return nil, fmt.Errorf("decode ast.json: %w", err)
	}
	kinds, err := parseEnumFile(filepath.Join(directory, "upstream", "syntaxKind.enum.ts"), "SyntaxKind")
	if err != nil {
		return nil, err
	}
	model := &schemaModel{
		directory:            directory,
		raw:                  schema,
		syntaxKinds:          kinds,
		syntaxKindByName:     make(map[string]uint32, len(kinds)),
		kindAliases:          make(map[string][]string),
		nodesBySchema:        make(map[string][]int),
		concreteByName:       make(map[string]int),
		concreteByKind:       make(map[string][]int),
		instantiationOwner:   make(map[string]string),
		instantiationKind:    make(map[string]string),
		baseGoOnly:           make(map[string]bool),
		normalizedNodeMember: make(map[string][]member),
	}
	for _, kind := range kinds {
		model.syntaxKindByName[kind.Name] = kind.Value
	}
	if err := model.buildKindModel(); err != nil {
		return nil, err
	}
	if err := model.buildNodeModel(); err != nil {
		return nil, err
	}
	return model, nil
}

func (m *schemaModel) buildKindModel() error {
	for _, rawElement := range m.raw.Kinds.Elements {
		var name string
		if err := json.Unmarshal(rawElement, &name); err == nil {
			m.canonicalKinds = append(m.canonicalKinds, name)
			continue
		}
		var object struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(rawElement, &object); err != nil {
			return fmt.Errorf("decode kind element: %w", err)
		}
		if object.Name != "" {
			m.canonicalKinds = append(m.canonicalKinds, object.Name)
		}
	}
	markerValues := make(map[string]string, len(m.raw.Kinds.Markers))
	for _, marker := range m.raw.Kinds.Markers {
		markerValues[marker.Name] = marker.Value
	}
	var resolveMarker func(string) string
	resolveMarker = func(name string) string {
		value, ok := markerValues[name]
		if !ok {
			return name
		}
		return resolveMarker(value)
	}
	for name, rawAlias := range m.raw.Kinds.Aliases {
		var members []string
		if err := json.Unmarshal(rawAlias, &members); err == nil {
			m.kindAliases[name] = members
			continue
		}
		var ranged struct {
			Range [2]string `json:"range"`
		}
		if err := json.Unmarshal(rawAlias, &ranged); err != nil {
			return fmt.Errorf("decode kind alias %s: %w", name, err)
		}
		first := slices.Index(m.canonicalKinds, resolveMarker(ranged.Range[0]))
		last := slices.Index(m.canonicalKinds, resolveMarker(ranged.Range[1]))
		if first < 0 || last < first {
			return fmt.Errorf("kind alias %s has invalid range %q", name, ranged.Range)
		}
		m.kindAliases[name] = slices.Clone(m.canonicalKinds[first : last+1])
	}
	for index, name := range m.canonicalKinds {
		value, exists := m.syntaxKindByName[name]
		if !exists {
			return fmt.Errorf("canonical syntax kind %s is absent from syntaxKind.enum.ts", name)
		}
		if value != uint32(index) {
			return fmt.Errorf("syntax kind %s = %d, schema position is %d", name, value, index)
		}
	}
	if count, exists := m.syntaxKindByName["Count"]; !exists || count != uint32(len(m.canonicalKinds)) {
		return fmt.Errorf("SyntaxKind.Count = %d, canonical kinds = %d", count, len(m.canonicalKinds))
	}
	return nil
}

func (m *schemaModel) buildNodeModel() error {
	for name, definition := range m.raw.Nodes.Definitions {
		for alias, kind := range definition.InstantiationAliases {
			m.instantiationOwner[alias] = name
			m.instantiationKind[alias] = kind
		}
	}
	for name := range m.raw.Bases {
		m.baseGoOnly[name] = m.isGoOnlyBase(name)
	}
	nodeNames := sortedKeys(m.raw.Nodes.Definitions)
	for _, name := range nodeNames {
		members, err := m.normalizeNodeMembers(name)
		if err != nil {
			return err
		}
		m.normalizedNodeMember[name] = members
	}
	for _, name := range nodeNames {
		definition := m.raw.Nodes.Definitions[name]
		if len(definition.TypeParameters) != 0 {
			continue
		}
		kinds, variant := m.nodeKinds(name)
		if len(kinds) == 0 {
			return fmt.Errorf("node %s has no syntax kind", name)
		}
		if variant {
			for _, kind := range kinds {
				m.addConcrete(concreteNode{
					Name:           kind,
					SchemaName:     name,
					Kind:           kind,
					Bases:          m.visibleBases(definition.Extends),
					Members:        slices.Clone(m.normalizedNodeMember[name]),
					VariantOf:      name,
					Constructible:  true,
					UnsupportedAny: m.hasAnyMember(name),
				})
			}
			continue
		}
		nodeName := definition.TSName
		if nodeName == "" {
			nodeName = name
		}
		m.addConcrete(concreteNode{
			Name:           nodeName,
			SchemaName:     name,
			Kind:           kinds[0],
			Bases:          m.visibleBases(definition.Extends),
			Members:        slices.Clone(m.normalizedNodeMember[name]),
			Constructible:  !definition.HandWritten || name == "SourceFile",
			UnsupportedAny: m.hasAnyMember(name),
		})
	}
	for _, genericName := range []string{"Token", "KeywordExpression"} {
		definition := m.raw.Nodes.Definitions[genericName]
		aliases := sortedKeys(definition.InstantiationAliases)
		for _, alias := range aliases {
			kind := definition.InstantiationAliases[alias]
			if _, broad := m.kindAliases[kind]; broad {
				continue
			}
			m.addConcrete(concreteNode{
				Name:          alias,
				SchemaName:    genericName,
				Kind:          kind,
				Bases:         m.visibleBases(definition.Extends),
				Members:       nil,
				VariantOf:     genericName,
				Constructible: true,
			})
		}
	}
	if err := m.assignUnionAliases(); err != nil {
		return err
	}
	if err := m.assignInlineAliases(); err != nil {
		return err
	}
	m.assignGenericAliases()
	sort.Slice(m.nodes, func(left, right int) bool {
		leftKind := m.syntaxKindByName[m.nodes[left].Kind]
		rightKind := m.syntaxKindByName[m.nodes[right].Kind]
		if leftKind != rightKind {
			return leftKind < rightKind
		}
		return m.nodes[left].Name < m.nodes[right].Name
	})
	m.reindexNodes()
	return nil
}

func (m *schemaModel) addConcrete(node concreteNode) {
	index := len(m.nodes)
	m.nodes = append(m.nodes, node)
	m.nodesBySchema[node.SchemaName] = append(m.nodesBySchema[node.SchemaName], index)
	m.concreteByName[node.Name] = index
	m.concreteByKind[node.Kind] = append(m.concreteByKind[node.Kind], index)
}

func (m *schemaModel) reindexNodes() {
	clear(m.nodesBySchema)
	clear(m.concreteByName)
	clear(m.concreteByKind)
	for index, node := range m.nodes {
		m.nodesBySchema[node.SchemaName] = append(m.nodesBySchema[node.SchemaName], index)
		m.concreteByName[node.Name] = index
		m.concreteByKind[node.Kind] = append(m.concreteByKind[node.Kind], index)
	}
}

func (m *schemaModel) nodeKinds(name string) ([]string, bool) {
	definition := m.raw.Nodes.Definitions[name]
	for _, rawMember := range definition.Members {
		if rawMember.Name != "Kind" && rawMember.Name != "kind" {
			continue
		}
		kinds := make([]string, 0, len(rawMember.Type))
		for _, value := range rawMember.Type {
			kinds = append(kinds, strings.TrimPrefix(value, "SyntaxKind."))
		}
		return kinds, len(kinds) > 1
	}
	if len(definition.Kind) != 0 {
		return slices.Clone(definition.Kind), false
	}
	return []string{name}, false
}

func (m *schemaModel) normalizeNodeMembers(name string) ([]member, error) {
	definition := m.raw.Nodes.Definitions[name]
	result := make([]member, 0, len(definition.Members))
	for _, rawValue := range definition.Members {
		if rawValue.Name == "Kind" || rawValue.Name == "kind" {
			continue
		}
		value := rawValue
		owner := name
		if value.Inherited {
			inherited, inheritedOwner, ok := m.findBaseField(definition.Extends, value.Name)
			if !ok {
				return nil, fmt.Errorf("%s.%s has no inherited field", name, value.Name)
			}
			if len(value.Type) == 0 {
				owner = inheritedOwner
			}
			value = resolveInheritedMember(inherited, value)
		}
		result = append(result, memberFromRaw(value, owner))
	}
	return result, nil
}

func (m *schemaModel) findBaseField(bases []string, name string) (rawMember, string, bool) {
	for _, baseName := range bases {
		base, ok := m.raw.Bases[baseName]
		if !ok {
			continue
		}
		if field, ok := base.Fields[name]; ok {
			field.Name = name
			return field, baseName, true
		}
		if field, owner, ok := m.findBaseField(base.Extends, name); ok {
			return field, owner, true
		}
	}
	return rawMember{}, "", false
}

func resolveInheritedMember(base rawMember, own rawMember) rawMember {
	result := own
	if len(result.Type) == 0 {
		result.Type = base.Type
	}
	result.List = base.List
	if result.Optional == nil {
		result.Optional = base.Optional
	}
	if boolValue(base.Private) {
		result.Private = base.Private
	}
	if result.Visit == "" {
		result.Visit = base.Visit
	}
	if result.TypeGuard == "" {
		result.TypeGuard = base.TypeGuard
	}
	return result
}

func memberFromRaw(value rawMember, owner string) member {
	return member{
		Owner:     owner,
		Name:      value.Name,
		Type:      slices.Clone(value.Type),
		Optional:  boolValue(value.Optional),
		List:      value.List,
		Private:   boolValue(value.Private),
		GoOnly:    boolValue(value.GoOnly),
		NoTS:      boolValue(value.NoTS),
		NoGo:      boolValue(value.NoGo),
		NoFactory: boolValue(value.NoFactory),
		Bitmask:   value.Bitmask,
	}
}

func (m *schemaModel) isGoOnlyBase(name string) bool {
	base := m.raw.Bases[name]
	if base.Brand != "" {
		return false
	}
	if len(base.Fields) == 0 {
		return true
	}
	for _, field := range base.Fields {
		if !boolValue(field.GoOnly) && !(boolValue(field.NoTS) && boolValue(field.NoFactory)) {
			return false
		}
	}
	return true
}

func (m *schemaModel) visibleBases(names []string) []string {
	var result []string
	seen := make(map[string]struct{})
	var add func(string)
	add = func(name string) {
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		if m.baseGoOnly[name] {
			for _, parent := range m.raw.Bases[name].Extends {
				add(parent)
			}
			return
		}
		result = append(result, name)
	}
	for _, name := range names {
		add(name)
	}
	return result
}

func (m *schemaModel) allBaseNames(names []string) []string {
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
		if !m.baseGoOnly[name] {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func (m *schemaModel) hasAnyMember(name string) bool {
	for _, value := range m.normalizedNodeMember[name] {
		if slices.Contains(value.Type, "any") {
			return true
		}
	}
	return false
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func sortedKeys[Value any](values map[string]Value) []string {
	result := make([]string, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
