package config

import "encoding/json"

func (p Project) CanonicalJSON() ([]byte, error) {
	document := struct {
		SchemaVersion int `json:"schemaVersion"`
		Distribution  struct {
			Root string `json:"root"`
		} `json:"distribution"`
		Source struct {
			Root    string   `json:"root"`
			Package string   `json:"package"`
			Mode    RootMode `json:"mode"`
		} `json:"source"`
		Go struct {
			GOOS   string   `json:"goos"`
			GOARCH string   `json:"goarch"`
			CGO    bool     `json:"cgo"`
			Tags   []string `json:"tags"`
		} `json:"go"`
		Semantics struct {
			Integers        string `json:"integers"`
			EvaluationOrder string `json:"evaluationOrder"`
			Concurrency     string `json:"concurrency"`
		} `json:"semantics"`
		Providers struct {
			StandardLibrary bool `json:"standardLibrary"`
			Externals       bool `json:"externals"`
		} `json:"providers"`
		Implementations struct {
			Bundles []string `json:"bundles"`
		} `json:"implementations"`
		Output struct {
			Directory string `json:"directory"`
		} `json:"output"`
	}{SchemaVersion: SchemaVersion}
	document.Distribution.Root = p.distributionRoot
	document.Source.Root = p.sourceRoot
	document.Source.Package = p.packagePattern
	document.Source.Mode = p.rootMode
	document.Go.GOOS = p.buildProfile.GOOS()
	document.Go.GOARCH = p.buildProfile.GOARCH()
	document.Go.CGO = p.buildProfile.CgoEnabled()
	document.Go.Tags = p.buildProfile.Tags()
	document.Semantics.Integers = p.integer.String()
	document.Semantics.EvaluationOrder = p.evaluation.String()
	document.Semantics.Concurrency = p.concurrency.String()
	document.Providers.StandardLibrary = p.standardLibrary
	document.Providers.Externals = p.externals
	document.Implementations.Bundles = p.ImplementationBundles()
	document.Output.Directory = p.outputDirectory
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, projectError("encode resolved config", "", err.Error())
	}
	return append(payload, '\n'), nil
}
