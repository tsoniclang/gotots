package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var protocolConstantPattern = regexp.MustCompile(`^export const ([A-Z][A-Z0-9_]*) = (0x[0-9A-Fa-f]+|[0-9]+);$`)

func renderProtocol(model *schemaModel) ([]byte, error) {
	file, err := os.Open(schemaPath(model, "protocol.ts"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buffer bytes.Buffer
	generatedHeader(&buffer)
	fmt.Fprintf(&buffer, "const pinnedToolModule = %q\n", model.manifest.Module)
	fmt.Fprintf(&buffer, "const pinnedToolPackage = %q\n", model.manifest.ToolPackage)
	fmt.Fprintf(&buffer, "const pinnedToolVersion = %q\n", model.manifest.ToolVersion)
	fmt.Fprintf(&buffer, "const pinnedToolSum = %q\n", model.manifest.ToolSum)
	fmt.Fprintf(&buffer, "const pinnedSchemaRevision = %q\n", model.manifest.Revision)
	fmt.Fprintf(
		&buffer,
		"const pinnedSchemaContractDigest = %q\n\n",
		model.manifest.contractDigest,
	)
	buffer.WriteString("const (\n")
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		match := protocolConstantPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		value, err := strconv.ParseUint(match[2], 0, 32)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buffer, "\t%s = %d\n", lowerProtocolName(match[1]), value)
		count++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("protocol.ts contains no constants")
	}
	buffer.WriteString(")\n")
	return buffer.Bytes(), nil
}

func lowerProtocolName(name string) string {
	parts := strings.Split(strings.ToLower(name), "_")
	for index := 1; index < len(parts); index++ {
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, "")
}
