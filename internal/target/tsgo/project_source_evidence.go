package tsgo

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type projectNodeEvidence struct {
	kind   uint32
	parent uint32
	data   uint32
}

type projectSourceEvidence struct {
	nodes []projectNodeEvidence
	wire  []byte
}

type officialEncodedNode interface {
	officialEncoding() []byte
}

type officialSourceFile struct {
	nodeCore
	wire []byte
}

func (*officialSourceFile) isDeclarationBase() {}
func (*officialSourceFile) isNodeBase()        {}
func (*officialSourceFile) isSourceFile()      {}

func (s *officialSourceFile) Statements() []Statement {
	panic("official source file is opaque outside TS-Go encoding")
}

func (s *officialSourceFile) EndOfFileToken() EndOfFile {
	panic("official source file is opaque outside TS-Go encoding")
}

func (s *officialSourceFile) SourceData() SourceFileData {
	panic("official source file is opaque outside TS-Go encoding")
}

func (s *officialSourceFile) targetEncoding() nodeEncoding {
	panic("official source file must use its official TS-Go encoding")
}

func (s *officialSourceFile) officialEncoding() []byte {
	return bytes.Clone(s.wire)
}

func (p *ProjectInspection) SourceFile(sourceFile string) (SourceFile, error) {
	if p == nil || p.client == nil || p.snapshot == 0 || p.project == "" {
		return nil, &ProjectInspectionError{
			Operation: "source file",
			Path:      sourceFile,
			Reason:    "project inspection is invalid",
		}
	}
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("source file", sourceFile, err)
	}
	path := filepath.ToSlash(absolute)
	evidence, err := p.projectSourceEvidence(path)
	if err != nil {
		return nil, err
	}
	if len(evidence.wire) == 0 {
		return nil, &ProjectInspectionError{
			Operation: "source file",
			Path:      path,
			Reason:    "official AST encoding is absent",
		}
	}
	return &officialSourceFile{
		nodeCore: newNodeCore(SyntaxKindSourceFile, NodeFlagsNone),
		wire:     bytes.Clone(evidence.wire),
	}, nil
}

func (p *ProjectInspection) projectSourceEvidence(
	sourcePath string,
) (projectSourceEvidence, error) {
	if source, ok := p.sources[sourcePath]; ok {
		return source, nil
	}
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
		return projectSourceEvidence{}, err
	}
	if encoded == nil || encoded.Data == "" {
		return projectSourceEvidence{}, &ProjectInspectionError{
			Operation: "source evidence",
			Path:      sourcePath,
			Reason:    "source file is absent from the project",
		}
	}
	source, err := decodeProjectSourceEvidence(encoded.Data)
	if err != nil {
		return projectSourceEvidence{}, &ProjectInspectionError{
			Operation: "source evidence",
			Path:      sourcePath,
			Reason:    err.Error(),
		}
	}
	p.sources[sourcePath] = source
	return source, nil
}

func decodeProjectSourceEvidence(
	encoded string,
) (projectSourceEvidence, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return projectSourceEvidence{}, fmt.Errorf("decode official AST: %w", err)
	}
	if len(raw) < headerSize {
		return projectSourceEvidence{}, fmt.Errorf(
			"official AST header has %d bytes, want at least %d",
			len(raw),
			headerSize,
		)
	}
	metadata := binary.LittleEndian.Uint32(raw[headerOffsetMetadata:])
	if version := metadata >> 24; version != protocolVersion {
		return projectSourceEvidence{}, fmt.Errorf(
			"official AST protocol version is %d, want %d",
			version,
			protocolVersion,
		)
	}
	nodesOffset := int(binary.LittleEndian.Uint32(raw[headerOffsetNodes:]))
	if nodesOffset < headerSize || nodesOffset > len(raw) ||
		(len(raw)-nodesOffset)%nodeLen != 0 {
		return projectSourceEvidence{}, fmt.Errorf(
			"official AST node region is invalid",
		)
	}
	nodeCount := (len(raw) - nodesOffset) / nodeLen
	if nodeCount < 2 {
		return projectSourceEvidence{}, fmt.Errorf("official AST has no source node")
	}
	nodes := make([]projectNodeEvidence, nodeCount)
	for index := range nodeCount {
		offset := nodesOffset + index*nodeLen
		nodes[index] = projectNodeEvidence{
			kind: binary.LittleEndian.Uint32(
				raw[offset+nodeOffsetKind:],
			),
			parent: binary.LittleEndian.Uint32(
				raw[offset+nodeOffsetParent:],
			),
			data: binary.LittleEndian.Uint32(
				raw[offset+nodeOffsetData:],
			),
		}
		if nodes[index].parent >= uint32(nodeCount) {
			return projectSourceEvidence{}, fmt.Errorf(
				"official AST node %d has invalid parent %d",
				index,
				nodes[index].parent,
			)
		}
	}
	return projectSourceEvidence{nodes: nodes, wire: raw}, nil
}

