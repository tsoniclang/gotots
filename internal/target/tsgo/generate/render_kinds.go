package main

import (
	"bytes"
	"fmt"
)

func renderKinds(model *schemaModel) ([]byte, error) {
	var buffer bytes.Buffer
	generatedHeader(&buffer)
	writeEnumConstants(&buffer, "SyntaxKind", model.syntaxKinds)

	enumFiles := []struct {
		typeName string
		fileName string
	}{
		{"NodeFlags", "nodeFlags.enum.ts"},
		{"TokenFlags", "tokenFlags.enum.ts"},
		{"LanguageVariant", "languageVariant.enum.ts"},
		{"ScriptKind", "scriptKind.enum.ts"},
	}
	for _, item := range enumFiles {
		values, err := parseEnumFile(schemaPath(model, item.fileName), item.typeName)
		if err != nil {
			return nil, err
		}
		writeEnumConstants(&buffer, item.typeName, values)
	}

	for _, name := range sortedKeys(model.kindAliases) {
		fmt.Fprintf(&buffer, "type %s SyntaxKind\n\n", name)
		buffer.WriteString("const (\n")
		for _, kind := range model.expandKindMembers(model.kindAliases[name]) {
			fmt.Fprintf(
				&buffer,
				"\t%s%s %s = %d\n",
				name,
				kind,
				name,
				model.syntaxKindByName[kind],
			)
		}
		buffer.WriteString(")\n\n")
	}
	return buffer.Bytes(), nil
}

func writeEnumConstants(buffer *bytes.Buffer, typeName string, values []enumValue) {
	buffer.WriteString("const (\n")
	for _, value := range values {
		fmt.Fprintf(
			buffer,
			"\t%s%s %s = %d\n",
			typeName,
			value.Name,
			typeName,
			value.Value,
		)
	}
	buffer.WriteString(")\n\n")
}
