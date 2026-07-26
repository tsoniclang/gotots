package main

import (
	"bytes"
	"fmt"
	"strings"
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
	if err := writeKeywordIdentifierPredicate(&buffer, model); err != nil {
		return nil, err
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

func writeKeywordIdentifierPredicate(buffer *bytes.Buffer, model *schemaModel) error {
	first, firstExists := model.syntaxKindByName["FirstKeyword"]
	last, lastExists := model.syntaxKindByName["LastKeyword"]
	if !firstExists || !lastExists || first > last {
		return fmt.Errorf("pinned SyntaxKind keyword range is invalid")
	}
	buffer.WriteString("func RequiresBindingIdentifierEscape(text string) bool {\n")
	buffer.WriteString("\tswitch text {\n")
	buffer.WriteString("\tcase\n")
	for _, value := range model.syntaxKinds {
		if value.Alias || value.Value < first || value.Value > last {
			continue
		}
		stem, found := strings.CutSuffix(value.Name, "Keyword")
		if !found || stem == "" {
			return fmt.Errorf("keyword syntax kind %s has no Keyword suffix", value.Name)
		}
		fmt.Fprintf(buffer, "\t\t%q,\n", strings.ToLower(stem))
	}
	buffer.WriteString("\t\t\"arguments\",\n")
	buffer.WriteString("\t\t\"eval\":\n")
	buffer.WriteString("\t\treturn true\n")
	buffer.WriteString("\tdefault:\n")
	buffer.WriteString("\t\treturn false\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n")
	return nil
}
