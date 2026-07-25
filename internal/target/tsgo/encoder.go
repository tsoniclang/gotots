package tsgo

import (
	"encoding/binary"
	"fmt"
	"math"
)

type EncodeError struct {
	Kind   SyntaxKind
	Field  string
	Reason string
}

func (e *EncodeError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("encode TS-Go node kind %d: %s", e.Kind, e.Reason)
	}
	return fmt.Sprintf("encode TS-Go node kind %d field %s: %s", e.Kind, e.Field, e.Reason)
}

func EncodeSourceFile(sourceFile SourceFile) ([]byte, error) {
	return EncodeNode(sourceFile)
}

func EncodeNode(root Node) ([]byte, error) {
	if root == nil {
		return nil, &EncodeError{Reason: "root is nil"}
	}
	state := encoderState{
		strings:    newStringTable(),
		nodeValues: make([]uint32, nodeLen/4),
		active:     make(map[Node]struct{}),
	}
	if err := state.appendRoot(root); err != nil {
		return nil, err
	}
	return state.finish()
}

type encoderState struct {
	strings        stringTable
	extendedData   []byte
	structuredData []byte
	nodeValues     []uint32
	nodeCount      uint32
	parentIndex    uint32
	previousIndex  uint32
	active         map[Node]struct{}
}

func (s *encoderState) appendRoot(root Node) error {
	s.nodeCount = 1
	data, err := s.nodeData(root)
	if err != nil {
		return err
	}
	s.appendNodeRow(root, 0, data)
	s.parentIndex = 1
	s.previousIndex = 0
	if err := s.visitChildren(root); err != nil {
		return err
	}
	return nil
}

func (s *encoderState) visitNode(node Node) error {
	if node == nil {
		return &EncodeError{Reason: "present child is nil"}
	}
	if _, exists := s.active[node]; exists {
		return &EncodeError{Kind: node.Kind(), Reason: "cycle in target AST"}
	}
	s.active[node] = struct{}{}
	defer delete(s.active, node)

	s.nodeCount++
	current := s.nodeCount
	s.linkPrevious(current)
	data, err := s.nodeData(node)
	if err != nil {
		return err
	}
	s.appendNodeRow(node, s.parentIndex, data)

	savedParent := s.parentIndex
	s.parentIndex = current
	s.previousIndex = 0
	if err := s.visitChildren(node); err != nil {
		return err
	}
	s.previousIndex = current
	s.parentIndex = savedParent
	return nil
}

func (s *encoderState) visitNodeList(nodes []Node) error {
	s.nodeCount++
	current := s.nodeCount
	s.linkPrevious(current)
	s.nodeValues = append(
		s.nodeValues,
		kindNodeList,
		0,
		0,
		0,
		s.parentIndex,
		uint32(len(nodes)),
		0,
	)

	savedParent := s.parentIndex
	s.parentIndex = current
	s.previousIndex = 0
	for _, node := range nodes {
		if err := s.visitNode(node); err != nil {
			return err
		}
	}
	s.previousIndex = current
	s.parentIndex = savedParent
	return nil
}

func (s *encoderState) visitChildren(node Node) error {
	encoding := node.targetEncoding()
	for _, child := range encoding.children {
		if !child.present {
			if child.required {
				return &EncodeError{
					Kind:   node.Kind(),
					Field:  child.name,
					Reason: "required child is absent",
				}
			}
			continue
		}
		if child.raw {
			for _, rawChild := range child.nodes {
				if err := s.visitNode(rawChild); err != nil {
					return err
				}
			}
			continue
		}
		if child.nodes != nil || child.node == nil {
			if err := s.visitNodeList(child.nodes); err != nil {
				return err
			}
			continue
		}
		if err := s.visitNode(child.node); err != nil {
			return err
		}
	}
	return nil
}

func (s *encoderState) appendNodeRow(node Node, parent uint32, data uint32) {
	s.nodeValues = append(
		s.nodeValues,
		uint32(node.Kind()),
		encodedPosition(node.Pos()),
		encodedPosition(node.End()),
		0,
		parent,
		data,
		uint32(node.Flags()),
	)
}

func (s *encoderState) linkPrevious(current uint32) {
	if s.previousIndex == 0 {
		return
	}
	const nextField = 3
	s.nodeValues[int(s.previousIndex)*(nodeLen/4)+nextField] = current
}

func (s *encoderState) nodeData(node Node) (uint32, error) {
	encoding := node.targetEncoding()
	switch encoding.dataType {
	case nodeDataChildren:
		return nodeDataTypeChildren | encoding.commonData | childMask(encoding.children), nil
	case nodeDataString:
		index, err := s.strings.add(encoding.text)
		if err != nil {
			return 0, err
		}
		if index > nodeStringIndexMask {
			return 0, &EncodeError{Kind: node.Kind(), Reason: "string index exceeds protocol width"}
		}
		return nodeDataTypeString | encoding.commonData | index, nil
	case nodeDataExtended:
		offset := uint32(len(s.extendedData))
		if offset > nodeExtendedDataMask {
			return 0, &EncodeError{Kind: node.Kind(), Reason: "extended-data offset exceeds protocol width"}
		}
		if err := s.appendExtended(node, encoding); err != nil {
			return 0, err
		}
		return nodeDataTypeExtended | encoding.commonData | offset, nil
	default:
		return 0, &EncodeError{Kind: node.Kind(), Reason: "unknown node data type"}
	}
}

