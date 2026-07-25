package main

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func renderNodes(model *schemaModel) ([]byte, error) {
	var buffer bytes.Buffer
	generatedHeader(&buffer)

	writeSourceFileData(&buffer)
	writeInlineKinds(&buffer, model)
	if err := writeBaseInterfaces(&buffer, model); err != nil {
		return nil, err
	}
	if err := writeAliasInterfaces(&buffer, model); err != nil {
		return nil, err
	}
	if err := writeAbstractNodeInterfaces(&buffer, model); err != nil {
		return nil, err
	}
	writeDynamicGenericNodes(&buffer, model)
	for _, node := range model.nodes {
		if err := writeConcreteNode(&buffer, model, node); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func writeInlineKinds(buffer *bytes.Buffer, model *schemaModel) {
	type inlineKind struct {
		name   string
		values []string
	}
	seen := make(map[string]inlineKind)
	for _, members := range model.normalizedNodeMember {
		for _, value := range members {
			if len(value.Type) < 2 || !allSyntaxKinds(value.Type) {
				continue
			}
			name := inlineKindName(value.Owner, value.Name)
			seen[name] = inlineKind{name: name, values: value.Type}
		}
	}
	for _, name := range sortedKeys(seen) {
		item := seen[name]
		fmt.Fprintf(buffer, "type %s SyntaxKind\n\nconst (\n", item.name)
		for _, value := range item.values {
			kind := strings.TrimPrefix(value, "SyntaxKind.")
			fmt.Fprintf(
				buffer,
				"\t%s%s %s = %d\n",
				item.name,
				kind,
				item.name,
				model.syntaxKindByName[kind],
			)
		}
		buffer.WriteString(")\n\n")
	}
}

func writeDynamicGenericNodes(buffer *bytes.Buffer, model *schemaModel) {
	type dynamicNode struct {
		name    string
		generic string
		bases   []string
		aliases []string
	}
	nodes := []dynamicNode{
		{
			name:    "token",
			generic: "Token",
			bases:   model.visibleBases(model.raw.Nodes.Definitions["Token"].Extends),
		},
		{
			name:    "keywordTypeNode",
			generic: "KeywordTypeNode",
			bases:   model.visibleBases(model.raw.Nodes.Definitions["KeywordTypeNode"].Extends),
		},
	}
	for _, alias := range sortedKeys(model.instantiationOwner) {
		owner := model.instantiationOwner[alias]
		if _, broad := model.kindAliases[model.instantiationKind[alias]]; broad {
			nodes = append(nodes, dynamicNode{
				name:    privateName(alias),
				generic: owner,
				bases:   model.visibleBases(model.raw.Nodes.Definitions[owner].Extends),
				aliases: []string{alias},
			})
		}
	}
	sort.Slice(nodes, func(left, right int) bool {
		return nodes[left].name < nodes[right].name
	})
	for _, node := range nodes {
		private := node.name + "Node"
		fmt.Fprintf(buffer, "type %s struct {\n\tnodeCore\n}\n\n", private)
		for _, base := range model.allBaseNames(node.bases) {
			fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, base)
		}
		fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, node.generic)
		for _, alias := range node.aliases {
			fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, alias)
		}
		buffer.WriteString("\n")
	}
}

func writeSourceFileData(buffer *bytes.Buffer) {
	buffer.WriteString("type SourceFileData struct {\n")
	buffer.WriteString("\tText string\n")
	buffer.WriteString("\tFileName Path\n")
	buffer.WriteString("\tPath Path\n")
	buffer.WriteString("\tLanguageVariant LanguageVariant\n")
	buffer.WriteString("\tScriptKind ScriptKind\n")
	buffer.WriteString("\tIsDeclarationFile bool\n")
	buffer.WriteString("\tReferencedFiles []FileReference\n")
	buffer.WriteString("\tTypeReferenceDirectives []FileReference\n")
	buffer.WriteString("\tLibReferenceDirectives []FileReference\n")
	buffer.WriteString("\tImports []Node\n")
	buffer.WriteString("\tModuleAugmentations []Node\n")
	buffer.WriteString("\tAmbientModuleNames []string\n")
	buffer.WriteString("\tExternalModuleIndicator Node\n")
	buffer.WriteString("}\n\n")
}

