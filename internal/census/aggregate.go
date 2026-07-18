package census

import "sort"

// sortRecords deduplicates dependency edges by full identity and imposes
// the deterministic report ordering.
func sortRecords(report *Report, edges []Edge, externalUse map[string]*ExternalUse) {
	edgeSeen := map[string]bool{}
	for _, e := range edges {
		key := e.From + "\x00" + e.File + "\x00" + e.To + "\x00" + e.Scope
		if !edgeSeen[key] {
			edgeSeen[key] = true
			report.Contradictions = append(report.Contradictions, e)
		}
	}
	sort.Slice(report.Contradictions, func(i, j int) bool {
		a, b := report.Contradictions[i], report.Contradictions[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Scope < b.Scope
	})

	for _, use := range externalUse {
		report.External = append(report.External, *use)
	}
	sort.Slice(report.External, func(i, j int) bool { return report.External[i].Path < report.External[j].Path })

	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Path < report.Files[j].Path })
	sort.Slice(report.Declarations, func(i, j int) bool {
		a, b := report.Declarations[i], report.Declarations[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.StartLine < b.StartLine
	})
	sort.Slice(report.Directives, func(i, j int) bool {
		a, b := report.Directives[i], report.Directives[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	sort.Slice(report.RareConstructs, func(i, j int) bool {
		a, b := report.RareConstructs[i], report.RareConstructs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	sort.Slice(report.ExternalObligations, func(i, j int) bool {
		a, b := report.ExternalObligations[i], report.ExternalObligations[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Kind < b.Kind
	})
	sort.Slice(report.TestFunctions, func(i, j int) bool {
		return report.TestFunctions[i].ID < report.TestFunctions[j].ID
	})
}

// sortShapes imposes the deterministic declaration-contract ordering.
func sortShapes(shapes *DeclarationShapes) {
	sort.Slice(shapes.Functions, func(i, j int) bool { return shapes.Functions[i].ID < shapes.Functions[j].ID })
	sort.Slice(shapes.Types, func(i, j int) bool { return shapes.Types[i].ID < shapes.Types[j].ID })
	sort.Slice(shapes.Constants, func(i, j int) bool { return shapes.Constants[i].ID < shapes.Constants[j].ID })
	sort.Slice(shapes.Variables, func(i, j int) bool { return shapes.Variables[i].ID < shapes.Variables[j].ID })
	sort.Slice(shapes.Aliases, func(i, j int) bool { return shapes.Aliases[i].ID < shapes.Aliases[j].ID })
	sort.Slice(shapes.FunctionLiterals, func(i, j int) bool { return shapes.FunctionLiterals[i].ID < shapes.FunctionLiterals[j].ID })
}

// deriveDeclarationAggregates computes every DeclCounts/Bodies/Statements
// total from the declaration records so aggregates cannot drift from the
// identity-bearing data.
func deriveDeclarationAggregates(report *Report) {
	scopeOf := func(name string) *ScopeReport {
		if name == "test" {
			return report.Test
		}
		return report.Production
	}
	for i := range report.Declarations {
		d := &report.Declarations[i]
		scope := scopeOf(d.Scope)
		switch d.Kind {
		case "func":
			scope.Declarations.Functions++
		case "method":
			scope.Declarations.Methods++
		case "type":
			scope.Declarations.NamedTypes++
		case "alias":
			scope.Declarations.Aliases++
		case "const":
			scope.Declarations.Constants++
		case "var":
			scope.Declarations.Variables++
		}
		if d.Kind == "func" || d.Kind == "method" {
			if d.HasBody {
				scope.Bodies++
				scope.Statements += d.Statements
			} else {
				scope.Declarations.BodylessFunctions++
			}
		}
	}
	fileScopes := make(map[string]string, len(report.Files))
	for _, f := range report.Files {
		fileScopes[f.Path] = f.Scope
	}
	for _, directive := range report.Directives {
		scope := fileScopes[directive.File]
		if scope == "" {
			scope = "production"
		}
		scopeOf(scope).Directives[directive.Directive]++
	}
}
