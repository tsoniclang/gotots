package main

import (
	"bytes"
	"fmt"
	"math/bits"
	"slices"
	"strings"
)

func renderEncoding(model *schemaModel) ([]byte, error) {
	var buffer bytes.Buffer
	generatedHeader(&buffer)
	buffer.WriteString("import \"strings\"\n\n")

	dynamic := []string{"keywordTypeNodeNode", "tokenNode"}
	for alias := range model.instantiationOwner {
		if _, broad := model.kindAliases[model.instantiationKind[alias]]; broad {
			dynamic = append(dynamic, privateName(alias)+"Node")
		}
	}
	slices.Sort(dynamic)
	for _, private := range dynamic {
		fmt.Fprintf(
			&buffer,
			"func (*%s) targetEncoding() nodeEncoding {\n\treturn nodeEncoding{dataType: nodeDataChildren}\n}\n\n",
			private,
		)
	}
	for _, node := range model.nodes {
		if err := writeNodeEncoding(&buffer, model, node); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func writeNodeEncoding(buffer *bytes.Buffer, model *schemaModel, node concreteNode) error {
	private := privateName(node.Name) + "Node"
	dataType := model.nodeDataType(node)
	fmt.Fprintf(buffer, "func (n *%s) targetEncoding() nodeEncoding {\n", private)

	commonExpression, prelude, err := commonDataExpression(model, node)
	if err != nil {
		return err
	}
	buffer.WriteString(prelude)
	children := model.childMembers(node)
	if len(children) > 8 {
		return fmt.Errorf("%s has %d encoded children, protocol supports 8", node.Name, len(children))
	}
	dynamicJSDocOrder := node.SchemaName == "JSDocParameterOrPropertyTag"
	if dynamicJSDocOrder {
		buffer.WriteString("\tchildren := []childEncoding{\n")
		writeChildEntries(buffer, node, children, "\t\t")
		buffer.WriteString("\t}\n")
		buffer.WriteString("\tif !n.isNameFirst {\n")
		buffer.WriteString("\t\tchildren[1], children[2] = children[2], children[1]\n")
		buffer.WriteString("\t}\n")
	}
	buffer.WriteString("\treturn nodeEncoding{\n")
	fmt.Fprintf(buffer, "\t\tdataType: %s,\n", dataType)
	if commonExpression != "" && commonExpression != "0" {
		fmt.Fprintf(buffer, "\t\tcommonData: %s,\n", commonExpression)
	}
	if len(children) != 0 {
		if dynamicJSDocOrder {
			buffer.WriteString("\t\tchildren: children,\n")
		} else {
			buffer.WriteString("\t\tchildren: []childEncoding{\n")
			writeChildEntries(buffer, node, children, "\t\t\t")
			buffer.WriteString("\t\t},\n")
		}
	}
	switch dataType {
	case "nodeDataString":
		text, err := textExpression(node)
		if err != nil {
			return err
		}
		fmt.Fprintf(buffer, "\t\ttext: %s,\n", text)
	case "nodeDataExtended":
		if node.Name == "SourceFile" {
			buffer.WriteString("\t\textended: extendedSourceFile,\n")
			buffer.WriteString("\t\tsourceFile: cloneSourceFileData(n.sourceData),\n")
		} else if node.UnsupportedAny {
			buffer.WriteString("\t\textended: extendedUnsupported,\n")
		} else {
			text, err := textExpression(node)
			if err != nil {
				return err
			}
			fmt.Fprintf(buffer, "\t\ttext: %s,\n", text)
			if rawText := memberNamed(node.Members, "RawText"); rawText != nil {
				buffer.WriteString("\t\textended: extendedTemplate,\n")
				fmt.Fprintf(buffer, "\t\trawText: n.%s,\n", parameterName(rawText.Name))
				flags := memberNamed(node.Members, "TemplateFlags")
				fmt.Fprintf(buffer, "\t\ttokenFlags: n.%s,\n", parameterName(flags.Name))
			} else {
				buffer.WriteString("\t\textended: extendedLiteral,\n")
				flags := memberNamed(node.Members, "TokenFlags")
				if flags == nil {
					flags = memberNamed(node.Members, "TemplateFlags")
				}
				if flags == nil {
					return fmt.Errorf("%s extended literal has no flags", node.Name)
				}
				fmt.Fprintf(buffer, "\t\ttokenFlags: n.%s,\n", parameterName(flags.Name))
			}
		}
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n\n")
	return nil
}

func writeChildEntries(
	buffer *bytes.Buffer,
	node concreteNode,
	children []member,
	indent string,
) {
	for _, value := range children {
		field := "n." + parameterName(value.Name)
		required := childRequired(node, value)
		if value.List == "" {
			fmt.Fprintf(
				buffer,
				"%s{name: %q, present: %s != nil, required: %t, node: %s},\n",
				indent,
				value.Name,
				field,
				required,
				field,
			)
			continue
		}
		present := "true"
		if value.Optional || value.List == "ModifierList" {
			present = field + " != nil"
		}
		if value.List == "raw" {
			present = "len(" + field + ") != 0"
		}
		fmt.Fprintf(
			buffer,
			"%s{name: %q, present: %s, required: %t, raw: %t, nodes: nodesOf(%s)},\n",
			indent,
			value.Name,
			present,
			required && value.List != "raw",
			value.List == "raw",
			field,
		)
	}
}

func childRequired(node concreteNode, value member) bool {
	if node.Kind == "DefaultClause" && value.Name == "Expression" {
		return false
	}
	return !value.Optional
}

func (m *schemaModel) nodeDataType(node concreteNode) string {
	if node.Name == "SourceFile" || node.UnsupportedAny {
		return "nodeDataExtended"
	}
	var stringMembers []member
	handWrittenCommon := false
	for _, value := range m.encodedMembers(node) {
		if m.memberIsChild(value) {
			continue
		}
		if memberIsString(value) {
			stringMembers = append(stringMembers, value)
			continue
		}
		if value.Name == "Flags" && slices.Equal(value.Type, []string{"NodeFlags"}) {
			continue
		}
		if memberIsBool(value) || len(m.memberKindValues(value)) != 0 {
			continue
		}
		handWrittenCommon = true
	}
	if len(stringMembers) > 1 || (len(stringMembers) == 1 && handWrittenCommon) {
		return "nodeDataExtended"
	}
	if len(stringMembers) == 1 {
		return "nodeDataString"
	}
	return "nodeDataChildren"
}

func (m *schemaModel) childMembers(node concreteNode) []member {
	var result []member
	for _, value := range m.encodedMembers(node) {
		if m.memberIsChild(value) {
			result = append(result, value)
		}
	}
	return result
}

func (m *schemaModel) memberKindValues(value member) []string {
	if len(value.Type) > 1 && allSyntaxKinds(value.Type) {
		result := make([]string, len(value.Type))
		for index, name := range value.Type {
			result[index] = strings.TrimPrefix(name, "SyntaxKind.")
		}
		return result
	}
	if len(value.Type) == 1 {
		if members, exists := m.kindAliases[value.Type[0]]; exists {
			return m.expandKindMembers(members)
		}
	}
	return nil
}

func commonDataExpression(model *schemaModel, node concreteNode) (string, string, error) {
	if model.nodeDataType(node) == "nodeDataExtended" {
		return "0", "", nil
	}
	var expressions []string
	var prelude strings.Builder
	bitPosition := 24
	for _, value := range model.encodedMembers(node) {
		if model.memberIsChild(value) || memberIsString(value) ||
			(value.Name == "Flags" && slices.Equal(value.Type, []string{"NodeFlags"})) {
			continue
		}
		field := "n." + parameterName(value.Name)
		if memberIsBool(value) {
			expressions = append(expressions, fmt.Sprintf("boolBit(%s) << %d", field, bitPosition))
			bitPosition++
			continue
		}
		kinds := model.memberKindValues(value)
		if len(kinds) == 0 {
			if node.UnsupportedAny {
				continue
			}
			return "", "", fmt.Errorf("%s.%s has no common-data encoding", node.Name, value.Name)
		}
		width := bits.Len(uint(len(kinds) - 1))
		if value.Optional {
			width = bits.Len(uint(len(kinds)))
		}
		variable := parameterName(value.Name) + "Index"
		fmt.Fprintf(&prelude, "\tvar %s uint32\n", variable)
		fmt.Fprintf(&prelude, "\tswitch SyntaxKind(%s) {\n", field)
		start := 1
		if !value.Optional {
			start = 0
		}
		for index, kind := range kinds {
			encoded := index + start
			if encoded == 0 {
				continue
			}
			fmt.Fprintf(
				&prelude,
				"\tcase SyntaxKind%s:\n\t\t%s = %d\n",
				kind,
				variable,
				encoded,
			)
		}
		prelude.WriteString("\t}\n")
		expressions = append(expressions, fmt.Sprintf("%s << %d", variable, bitPosition))
		bitPosition += width
	}
	if bitPosition > 30 {
		return "", "", fmt.Errorf("%s common data exceeds six bits", node.Name)
	}
	if len(expressions) == 0 {
		return "0", prelude.String(), nil
	}
	return strings.Join(expressions, " | "), prelude.String(), nil
}

func textExpression(node concreteNode) (string, error) {
	for _, value := range node.Members {
		if !memberIsString(value) {
			continue
		}
		field := "n." + parameterName(value.Name)
		if value.List == "raw" {
			return "strings.Join(" + field + ", \"\")", nil
		}
		return field, nil
	}
	return "", fmt.Errorf("%s has no text member", node.Name)
}

func memberIsString(value member) bool {
	return len(value.Type) == 1 && value.Type[0] == "string"
}

func memberIsBool(value member) bool {
	return len(value.Type) == 1 && (value.Type[0] == "bool" || value.Type[0] == "boolean")
}

func memberNamed(members []member, name string) *member {
	for index := range members {
		if members[index].Name == name {
			return &members[index]
		}
	}
	return nil
}
