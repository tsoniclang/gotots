package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"unicode"
)

type codePointRange struct {
	low  rune
	high rune
}

type category struct {
	name      string
	predicate func(rune) bool
}

type table struct {
	name  string
	value *unicode.RangeTable
}

func main() {
	output := flag.String("output", "", "generated TypeScript output")
	check := flag.Bool("check", false, "verify output without writing")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "-output is required")
		os.Exit(2)
	}

	payload := render(
		[]category{
			{name: "letter", predicate: unicode.IsLetter},
			{name: "number", predicate: unicode.IsNumber},
			{name: "print", predicate: unicode.IsPrint},
		},
		[]table{
			{name: "ideographic", value: unicode.Ideographic},
			{name: "nd", value: unicode.Nd},
			{name: "no", value: unicode.No},
			{name: "zs", value: unicode.Zs},
		},
	)
	if *check {
		current, err := os.ReadFile(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", *output, err)
			os.Exit(1)
		}
		if !bytes.Equal(current, payload) {
			fmt.Fprintf(os.Stderr, "%s is stale; run unicode:generate\n", *output)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*output, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *output, err)
		os.Exit(1)
	}
}

func render(categories []category, tables []table) []byte {
	var output bytes.Buffer
	output.WriteString("import type { RangeRecord } from \"./data.js\";\n\n")
	for _, selected := range categories {
		ranges := classify(selected.predicate)
		writeRanges(&output, selected.name+"Ranges16", ranges, 0, 0xffff)
		writeRanges(&output, selected.name+"Ranges32", ranges, 0x10000, unicode.MaxRune)
	}
	for _, selected := range tables {
		writeTable(&output, selected)
	}
	return append(bytes.TrimRight(output.Bytes(), "\n"), '\n')
}

func classify(predicate func(rune) bool) []codePointRange {
	ranges := make([]codePointRange, 0)
	start := rune(-1)
	for value := rune(0); value <= unicode.MaxRune; value++ {
		selected := predicate(value)
		if selected && start < 0 {
			start = value
		}
		if !selected && start >= 0 {
			ranges = append(ranges, codePointRange{low: start, high: value - 1})
			start = -1
		}
	}
	if start >= 0 {
		ranges = append(ranges, codePointRange{low: start, high: unicode.MaxRune})
	}
	return ranges
}

func writeRanges(
	output *bytes.Buffer,
	name string,
	ranges []codePointRange,
	minimum rune,
	maximum rune,
) {
	fmt.Fprintf(output, "export const %s: readonly RangeRecord[] = [\n", name)
	for _, selected := range ranges {
		low := max(selected.low, minimum)
		high := min(selected.high, maximum)
		if low > high {
			continue
		}
		fmt.Fprintf(output, "  [0x%06x, 0x%06x, 1],\n", low, high)
	}
	output.WriteString("];\n\n")
}

func writeTable(output *bytes.Buffer, selected table) {
	fmt.Fprintf(output, "export const %sRanges16: readonly RangeRecord[] = [\n", selected.name)
	for _, value := range selected.value.R16 {
		fmt.Fprintf(
			output,
			"  [0x%06x, 0x%06x, %d],\n",
			value.Lo,
			value.Hi,
			value.Stride,
		)
	}
	output.WriteString("];\n\n")
	fmt.Fprintf(output, "export const %sRanges32: readonly RangeRecord[] = [\n", selected.name)
	for _, value := range selected.value.R32 {
		fmt.Fprintf(
			output,
			"  [0x%06x, 0x%06x, %d],\n",
			value.Lo,
			value.Hi,
			value.Stride,
		)
	}
	output.WriteString("];\n\n")
	fmt.Fprintf(
		output,
		"export const %sLatinOffset = %d;\n\n",
		selected.name,
		selected.value.LatinOffset,
	)
}
