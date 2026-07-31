package tsgo

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type ProjectInspectionError struct {
	Operation string
	Path      string
	Reason    string
}

func (e *ProjectInspectionError) Error() string {
	return fmt.Sprintf(
		"inspect TS-Go project %s %q: %s",
		e.Operation,
		e.Path,
		e.Reason,
	)
}

type ProjectInspection struct {
	client   *Client
	snapshot uint64
	project  string
	config   string
}

type ProjectExport struct {
	name         string
	flags        uint32
	typeString   string
	declarations []string
}

func (e ProjectExport) Name() string {
	return e.name
}

func (e ProjectExport) Flags() uint32 {
	return e.flags
}

func (e ProjectExport) TypeString() string {
	return e.typeString
}

func (e ProjectExport) Declarations() []string {
	return slices.Clone(e.declarations)
}

func (c *Client) OpenProject(configFile string) (*ProjectInspection, error) {
	absolute, err := filepath.Abs(configFile)
	if err != nil {
		return nil, projectInspectionError("open", configFile, err)
	}
	var response updateSnapshotResponse
	if err := requestProjectJSON(
		c,
		"updateSnapshot",
		updateSnapshotParams{OpenProject: filepath.ToSlash(absolute)},
		&response,
	); err != nil {
		return nil, err
	}
	if response.Snapshot == 0 {
		return nil, &ProjectInspectionError{
			Operation: "open",
			Path:      absolute,
			Reason:    "snapshot handle is absent",
		}
	}
	var selected *projectResponse
	for index := range response.Projects {
		project := &response.Projects[index]
		if sameProjectPath(project.ConfigFileName, absolute) {
			selected = project
			break
		}
	}
	if selected == nil || selected.ID == "" {
		return nil, &ProjectInspectionError{
			Operation: "open",
			Path:      absolute,
			Reason:    "configured project is absent from the snapshot",
		}
	}
	return &ProjectInspection{
		client:   c,
		snapshot: response.Snapshot,
		project:  selected.ID,
		config:   filepath.ToSlash(absolute),
	}, nil
}

func (p *ProjectInspection) Exports(sourceFile string) ([]ProjectExport, error) {
	if p == nil || p.client == nil || p.snapshot == 0 || p.project == "" {
		return nil, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourceFile,
			Reason:    "project inspection is invalid",
		}
	}
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("exports", sourceFile, err)
	}
	sourcePath := filepath.ToSlash(absolute)
	var encoded *sourceFileResponse
	if err := requestProjectJSON(
		p.client,
		"getSourceFile",
		getSourceFileParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			File:     sourcePath,
		},
		&encoded,
	); err != nil {
		return nil, err
	}
	if encoded == nil || encoded.Data == "" {
		return nil, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "source file is absent from the project",
		}
	}
	var module *symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getSymbolAtLocation",
		getSymbolAtLocationParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Location: "1.0." + sourcePath,
		},
		&module,
	); err != nil {
		return nil, err
	}
	if module == nil || module.ID == 0 {
		return nil, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "source-file module symbol is absent",
		}
	}
	var symbols []symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getExportsOfSymbol",
		getExportsOfSymbolParams{
			Snapshot: p.snapshot,
			Symbol:   module.ID,
		},
		&symbols,
	); err != nil {
		return nil, err
	}
	result := make([]ProjectExport, 0, len(symbols))
	for _, symbol := range symbols {
		selected, err := p.projectExport(sourcePath, symbol)
		if err != nil {
			return nil, err
		}
		result = append(result, selected)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].name < result[right].name
	})
	for index := 1; index < len(result); index++ {
		if result[index-1].name == result[index].name {
			return nil, &ProjectInspectionError{
				Operation: "exports",
				Path:      sourcePath,
				Reason:    "duplicate export " + result[index].name,
			}
		}
	}
	return result, nil
}