func (p *ProjectInspection) projectMemberAccess(
	declarations []string,
) (ProjectMemberAccess, error) {
	var result ProjectMemberAccess
	for _, handle := range declarations {
		index, kind, sourcePath, err := parseProjectNodeHandle(handle)
		if err != nil {
			return ProjectMemberAccessInvalid, err
		}
		source, err := p.projectSourceEvidence(sourcePath)
		if err != nil {
			return ProjectMemberAccessInvalid, err
		}
		access, err := source.declarationAccess(index, kind)
		if err != nil {
			return ProjectMemberAccessInvalid, err
		}
		if result != ProjectMemberAccessInvalid && result != access {
			return ProjectMemberAccessInvalid, fmt.Errorf(
				"declarations disagree on accessibility",
			)
		}
		result = access
	}
	if result == ProjectMemberAccessInvalid {
		return ProjectMemberAccessInvalid, fmt.Errorf(
			"declaration accessibility is absent",
		)
	}
	return result, nil
}

func (p *ProjectInspection) projectTypeParameterCount(
	declarations []string,
) (int, error) {
	count := -1
	for _, handle := range declarations {
		index, kind, sourcePath, err := parseProjectNodeHandle(handle)
		if err != nil {
			return 0, err
		}
		source, err := p.projectSourceEvidence(sourcePath)
		if err != nil {
			return 0, err
		}
		selected, err := source.declarationTypeParameterCount(index, kind)
		if err != nil {
			return 0, err
		}
		if count >= 0 && count != selected {
			return 0, fmt.Errorf("declarations disagree on type-parameter count")
		}
		count = selected
	}
	if count < 0 {
		return 0, fmt.Errorf("declaration type-parameter evidence is absent")
	}
	return count, nil
}

func parseProjectNodeHandle(
	handle string,
) (uint32, uint32, string, error) {
	first := strings.IndexByte(handle, '.')
	if first <= 0 {
		return 0, 0, "", fmt.Errorf("invalid declaration handle")
	}
	secondRelative := strings.IndexByte(handle[first+1:], '.')
	if secondRelative <= 0 {
		return 0, 0, "", fmt.Errorf("invalid declaration handle")
	}
	second := first + 1 + secondRelative
	indexValue, indexErr := strconv.ParseUint(handle[:first], 10, 32)
	kindValue, kindErr := strconv.ParseUint(handle[first+1:second], 10, 32)
	sourcePath := handle[second+1:]
	if indexErr != nil || kindErr != nil || indexValue == 0 || sourcePath == "" {
		return 0, 0, "", fmt.Errorf("invalid declaration handle")
	}
	return uint32(indexValue), uint32(kindValue), sourcePath, nil
}

func (s projectSourceEvidence) declarationAccess(
	index uint32,
	kind uint32,
) (ProjectMemberAccess, error) {
	if index >= uint32(len(s.nodes)) || s.nodes[index].kind != kind {
		return ProjectMemberAccessInvalid, fmt.Errorf(
			"declaration handle does not identify its official AST node",
		)
	}
	for childIndex, child := range s.nodes {
		if child.parent != index {
			continue
		}
		if child.kind == uint32(SyntaxKindPrivateIdentifier) {
			return ProjectMemberAccessNonPublic, nil
		}
		if child.kind != kindNodeList {
			continue
		}
		for _, modifier := range s.nodes {
			if modifier.parent != uint32(childIndex) {
				continue
			}
			if modifier.kind == uint32(SyntaxKindPrivateKeyword) ||
				modifier.kind == uint32(SyntaxKindProtectedKeyword) {
				return ProjectMemberAccessNonPublic, nil
			}
		}
	}
	return ProjectMemberAccessPublic, nil
}

func (s projectSourceEvidence) declarationTypeParameterCount(
	index uint32,
	kind uint32,
) (int, error) {
	if index >= uint32(len(s.nodes)) || s.nodes[index].kind != kind {
		return 0, fmt.Errorf(
			"declaration handle does not identify its official AST node",
		)
	}
	count := 0
	for listIndex, list := range s.nodes {
		if list.parent != index || list.kind != kindNodeList {
			continue
		}
		for _, child := range s.nodes {
			if child.parent == uint32(listIndex) &&
				child.kind == uint32(SyntaxKindTypeParameter) {
				count++
			}
		}
	}
	return count, nil
}
