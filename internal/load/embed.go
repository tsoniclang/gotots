package load

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type EmbedKind uint8

const (
	EmbedInvalid EmbedKind = iota
	EmbedString
	EmbedBytes
	EmbedFileSystem
)

func (k EmbedKind) Valid() bool {
	return k == EmbedString || k == EmbedBytes || k == EmbedFileSystem
}

type EmbedFile struct {
	name    string
	content []byte
}

func (f EmbedFile) Name() string {
	return f.name
}

func (f EmbedFile) Bytes() []byte {
	return slices.Clone(f.content)
}

type EmbedValue struct {
	kind  EmbedKind
	files []EmbedFile
}

func (v EmbedValue) Kind() EmbedKind {
	return v.kind
}

func (v EmbedValue) Files() []EmbedFile {
	result := make([]EmbedFile, len(v.files))
	for index, file := range v.files {
		result[index] = EmbedFile{
			name:    file.name,
			content: slices.Clone(file.content),
		}
	}
	return result
}

func (v EmbedValue) String() (string, bool) {
	if v.kind != EmbedString || len(v.files) != 1 {
		return "", false
	}
	return string(v.files[0].content), true
}

func (p *Package) Embed(variable *types.Var) (EmbedValue, bool) {
	if p == nil || variable == nil {
		return EmbedValue{}, false
	}
	value, ok := p.embeds[variable]
	if !ok {
		return EmbedValue{}, false
	}
	return EmbedValue{
		kind:  value.kind,
		files: value.Files(),
	}, true
}

type embedDeclaration struct {
	variable *types.Var
	kind     EmbedKind
	patterns []string
}

func resolvePackageEmbeds(
	selected *packages.Package,
) (map[*types.Var]EmbedValue, error) {
	if len(selected.EmbedPatterns) == 0 && len(selected.EmbedFiles) == 0 {
		return nil, nil
	}
	if selected.Dir == "" || selected.TypesInfo == nil {
		return nil, fmt.Errorf("embed evidence lacks package directory or types")
	}
	declarations, observedPatterns, err := embedDeclarations(selected)
	if err != nil {
		return nil, err
	}
	selectedPatterns, err := selectedEmbedPatterns(selected)
	if err != nil {
		return nil, err
	}
	if err := compareStringSets(
		"embed patterns",
		observedPatterns,
		selectedPatterns,
	); err != nil {
		return nil, err
	}

	selectedFiles, err := selectedEmbedFiles(selected)
	if err != nil {
		return nil, err
	}
	patternFiles := make(map[string][]string, len(selectedPatterns))
	resolvedFiles := make(map[string]struct{}, len(selectedFiles))
	for pattern := range selectedPatterns {
		for name := range selectedFiles {
			if embedPatternMatches(pattern, name) {
				patternFiles[pattern] = append(patternFiles[pattern], name)
				resolvedFiles[name] = struct{}{}
			}
		}
		if len(patternFiles[pattern]) == 0 {
			return nil, fmt.Errorf(
				"embed pattern %q has no selected files",
				pattern,
			)
		}
		sort.Strings(patternFiles[pattern])
	}
	if err := compareStringSets(
		"embed files",
		resolvedFiles,
		stringSetMapKeys(selectedFiles),
	); err != nil {
		return nil, err
	}

	contents := make(map[string][]byte, len(selectedFiles))
	values := make(map[*types.Var]EmbedValue, len(declarations))
	for _, declaration := range declarations {
		names := make(map[string]struct{})
		for _, pattern := range declaration.patterns {
			for _, name := range patternFiles[pattern] {
				names[name] = struct{}{}
			}
		}
		if declaration.kind != EmbedFileSystem && len(names) != 1 {
			return nil, fmt.Errorf(
				"embedded variable %s has %d files, want one",
				declaration.variable.Name(),
				len(names),
			)
		}
		orderedNames := make([]string, 0, len(names))
		for name := range names {
			orderedNames = append(orderedNames, name)
		}
		sort.Strings(orderedNames)
		files := make([]EmbedFile, 0, len(orderedNames))
		for _, name := range orderedNames {
			content, ok := contents[name]
			if !ok {
				content, err = os.ReadFile(selectedFiles[name])
				if err != nil {
					return nil, fmt.Errorf(
						"read embedded file %q: %w",
						name,
						err,
					)
				}
				contents[name] = content
			}
			files = append(files, EmbedFile{
				name:    name,
				content: slices.Clone(content),
			})
		}
		values[declaration.variable] = EmbedValue{
			kind:  declaration.kind,
			files: files,
		}
	}
	return values, nil
}