func (p *ProjectInspection) projectExport(
	sourcePath string,
	symbol symbolResponse,
) (ProjectExport, error) {
	if symbol.ID == 0 || symbol.Name == "" {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export symbol is invalid",
		}
	}
	var targetType *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypeOfSymbol",
		getTypeOfSymbolParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Symbol:   symbol.ID,
		},
		&targetType,
	); err != nil {
		return ProjectExport{}, err
	}
	if targetType == nil || targetType.ID == 0 {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export " + symbol.Name + " has no type",
		}
	}
	var typeString string
	if err := requestProjectJSON(
		p.client,
		"typeToString",
		typeToStringParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     targetType.ID,
		},
		&typeString,
	); err != nil {
		return ProjectExport{}, err
	}
	if typeString == "" {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export " + symbol.Name + " has an empty type",
		}
	}
	declarationHandles := symbol.Declarations
	var typeSymbol *symbolResponse
	if targetType.Symbol != 0 {
		if err := requestProjectJSON(
			p.client,
			"getSymbolOfType",
			getSymbolOfTypeParams{
				Snapshot: p.snapshot,
				Type:     targetType.ID,
			},
			&typeSymbol,
		); err != nil {
			return ProjectExport{}, err
		}
		if typeSymbol != nil && len(typeSymbol.Declarations) != 0 {
			declarationHandles = typeSymbol.Declarations
		}
	}
	declarations := make([]string, 0, len(declarationHandles))
	seen := make(map[string]struct{}, len(declarationHandles))
	for _, handle := range declarationHandles {
		path, ok := declarationPath(handle)
		if !ok {
			return ProjectExport{}, &ProjectInspectionError{
				Operation: "exports",
				Path:      sourcePath,
				Reason:    "export " + symbol.Name + " has an invalid declaration handle",
			}
		}
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		declarations = append(declarations, path)
	}
	sort.Strings(declarations)
	if len(declarations) == 0 {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export " + symbol.Name + " has no declaration owner",
		}
	}
	return ProjectExport{
		name:         symbol.Name,
		flags:        symbol.Flags,
		typeString:   typeString,
		declarations: declarations,
	}, nil
}

func requestProjectJSON(
	client *Client,
	operation string,
	params interface{},
	result interface{},
) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return &ClientError{
			Operation: "encode " + operation + " request",
			Reason:    err.Error(),
		}
	}
	response, err := client.request(operation, encoded)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response, result); err != nil {
		return &ClientError{
			Operation: "decode " + operation + " result",
			Reason:    err.Error(),
		}
	}
	return nil
}

func sameProjectPath(left string, right string) bool {
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)
	return leftErr == nil &&
		rightErr == nil &&
		filepath.Clean(leftAbsolute) == filepath.Clean(rightAbsolute)
}

func declarationPath(handle string) (string, bool) {
	first := strings.IndexByte(handle, '.')
	if first < 0 {
		return "", false
	}
	secondRelative := strings.IndexByte(handle[first+1:], '.')
	if secondRelative < 0 {
		return "", false
	}
	path := handle[first+1+secondRelative+1:]
	if path == "" {
		return "", false
	}
	return path, true
}

func projectInspectionError(
	operation string,
	path string,
	cause error,
) error {
	return &ProjectInspectionError{
		Operation: operation,
		Path:      path,
		Reason:    cause.Error(),
	}
}

type updateSnapshotParams struct {
	OpenProject string `json:"openProject"`
}

type updateSnapshotResponse struct {
	Snapshot uint64            `json:"snapshot"`
	Projects []projectResponse `json:"projects"`
}

type projectResponse struct {
	ID             string `json:"id"`
	ConfigFileName string `json:"configFileName"`
}

type getSourceFileParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	File     string `json:"file"`
}

type sourceFileResponse struct {
	Data string `json:"data"`
}

type getSymbolAtLocationParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Location string `json:"location"`
}

type getExportsOfSymbolParams struct {
	Snapshot uint64 `json:"snapshot"`
	Symbol   uint64 `json:"symbol"`
}

type getTypeOfSymbolParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Symbol   uint64 `json:"symbol"`
}

type typeToStringParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Type     uint32 `json:"type"`
}

type getSymbolOfTypeParams struct {
	Snapshot uint64 `json:"snapshot"`
	Type     uint32 `json:"type"`
}

type symbolResponse struct {
	ID           uint64   `json:"id"`
	Name         string   `json:"name"`
	Flags        uint32   `json:"flags"`
	Declarations []string `json:"declarations"`
}

type typeResponse struct {
	ID     uint32 `json:"id"`
	Symbol uint64 `json:"symbol"`
}
