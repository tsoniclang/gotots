package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func renderContractEvidence(model *schemaModel) ([]byte, error) {
	var buffer bytes.Buffer
	generatedHeader(&buffer)
	buffer.WriteString("var generatedSchemaNodeNames = [...]string{\n")
	for _, name := range sortedKeys(model.raw.Nodes.Definitions) {
		fmt.Fprintf(&buffer, "\t%q,\n", name)
	}
	buffer.WriteString("}\n\n")

	properties, err := schemaChildProperties(model)
	if err != nil {
		return nil, err
	}
	buffer.WriteString("var generatedChildProperties = map[string][]string{\n")
	kinds := sortedKeys(properties)
	sort.Slice(kinds, func(left, right int) bool {
		return model.syntaxKindByName[kinds[left]] < model.syntaxKindByName[kinds[right]]
	})
	for _, kind := range kinds {
		fmt.Fprintf(&buffer, "\t%q: {", kind)
		for index, property := range properties[kind] {
			if index != 0 {
				buffer.WriteString(", ")
			}
			fmt.Fprintf(&buffer, "%q", property)
		}
		buffer.WriteString("},\n")
	}
	buffer.WriteString("}\n")
	return buffer.Bytes(), nil
}

func schemaChildProperties(model *schemaModel) (map[string][]string, error) {
	result := make(map[string][]string)
	for name := range model.raw.Nodes.Definitions {
		members := model.normalizedNodeMember[name]
		var properties []string
		for _, value := range members {
			if model.memberIsEncoded(value) && model.memberIsChild(value) {
				properties = append(properties, targetPropertyName(value.Name))
			}
		}
		if len(properties) == 0 {
			continue
		}
		kinds, _ := model.nodeKinds(name)
		for _, kind := range kinds {
			result[kind] = properties
		}
	}
	return result, nil
}

func targetPropertyName(name string) string {
	if strings.HasPrefix(name, "JSDoc") {
		return "jsdoc" + name[len("JSDoc"):]
	}
	return strings.ToLower(name[:1]) + name[1:]
}