func embedDeclarations(
	selected *packages.Package,
) ([]embedDeclaration, map[string]struct{}, error) {
	var declarations []embedDeclaration
	patterns := make(map[string]struct{})
	for _, file := range selected.Syntax {
		for _, sourceDeclaration := range file.Decls {
			declaration, ok := sourceDeclaration.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				continue
			}
			for _, sourceSpec := range declaration.Specs {
				spec, ok := sourceSpec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				doc := spec.Doc
				if doc == nil && len(declaration.Specs) == 1 {
					doc = declaration.Doc
				}
				embedPatterns, err := parseEmbedPatterns(doc)
				if err != nil {
					return nil, nil, err
				}
				if len(embedPatterns) == 0 {
					continue
				}
				if len(spec.Names) != 1 || len(spec.Values) != 0 {
					return nil, nil, fmt.Errorf(
						"go:embed must initialize one variable without a source value",
					)
				}
				variable, ok := selected.TypesInfo.Defs[spec.Names[0]].(*types.Var)
				if !ok {
					return nil, nil, fmt.Errorf(
						"go:embed variable %q has no go/types identity",
						spec.Names[0].Name,
					)
				}
				kind := embedKind(variable.Type())
				if !kind.Valid() {
					return nil, nil, fmt.Errorf(
						"go:embed variable %q has unsupported type %s",
						variable.Name(),
						variable.Type(),
					)
				}
				for _, pattern := range embedPatterns {
					patterns[pattern] = struct{}{}
				}
				declarations = append(declarations, embedDeclaration{
					variable: variable,
					kind:     kind,
					patterns: embedPatterns,
				})
			}
		}
	}
	return declarations, patterns, nil
}

func parseEmbedPatterns(doc *ast.CommentGroup) ([]string, error) {
	if doc == nil {
		return nil, nil
	}
	var patterns []string
	for _, comment := range doc.List {
		directive, ok := ast.ParseDirective(comment.Slash, comment.Text)
		if !ok || directive.Tool != "go" || directive.Name != "embed" {
			continue
		}
		arguments, err := directive.ParseArgs()
		if err != nil {
			return nil, fmt.Errorf("parse go:embed directive: %w", err)
		}
		if len(arguments) == 0 {
			return nil, fmt.Errorf("go:embed directive has no patterns")
		}
		for _, argument := range arguments {
			patterns = append(patterns, argument.Arg)
		}
	}
	return patterns, nil
}

func embedKind(source types.Type) EmbedKind {
	unalias := types.Unalias(source)
	if basic, ok := unalias.Underlying().(*types.Basic); ok &&
		basic.Info()&types.IsString != 0 {
		return EmbedString
	}
	if slice, ok := unalias.Underlying().(*types.Slice); ok {
		if basic, ok := slice.Elem().Underlying().(*types.Basic); ok &&
			basic.Kind() == types.Uint8 {
			return EmbedBytes
		}
	}
	named, ok := unalias.(*types.Named)
	if ok && named.Obj().Pkg() != nil &&
		named.Obj().Pkg().Path() == "embed" && named.Obj().Name() == "FS" {
		return EmbedFileSystem
	}
	return EmbedInvalid
}

func selectedEmbedFiles(
	selected *packages.Package,
) (map[string]string, error) {
	files := make(map[string]string, len(selected.EmbedFiles))
	for _, absolute := range selected.EmbedFiles {
		relative, err := filepath.Rel(selected.Dir, absolute)
		if err != nil {
			return nil, fmt.Errorf("relativize embedded file %q: %w", absolute, err)
		}
		name := filepath.ToSlash(relative)
		if name == "." || name == ".." || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf(
				"embedded file %q is outside package directory",
				absolute,
			)
		}
		if _, duplicate := files[name]; duplicate {
			return nil, fmt.Errorf("duplicate embedded file identity %q", name)
		}
		files[name] = absolute
	}
	return files, nil
}

func selectedEmbedPatterns(
	selected *packages.Package,
) (map[string]struct{}, error) {
	patterns := make(map[string]struct{}, len(selected.EmbedPatterns))
	for _, absolute := range selected.EmbedPatterns {
		pattern, all := strings.CutPrefix(absolute, "all:")
		relative, err := filepath.Rel(selected.Dir, pattern)
		if err != nil {
			return nil, fmt.Errorf(
				"relativize embed pattern %q: %w",
				absolute,
				err,
			)
		}
		pattern = filepath.ToSlash(relative)
		if pattern == "." || pattern == ".." ||
			strings.HasPrefix(pattern, "../") {
			return nil, fmt.Errorf(
				"embed pattern %q is outside package directory",
				absolute,
			)
		}
		if all {
			pattern = "all:" + pattern
		}
		patterns[pattern] = struct{}{}
	}
	return patterns, nil
}

func embedPatternMatches(pattern string, name string) bool {
	glob, all := strings.CutPrefix(pattern, "all:")
	if !all && !embedDescendantIsVisible(name) {
		return false
	}
	if matched, err := pathpkg.Match(glob, name); err == nil && matched {
		return true
	}
	for directory := pathpkg.Dir(name); directory != "."; directory = pathpkg.Dir(directory) {
		matched, err := pathpkg.Match(glob, directory)
		if err != nil || !matched {
			continue
		}
		return true
	}
	return false
}

func embedDescendantIsVisible(name string) bool {
	for _, element := range strings.Split(name, "/") {
		if strings.HasPrefix(element, ".") || strings.HasPrefix(element, "_") {
			return false
		}
	}
	return true
}

func stringSetMapKeys(values map[string]string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for value := range values {
		result[value] = struct{}{}
	}
	return result
}

func compareStringSets(
	name string,
	actual map[string]struct{},
	expected map[string]struct{},
) error {
	var missing []string
	var extra []string
	for value := range expected {
		if _, ok := actual[value]; !ok {
			missing = append(missing, value)
		}
	}
	for value := range actual {
		if _, ok := expected[value]; !ok {
			extra = append(extra, value)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return nil
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return fmt.Errorf(
		"%s differ: missing=%v extra=%v",
		name,
		missing,
		extra,
	)
}
