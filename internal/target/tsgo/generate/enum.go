package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var enumEntryPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+),\s*$`)

func parseEnumFile(path string, enumName string) ([]enumValue, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	start := "export enum " + enumName + " {"
	scanner := bufio.NewScanner(file)
	inside := false
	values := make(map[string]uint32)
	var result []enumValue
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if !inside {
			if strings.TrimSpace(line) == start {
				inside = true
			}
			continue
		}
		if strings.TrimSpace(line) == "}" {
			break
		}
		match := enumEntryPattern.FindStringSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("%s: unsupported %s enum line %q", path, enumName, line)
		}
		value, alias, err := evaluateEnumExpression(match[2], values)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, match[1], err)
		}
		values[match[1]] = value
		result = append(result, enumValue{Name: match[1], Value: value, Alias: alias})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !inside || len(result) == 0 {
		return nil, fmt.Errorf("%s: enum %s not found", path, enumName)
	}
	return result, nil
}

func evaluateEnumExpression(expression string, values map[string]uint32) (uint32, bool, error) {
	parts := strings.Split(expression, "|")
	var result uint32
	alias := false
	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if shift := strings.Split(part, "<<"); len(shift) == 2 {
			left, err := strconv.ParseUint(strings.TrimSpace(shift[0]), 0, 32)
			if err != nil {
				return 0, false, err
			}
			right, err := strconv.ParseUint(strings.TrimSpace(shift[1]), 0, 8)
			if err != nil {
				return 0, false, err
			}
			result |= uint32(left << right)
			continue
		}
		if number, err := strconv.ParseUint(part, 0, 32); err == nil {
			result |= uint32(number)
			continue
		}
		value, exists := values[part]
		if !exists {
			return 0, false, fmt.Errorf("unknown enum reference %q", part)
		}
		alias = true
		result |= value
	}
	return result, alias, nil
}