func writeBaseInterfaces(buffer *bytes.Buffer, model *schemaModel) error {
	for _, name := range sortedKeys(model.raw.Bases) {
		if model.baseGoOnly[name] {
			continue
		}
		bases := model.visibleBases(model.raw.Bases[name].Extends)
		if len(bases) == 0 {
			bases = []string{"Node"}
		}
		fmt.Fprintf(buffer, "type %s interface {\n", name)
		for _, base := range bases {
			fmt.Fprintf(buffer, "\t%s\n", base)
		}
		fmt.Fprintf(buffer, "\tis%s()\n", name)
		buffer.WriteString("}\n\n")
	}
	return nil
}

func writeAliasInterfaces(buffer *bytes.Buffer, model *schemaModel) error {
	used := make(map[string]struct{})
	for _, name := range sortedKeys(model.raw.Nodes.Aliases) {
		members, baseAlias, err := model.nodeAlias(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(buffer, "type %s interface {\n", name)
		if baseAlias {
			fmt.Fprintf(buffer, "\t%s\n", members[0])
		} else {
			buffer.WriteString("\tNode\n")
			fmt.Fprintf(buffer, "\tis%s()\n", name)
		}
		buffer.WriteString("}\n\n")
		used[name] = struct{}{}
	}
	inline, err := inlineAliases(model)
	if err != nil {
		return err
	}
	for _, name := range sortedKeys(inline) {
		if _, exists := used[name]; exists {
			return fmt.Errorf("inline alias %s collides with schema alias", name)
		}
		fmt.Fprintf(buffer, "type %s interface {\n\tNode\n\tis%s()\n}\n\n", name, name)
		used[name] = struct{}{}
	}
	for _, alias := range sortedKeys(model.instantiationOwner) {
		owner := model.instantiationOwner[alias]
		if _, broad := model.kindAliases[model.instantiationKind[alias]]; !broad {
			continue
		}
		if _, exists := used[alias]; exists {
			return fmt.Errorf("generic alias %s collides with another alias", alias)
		}
		fmt.Fprintf(buffer, "type %s interface {\n\t%s\n\tis%s()\n}\n\n", alias, owner, alias)
		used[alias] = struct{}{}
	}
	for _, name := range sortedKeys(model.raw.Nodes.ListAliases) {
		element := model.raw.Nodes.ListAliases[name]
		fmt.Fprintf(buffer, "type %s []%s\n\n", name, element)
	}
	return nil
}

func writeAbstractNodeInterfaces(buffer *bytes.Buffer, model *schemaModel) error {
	names := []string{"Token", "KeywordExpression", "KeywordTypeNode"}
	for _, node := range model.nodes {
		if node.VariantOf != "" &&
			node.VariantOf != "Token" &&
			node.VariantOf != "KeywordExpression" {
			names = append(names, node.VariantOf)
		}
	}
	sort.Strings(names)
	names = slices.Compact(names)
	for _, name := range names {
		definition, exists := model.raw.Nodes.Definitions[name]
		if !exists {
			continue
		}
		fmt.Fprintf(buffer, "type %s interface {\n", name)
		bases := model.visibleBases(definition.Extends)
		if len(bases) == 0 {
			bases = []string{"Node"}
		}
		for _, base := range bases {
			fmt.Fprintf(buffer, "\t%s\n", base)
		}
		fmt.Fprintf(buffer, "\tis%s()\n", name)
		for _, value := range model.normalizedNodeMember[name] {
			if !model.memberIsVisible(value) || slices.Contains(value.Type, "any") || value.Name == "Flags" {
				continue
			}
			goType, err := model.goMemberType(name, value)
			if err != nil {
				return err
			}
			fmt.Fprintf(buffer, "\t%s() %s\n", exportedName(value.Name), goType)
		}
		buffer.WriteString("}\n\n")
	}
	return nil
}

func writeConcreteNode(buffer *bytes.Buffer, model *schemaModel, node concreteNode) error {
	fmt.Fprintf(buffer, "type %s interface {\n", node.Name)
	if node.VariantOf != "" {
		fmt.Fprintf(buffer, "\t%s\n", node.VariantOf)
	} else if len(node.Bases) == 0 {
		buffer.WriteString("\tNode\n")
	} else {
		for _, base := range node.Bases {
			fmt.Fprintf(buffer, "\t%s\n", base)
		}
	}
	fmt.Fprintf(buffer, "\tis%s()\n", node.Name)
	for _, alias := range uniqueStrings(node.UnionAliases, node.InlineAliases, node.GenericAliases) {
		fmt.Fprintf(buffer, "\tis%s()\n", alias)
	}
	for _, value := range node.Members {
		if !model.memberIsVisible(value) || slices.Contains(value.Type, "any") || value.Name == "Flags" {
			continue
		}
		goType, err := model.goMemberType(node.SchemaName, value)
		if err != nil {
			return err
		}
		fmt.Fprintf(buffer, "\t%s() %s\n", exportedName(value.Name), goType)
	}
	if node.Name == "SourceFile" {
		buffer.WriteString("\tSourceData() SourceFileData\n")
	}
	buffer.WriteString("}\n\n")

	private := privateName(node.Name) + "Node"
	fmt.Fprintf(buffer, "type %s struct {\n\tnodeCore\n", private)
	for _, value := range node.Members {
		if !model.memberIsFactory(value) || slices.Contains(value.Type, "any") || value.Name == "Flags" {
			continue
		}
		goType, err := model.goMemberType(node.SchemaName, value)
		if err != nil {
			return err
		}
		fmt.Fprintf(buffer, "\t%s %s\n", parameterName(value.Name), goType)
	}
	if node.Name == "SourceFile" {
		buffer.WriteString("\tsourceData SourceFileData\n")
	}
	buffer.WriteString("}\n\n")

	for _, base := range model.allBaseNames(model.raw.Nodes.Definitions[node.SchemaName].Extends) {
		fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, base)
	}
	if node.VariantOf != "" {
		fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, node.VariantOf)
	}
	fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, node.Name)
	for _, alias := range uniqueStrings(node.UnionAliases, node.InlineAliases, node.GenericAliases) {
		fmt.Fprintf(buffer, "func (*%s) is%s() {}\n", private, alias)
	}
	buffer.WriteString("\n")

	for _, value := range node.Members {
		if !model.memberIsVisible(value) || slices.Contains(value.Type, "any") || value.Name == "Flags" {
			continue
		}
		goType, err := model.goMemberType(node.SchemaName, value)
		if err != nil {
			return err
		}
		method := exportedName(value.Name)
		field := parameterName(value.Name)
		fmt.Fprintf(buffer, "func (n *%s) %s() %s {\n", private, method, goType)
		if value.List != "" {
			fmt.Fprintf(buffer, "\treturn cloneSlice(n.%s)\n", field)
		} else {
			fmt.Fprintf(buffer, "\treturn n.%s\n", field)
		}
		buffer.WriteString("}\n\n")
	}
	if node.Name == "SourceFile" {
		fmt.Fprintf(buffer, "func (n *%s) SourceData() SourceFileData {\n", private)
		buffer.WriteString("\treturn cloneSourceFileData(n.sourceData)\n")
		buffer.WriteString("}\n\n")
	}
	return nil
}

func inlineAliases(model *schemaModel) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, members := range model.normalizedNodeMember {
		for _, value := range members {
			if value.List != "" || len(value.Type) < 2 || !model.memberIsChild(value) {
				continue
			}
			name := inlineAliasName(value.Owner, value.Name)
			if existing, exists := result[name]; exists && !slices.Equal(existing, value.Type) {
				return nil, fmt.Errorf("inline alias %s has conflicting members", name)
			}
			result[name] = slices.Clone(value.Type)
		}
	}
	return result, nil
}

func privateName(name string) string {
	if strings.HasPrefix(name, "JSDoc") {
		return "jsDoc" + name[len("JSDoc"):]
	}
	return strings.ToLower(name[:1]) + name[1:]
}
