package main

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
)

func renderFactories(model *schemaModel) ([]byte, error) {
	var buffer bytes.Buffer
	generatedHeader(&buffer)

	buffer.WriteString("func (Factory) Token(kind TokenSyntaxKind) Token {\n")
	buffer.WriteString("\treturn &tokenNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("func (Factory) KeywordTypeNode(kind KeywordTypeSyntaxKind) KeywordTypeNode {\n")
	buffer.WriteString("\treturn &keywordTypeNodeNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}\n")
	buffer.WriteString("}\n\n")

	var broad []string
	for alias := range model.instantiationOwner {
		if _, exists := model.kindAliases[model.instantiationKind[alias]]; exists {
			broad = append(broad, alias)
		}
	}
	sort.Strings(broad)
	for _, alias := range broad {
		kindType := model.instantiationKind[alias]
		fmt.Fprintf(&buffer, "func (Factory) %s(kind %s) %s {\n", alias, kindType, alias)
		fmt.Fprintf(
			&buffer,
			"\treturn &%sNode{nodeCore: newNodeCore(SyntaxKind(kind), NodeFlagsNone)}\n",
			privateName(alias),
		)
		buffer.WriteString("}\n\n")
	}

	for _, node := range model.nodes {
		if !node.Constructible || node.UnsupportedAny {
			continue
		}
		if err := writeFactory(&buffer, model, node); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func writeFactory(buffer *bytes.Buffer, model *schemaModel, node concreteNode) error {
	members := model.factoryMembers(node)
	fmt.Fprintf(buffer, "func (Factory) %s(", node.Name)
	for index, value := range members {
		if index != 0 {
			buffer.WriteString(", ")
		}
		goType, err := model.goMemberType(node.SchemaName, value)
		if err != nil {
			return err
		}
		fmt.Fprintf(buffer, "%s %s", parameterName(value.Name), goType)
	}
	if node.Name == "SourceFile" {
		if len(members) != 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString("sourceData SourceFileData")
	}
	fmt.Fprintf(buffer, ") %s {\n", node.Name)

	flags := "NodeFlagsNone"
	for _, value := range members {
		if value.Name == "Flags" && slices.Equal(value.Type, []string{"NodeFlags"}) {
			flags = parameterName(value.Name)
			break
		}
	}
	private := privateName(node.Name) + "Node"
	fmt.Fprintf(buffer, "\treturn &%s{\n", private)
	fmt.Fprintf(buffer, "\t\tnodeCore: newNodeCore(SyntaxKind%s, %s),\n", node.Kind, flags)
	for _, value := range members {
		if value.Name == "Flags" {
			continue
		}
		field := parameterName(value.Name)
		if value.List != "" {
			fmt.Fprintf(buffer, "\t\t%s: cloneSlice(%s),\n", field, field)
		} else {
			fmt.Fprintf(buffer, "\t\t%s: %s,\n", field, field)
		}
	}
	if node.Name == "SourceFile" {
		buffer.WriteString("\t\tsourceData: cloneSourceFileData(sourceData),\n")
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n\n")
	return nil
}
