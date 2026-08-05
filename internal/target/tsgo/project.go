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
	sources  map[string]projectSourceEvidence
}

type ProjectExport struct {
	name               string
	flags              uint32
	typeString         string
	declaredTypeString string
	typeID             uint32
	declaredTypeID     uint32
	typeSymbolID       uint64
	typeParameters     int
	exportHandles      []string
	handles            []string
	declarations       []string
	ownerKeys          []string
	valueMembers       []ProjectMember
	typeMembers        []ProjectMember
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

func (e ProjectExport) DeclaredTypeString() string {
	return e.declaredTypeString
}

func (e ProjectExport) TypeParameterCount() int {
	return e.typeParameters
}

func (e ProjectExport) Declarations() []string {
	return slices.Clone(e.declarations)
}

func (e ProjectExport) ImplementationOwners() []string {
	return slices.Clone(e.ownerKeys)
}

func (e ProjectExport) ValueMembers() []ProjectMember {
	return slices.Clone(e.valueMembers)
}

func (e ProjectExport) TypeMembers() []ProjectMember {
	return slices.Clone(e.typeMembers)
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
		sources:  make(map[string]projectSourceEvidence),
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
	if _, err := p.projectSourceEvidence(sourcePath); err != nil {
		return nil, err
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
	targetDeclarations, found, err := p.projectTypeDeclarations(targetType)
	if err != nil {
		return ProjectExport{}, err
	}
	if found {
		declarationHandles = targetDeclarations
	}
	var declaredType *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getDeclaredTypeOfSymbol",
		getTypeOfSymbolParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Symbol:   symbol.ID,
		},
		&declaredType,
	); err != nil {
		return ProjectExport{}, err
	}
	var declaredTypeString string
	if declaredType != nil && declaredType.ID != 0 {
		if err := requestProjectJSON(
			p.client,
			"typeToString",
			typeToStringParams{
				Snapshot: p.snapshot,
				Project:  p.project,
				Type:     declaredType.ID,
			},
			&declaredTypeString,
		); err != nil {
			return ProjectExport{}, err
		}
	}
	declaredDeclarations, found, err :=
		p.projectTypeDeclarations(declaredType)
	if err != nil {
		return ProjectExport{}, err
	}
	if found {
		declarationHandles = declaredDeclarations
	}
	typeParameters, err := p.projectTypeParameterCount(declarationHandles)
	if err != nil {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export " + symbol.Name + " " + err.Error(),
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
	ownerKeys, err := projectOwnerKeys(
		declarations,
		filepath.Dir(p.config),
	)
	if err != nil {
		return ProjectExport{}, &ProjectInspectionError{
			Operation: "exports",
			Path:      sourcePath,
			Reason:    "export " + symbol.Name + " " + err.Error(),
		}
	}
	valueMembers, err := p.projectMembers(
		sourcePath,
		"value members of "+symbol.Name,
		targetType.ID,
	)
	if err != nil {
		return ProjectExport{}, err
	}
	var typeMembers []ProjectMember
	if declaredType != nil && declaredType.ID != 0 {
		typeMembers, err = p.projectMembers(
			sourcePath,
			"type members of "+symbol.Name,
			declaredType.ID,
		)
		if err != nil {
			return ProjectExport{}, err
		}
	}
	return ProjectExport{
		name:               symbol.Name,
		flags:              symbol.Flags,
		typeString:         typeString,
		declaredTypeString: declaredTypeString,
		typeID:             targetType.ID,
		declaredTypeID:     typeResponseID(declaredType),
		typeSymbolID:       preferredTypeSymbol(declaredType, targetType),
		typeParameters:     typeParameters,
		exportHandles:      sortedStrings(symbol.Declarations),
		handles:            sortedStrings(declarationHandles),
		declarations:       declarations,
		ownerKeys:          ownerKeys,
		valueMembers:       valueMembers,
		typeMembers:        typeMembers,
	}, nil
}

func preferredTypeSymbol(primary *typeResponse, fallback *typeResponse) uint64 {
	if primary != nil && primary.Symbol != 0 {
		return primary.Symbol
	}
	if fallback != nil {
		return fallback.Symbol
	}
	return 0
}

func typeResponseID(source *typeResponse) uint32 {
	if source == nil {
		return 0
	}
	return source.ID
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

func declarationsWithinProject(handles []string, root string) bool {
	if len(handles) == 0 {
		return false
	}
	for _, handle := range handles {
		path, ok := declarationPath(handle)
		if !ok {
			return false
		}
		relative, err := filepath.Rel(root, filepath.FromSlash(path))
		if err != nil ||
			relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
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

type getPropertiesOfTypeParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Type     uint32 `json:"type"`
}

type symbolResponse struct {
	ID           uint64   `json:"id"`
	Name         string   `json:"name"`
	Flags        uint32   `json:"flags"`
	Declarations []string `json:"declarations"`
}

type typeResponse struct {
	ID          uint32 `json:"id"`
	Flags       uint32 `json:"flags"`
	ObjectFlags uint32 `json:"objectFlags"`
	Target      uint32 `json:"target"`
	Symbol      uint64 `json:"symbol"`
}
