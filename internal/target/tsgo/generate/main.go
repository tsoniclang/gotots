package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	schemaDirectory := flag.String("schema", "", "pinned TS-Go schema directory")
	outputDirectory := flag.String("output", "", "generated Go output directory")
	flag.Parse()
	if *schemaDirectory == "" || *outputDirectory == "" {
		fmt.Fprintln(os.Stderr, "-schema and -output are required")
		os.Exit(2)
	}
	model, err := loadModel(filepath.Clean(*schemaDirectory))
	if err != nil {
		fail(err)
	}
	outputs, err := renderAll(model)
	if err != nil {
		fail(err)
	}
	for name, data := range outputs {
		if err := os.WriteFile(filepath.Join(*outputDirectory, name), data, 0o644); err != nil {
			fail(err)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