func (s *encoderState) appendExtended(node Node, encoding nodeEncoding) error {
	switch encoding.extended {
	case extendedLiteral:
		textIndex, err := s.strings.add(encoding.text)
		if err != nil {
			return err
		}
		s.extendedData = appendUint32s(s.extendedData, textIndex, uint32(encoding.tokenFlags))
	case extendedTemplate:
		textIndex, err := s.strings.add(encoding.text)
		if err != nil {
			return err
		}
		rawTextIndex, err := s.strings.add(encoding.rawText)
		if err != nil {
			return err
		}
		s.extendedData = appendUint32s(
			s.extendedData,
			textIndex,
			rawTextIndex,
			uint32(encoding.tokenFlags),
		)
	case extendedSourceFile:
		return s.appendSourceFile(encoding.sourceFile)
	case extendedUnsupported:
		return &EncodeError{Kind: node.Kind(), Reason: "node is not encodable by TS-Go"}
	default:
		return &EncodeError{Kind: node.Kind(), Reason: "missing extended-data owner"}
	}
	return nil
}

func (s *encoderState) appendSourceFile(source SourceFileData) error {
	textIndex, err := s.strings.add(source.Text)
	if err != nil {
		return err
	}
	fileNameIndex, err := s.strings.add(source.FileName)
	if err != nil {
		return err
	}
	pathIndex, err := s.strings.add(source.Path)
	if err != nil {
		return err
	}
	references, err := appendFileReferences(s.structuredData, source.ReferencedFiles)
	if err != nil {
		return err
	}
	s.structuredData = references.data
	typeReferences, err := appendFileReferences(s.structuredData, source.TypeReferenceDirectives)
	if err != nil {
		return err
	}
	s.structuredData = typeReferences.data
	libReferences, err := appendFileReferences(s.structuredData, source.LibReferenceDirectives)
	if err != nil {
		return err
	}
	s.structuredData = libReferences.data
	s.extendedData = appendUint32s(
		s.extendedData,
		textIndex,
		fileNameIndex,
		pathIndex,
		uint32(source.LanguageVariant),
		uint32(source.ScriptKind),
		references.offset,
		typeReferences.offset,
		libReferences.offset,
		noStructuredData,
		noStructuredData,
		noStructuredData,
		0,
	)
	return nil
}

func (s *encoderState) finish() ([]byte, error) {
	stringOffsets, stringData := s.strings.encode()
	nodes := make([]byte, 0, len(s.nodeValues)*4)
	nodes = appendUint32s(nodes, s.nodeValues...)

	offsetStringOffsets := headerSize
	offsetStringData := offsetStringOffsets + len(stringOffsets)
	offsetExtendedData := offsetStringData + len(stringData)
	offsetStructuredData := offsetExtendedData + len(s.extendedData)
	offsetNodes := offsetStructuredData + len(s.structuredData)
	if offsetNodes > math.MaxUint32-len(nodes) {
		return nil, &EncodeError{Reason: "encoded node exceeds uint32 protocol size"}
	}

	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(header[headerOffsetMetadata:], uint32(protocolVersion)<<24)
	binary.LittleEndian.PutUint32(header[headerOffsetStringTableOffsets:], uint32(offsetStringOffsets))
	binary.LittleEndian.PutUint32(header[headerOffsetStringTable:], uint32(offsetStringData))
	binary.LittleEndian.PutUint32(header[headerOffsetExtendedData:], uint32(offsetExtendedData))
	binary.LittleEndian.PutUint32(header[headerOffsetStructuredData:], uint32(offsetStructuredData))
	binary.LittleEndian.PutUint32(header[headerOffsetNodes:], uint32(offsetNodes))

	result := make([]byte, 0, offsetNodes+len(nodes))
	result = append(result, header...)
	result = append(result, stringOffsets...)
	result = append(result, stringData...)
	result = append(result, s.extendedData...)
	result = append(result, s.structuredData...)
	result = append(result, nodes...)
	return result, nil
}

func childMask(children []childEncoding) uint32 {
	var result uint32
	for index, child := range children {
		if child.present {
			result |= 1 << index
		}
	}
	return result
}

func encodedPosition(value int32) uint32 {
	if value < 0 {
		return 0
	}
	return uint32(value)
}

func appendUint32s(destination []byte, values ...uint32) []byte {
	for _, value := range values {
		destination = binary.LittleEndian.AppendUint32(destination, value)
	}
	return destination
}

type stringTable struct {
	offsets []uint32
	data    []byte
}

func newStringTable() stringTable {
	return stringTable{}
}

func (s *stringTable) add(value string) (uint32, error) {
	if len(s.offsets) > math.MaxUint32-2 {
		return 0, &EncodeError{Reason: "too many protocol strings"}
	}
	if len(s.data) > math.MaxUint32-len(value) {
		return 0, &EncodeError{Reason: "protocol string table exceeds uint32 size"}
	}
	index := uint32(len(s.offsets))
	start := uint32(len(s.data))
	s.data = append(s.data, value...)
	s.offsets = append(s.offsets, start, uint32(len(s.data)))
	return index, nil
}

func (s stringTable) encode() ([]byte, []byte) {
	offsets := make([]byte, 0, len(s.offsets)*4)
	offsets = appendUint32s(offsets, s.offsets...)
	return offsets, s.data
}
