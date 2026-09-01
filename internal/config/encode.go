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
		} `json:"semantics"`
		Providers struct {
			StandardLibrary bool `json:"standardLibrary"`
			Externals       bool `json:"externals"`
		} `json:"providers"`
		Implementations struct {
			CertificationSources []string `json:"certificationSources"`
			Packages             []string `json:"packages"`
			Callables            []string `json:"callables"`
		} `json:"implementations"`
		Output struct {
			Directory string `json:"directory"`
		} `json:"output"`
		Tools struct {
			Go    string `json:"go"`
			TSGo  string `json:"tsgo"`
			Cache string `json:"cache"`
		} `json:"tools"`
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
	document.Providers.StandardLibrary = p.standardLibrary
	document.Providers.Externals = p.externals
	document.Implementations.Packages = p.PackageImplementations()
	document.Implementations.Callables = p.CallableImplementations()
	document.Implementations.CertificationSources = p.ImplementationCertificationSources()
	document.Output.Directory = p.outputDirectory
	document.Tools.Go = p.goTool.Path()
	document.Tools.TSGo = p.tsgoTool.Path()
	document.Tools.Cache = p.toolCacheRoot
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, projectError("encode resolved config", "", err.Error())
	}
	return append(payload, '\n'), nil
}
