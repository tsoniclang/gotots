package reflectvalue_test

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestReflectionMetadataAndValueOperationsAreDeferred(t *testing.T) {
	project := t.TempDir()
	emission := compileReflectFixture(
		t,
		project,
		`package reflectvalue

import "reflect"

func Facts() int64 { return reflect.ValueOf(int64(1)).Int() }
`,
		[]string{"Facts"},
	)
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	for _, required := range []string{
		".$create(() => ({",
		".$registerValue(",
		", () => ({",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"reflection artifact lacks lazy constructor %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
}

// TestDynamicPointerTypeCompositionCanonicalizesWithNativeEvidence covers PointerTo
// one canonical descriptor from a runtime-flowing Type without requiring a
// statically mentioned pointer type. It also proves value- and pointer-method
// implementation sets and repeated pointer composition.
func TestDynamicPointerTypeCompositionCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"fmt"
	"reflect"
)

type Marker interface { Mark() }

type ValueMethod struct{}
func (ValueMethod) Mark() {}

type PointerMethod struct{}
func (*PointerMethod) Mark() {}

func DynamicPointerFacts() string {
	marker := reflect.TypeOf((*Marker)(nil)).Elem()
	value := reflect.TypeOf(ValueMethod{})
	pointerOnly := reflect.TypeOf(PointerMethod{})
	valuePointer := reflect.PointerTo(value)
	pointerOnlyPointer := reflect.PointerTo(pointerOnly)
	doublePointer := reflect.PointerTo(valuePointer)
	again := reflect.PointerTo(value)
	return fmt.Sprintf(
		"%s %s %t %t %t %t %t %d %d",
		valuePointer.String(), doublePointer.String(),
		valuePointer == again, valuePointer.Elem() == value,
		value.Implements(marker), valuePointer.Implements(marker),
		pointerOnlyPointer.Implements(marker),
		valuePointer.Size(), valuePointer.Align(),
	)
}
`
	typescriptRunner := `const facts = DynamicPointerFacts();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.DynamicPointerFacts())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"DynamicPointerFacts",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			for _, required := range []string{
				"methodSet:",
				"pointerMethodSet:",
				"pointerInheritsMethods: true",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"dynamic pointer artifact lacks %q",
						required,
					)
				}
			}
		},
	)
}

func requireProviderStateOperations(
	t *testing.T,
	printed string,
	representation string,
	operations ...string,
) {
	t.Helper()
	exported := ""
	for _, line := range strings.Split(printed, "\n") {
		if !strings.Contains(line, "export const $goProviderState$") ||
			!strings.Contains(line, representation) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			exported = strings.TrimSuffix(fields[2], ":")
			break
		}
	}
	if exported == "" {
		t.Fatalf("provider closure artifact lacks state for %q", representation)
	}
	aliases := providerStateImportAliases(printed, exported)
	for _, operation := range operations {
		found := false
		for _, alias := range aliases {
			if strings.Contains(printed, alias+".$"+operation+"(") {
				found = true
				break
			}
		}
		if !found {
			relevant := make([]string, 0)
			for _, line := range strings.Split(printed, "\n") {
				for _, alias := range aliases {
					if strings.Contains(line, alias) {
						relevant = append(relevant, line)
						break
					}
				}
			}
			t.Fatalf(
				"provider closure artifact lacks %s operation %q through %v:\n%s",
				representation,
				operation,
				aliases,
				strings.Join(relevant, "\n"),
			)
		}
	}
}

func providerStateImportAliases(printed string, exported string) []string {
	result := []string{exported}
	for _, line := range strings.Split(printed, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "import {") ||
			!strings.Contains(line, exported) {
			continue
		}
		open := strings.Index(line, "{")
		close := strings.LastIndex(line, "}")
		if open < 0 || close <= open {
			continue
		}
		for _, imported := range strings.Split(line[open+1:close], ",") {
			fields := strings.Fields(imported)
			if len(fields) == 3 && fields[0] == exported && fields[1] == "as" {
				result = append(result, fields[2])
			}
		}
	}
	return result
}

func TestReachedProviderReflectionClosureCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"time"
)

func ProviderReflectionClosure() string {
	readerSource := bufio.NewReader(bytes.NewBuffer([]byte("abc")))
	readerFirst, _ := readerSource.ReadByte()
	readerTarget := bufio.NewReader(bytes.NewBuffer([]byte("zzz")))
	reflect.ValueOf(readerTarget).Elem().Set(
		reflect.ValueOf(readerSource).Elem(),
	)
	readerSourceNext, _ := readerSource.ReadByte()
	readerTargetNext, _ := readerTarget.ReadByte()

	writerSink := bytes.NewBuffer(nil)
	writerSource := bufio.NewWriter(writerSink)
	_, _ = writerSource.Write([]byte("abc"))
	writerTarget := bufio.NewWriter(bytes.NewBuffer(nil))
	reflect.ValueOf(writerTarget).Elem().Set(
		reflect.ValueOf(writerSource).Elem(),
	)
	_ = writerSource.WriteByte('d')
	_ = writerTarget.WriteByte('e')
	_ = writerSource.Flush()
	_ = writerTarget.Flush()

	bufferSource := bytes.NewBuffer(nil)
	bufferSource.Grow(8)
	_, _ = bufferSource.Write([]byte("abc"))
	bufferTarget := bytes.NewBuffer(nil)
	reflect.ValueOf(bufferTarget).Elem().Set(
		reflect.ValueOf(bufferSource).Elem(),
	)
	_ = bufferSource.Next(1)

	encoded := []byte{
		31, 139, 8, 0, 0, 0, 0, 0, 0, 3, 171, 0, 0, 131, 22, 220,
		140, 1, 0, 0, 0,
	}
	gzipSource, gzipFailure := gzip.NewReader(bytes.NewBuffer(encoded))
	if gzipFailure != nil {
		return gzipFailure.Error()
	}
	gzipTarget, gzipTargetFailure := gzip.NewReader(bytes.NewBuffer(encoded))
	if gzipTargetFailure != nil {
		return gzipTargetFailure.Error()
	}
	reflect.ValueOf(gzipTarget).Elem().Set(
		reflect.ValueOf(gzipSource).Elem(),
	)

	pathSource := &fs.PathError{
		Op:   "open",
		Path: "missing",
		Err:  errors.New("denied"),
	}
	pathTarget := &fs.PathError{}
	reflect.ValueOf(pathTarget).Elem().Set(
		reflect.ValueOf(pathSource).Elem(),
	)
	pathValues := map[fs.PathError]string{*pathSource: "path-hit"}

	fileSource := *os.Stdin
	fileTarget := *os.Stdout
	reflect.ValueOf(&fileTarget).Elem().Set(
		reflect.ValueOf(&fileSource).Elem(),
	)
	fileValues := map[os.File]string{fileSource: "file-hit"}

	stringSource := strings.NewReader("reader")
	stringTarget := strings.NewReader("target")
	reflect.ValueOf(stringTarget).Elem().Set(
		reflect.ValueOf(stringSource).Elem(),
	)
	stringValues := map[strings.Reader]string{*stringSource: "reader-hit"}
	stringByte := make([]byte, 1)
	_, _ = stringTarget.Read(stringByte)

	var parsed time.Time
	parseFailure := parsed.UnmarshalJSON([]byte(` + "`\"2024-01-02T03:04:05Z\"`" + `))
	invalidFailure := parsed.UnmarshalJSON([]byte("123"))

	var nilBuffer *bytes.Buffer
	return fmt.Sprintf(
		"reader=%c/%c/%c writer=%s buffer=%s/%s gzip=%s/%d path=%s/%s file=%s/%d strings=%s/%c methods=%s/%s/%t/%s",
		readerFirst,
		readerSourceNext,
		readerTargetNext,
		writerSink.String(),
		bufferSource.String(),
		bufferTarget.String(),
		gzipTarget.Name,
		gzipTarget.OS,
		pathValues[*pathTarget],
		pathTarget.Error(),
		fileValues[fileTarget],
		fileTarget.Fd(),
		stringValues[*stringTarget],
		stringByte[0],
		nilBuffer.String(),
		fs.FileMode(0o644).String(),
		parseFailure == nil,
		invalidFailure.Error(),
	)
}
`
	typescriptRunner := `const facts = ProviderReflectionClosure();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.ProviderReflectionClosure())
}
`
	verifyReflectCanonicalInspect(
		t,
		source,
		"ProviderReflectionClosure",
		"reflectvalue",
		typescriptRunner,
		goRunner,
		func(artifacts renderedArtifacts) {
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"provider_bufio_reader.DirectBufioReader",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"provider_bufio_writer.DirectBufioWriter",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"provider_compress_gzip_direct.DirectGzipReader",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"fs.PathError",
				"copy",
				"equal",
				"hash",
			)
			for _, required := range []string{
				"named_bytes.BytesBufferOperations.$assign",
				"named_bytes.BytesBufferOperations.$copy",
				"named_os.OsFileOperations.$assign",
				"named_os.OsFileOperations.$copy",
				"named_os.OsFileOperations.$equal",
				"named_os.OsFileOperations.$hash",
				"named_strings.StringsReaderOperations.$assign",
				"named_strings.StringsReaderOperations.$copy",
				"named_strings.StringsReaderOperations.$equal",
				"named_strings.StringsReaderOperations.$hash",
				"time__from_gostdlib.Time.UnmarshalJSON",
				"bytes__from_gostdlib.Buffer.String",
				"named_io_fs.IoFsFileModeValueOperations.$wrap(420).String()",
			} {
				if !strings.Contains(artifacts.printed, required) {
					relevant := make([]string, 0)
					for _, line := range strings.Split(artifacts.printed, "\n") {
						if strings.Contains(line, "PathError") {
							relevant = append(relevant, line)
						}
					}
					t.Fatalf(
						"provider closure artifact lacks %q:\n%s",
						required,
						strings.Join(relevant, "\n"),
					)
				}
			}
		},
	)
}

type reflectionRegistrationMeasurement struct {
	count       int
	bytes       int
	nodes       int
	runtimeSize int
}

func TestReflectionRegistrationsShareBoundedCommonMachinery(t *testing.T) {
	const (
		maxAddressableRegistrationBytesPerType = 4_100
		maxAddressableRegistrationNodesPerType = 600
	)
	small := measureReflectionRegistrations(t, 8)
	large := measureReflectionRegistrations(t, 32)
	countDelta := large.count - small.count
	byteDelta := large.bytes - small.bytes
	nodeDelta := large.nodes - small.nodes
	if small.runtimeSize != large.runtimeSize {
		t.Fatalf(
			"common reflection owner changed with type count: %d -> %d",
			small.runtimeSize,
			large.runtimeSize,
		)
	}
	if countDelta != 24 || byteDelta/countDelta > maxAddressableRegistrationBytesPerType ||
		nodeDelta/countDelta > maxAddressableRegistrationNodesPerType {
		t.Fatalf(
			"reflection registration growth is not bounded: counts=%d/%d bytes=%d/%d nodes=%d/%d",
			small.count,
			large.count,
			small.bytes,
			large.bytes,
			small.nodes,
			large.nodes,
		)
	}
	t.Logf(
		"reflection registrations counts=%d/%d bytes=%d/%d nodes=%d/%d runtime=%d",
		small.count,
		large.count,
		small.bytes,
		large.bytes,
		small.nodes,
		large.nodes,
		large.runtimeSize,
	)
}

func measureReflectionRegistrations(
	t *testing.T,
	count int,
) reflectionRegistrationMeasurement {
	t.Helper()
	project := t.TempDir()
	var source strings.Builder
	source.WriteString("package reflectvalue\n\nimport \"reflect\"\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"type Record%d struct { Count int; Name string; Ready bool }\n",
			index,
		)
	}
	source.WriteString("\nfunc Audit(index int) int {\n\tvalues := []any{")
	for index := range count {
		fmt.Fprintf(&source, "&Record%d{},", index)
	}
	source.WriteString(`}
	total := 0
	for _, value := range values {
		reflected := reflect.ValueOf(value).Elem()
		total += reflected.NumField()
		if index >= 0 && index < reflected.NumField() && reflected.Field(index).CanSet() {
			total++
		}
	}
	return total
}
`)
	emission := compileReflectFixture(
		t,
		project,
		source.String(),
		[]string{"Audit"},
	)
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	measurement := reflectionRegistrationMeasurement{count: count}
	for _, file := range emission.Files() {
		if file.OutputPath() != output.ReflectionTypeSupportPath {
			continue
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		measurement.nodes = encodedReflectionNodes(t, encoded)
		for _, path := range artifacts.paths {
			if !strings.HasSuffix(
				filepath.ToSlash(path),
				output.ReflectionTypeSupportPath,
			) {
				continue
			}
			printed, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			measurement.bytes = len(printed)
			text := string(printed)
			if structs := strings.Count(text, ".$registerStruct("); structs != count {
				t.Fatalf("struct registrations = %d, want %d", structs, count)
			}
			if pointers := strings.Count(text, ".$registerPointer("); pointers < count {
				t.Fatalf("pointer registrations = %d, want at least %d", pointers, count)
			}
			lazyStructs := regexp.MustCompile(
				`(?s)\.\$registerStruct\(\s*[^,]+,\s*\(\)\s*=>`,
			).FindAllString(text, -1)
			if len(lazyStructs) != count {
				t.Fatalf("lazy struct registrations = %d, want %d", len(lazyStructs), count)
			}
			lazyStructFields := regexp.MustCompile(
				`(?s)\.\$registerStruct\([^;]+,\s*fields\s*=>`,
			).FindAllString(text, -1)
			if len(lazyStructFields) != count {
				t.Fatalf("lazy struct field factories = %d, want %d", len(lazyStructFields), count)
			}
			lazyPointers := regexp.MustCompile(
				`(?s)\.\$registerPointer\(\s*[^,]+,\s*\(\)\s*=>`,
			).FindAllString(text, -1)
			if len(lazyPointers) < count {
				t.Fatalf("lazy pointer registrations = %d, want at least %d", len(lazyPointers), count)
			}
			lazyPointerElements := regexp.MustCompile(
				`(?s)\.\$registerPointer\([^;]+,\s*elements\s*=>`,
			).FindAllString(text, -1)
			if len(lazyPointerElements) < count {
				t.Fatalf("lazy pointer element factories = %d, want at least %d", len(lazyPointerElements), count)
			}
			for _, forbidden := range []string{
				"switch (index)",
				".$registerValue($goReflectType$Named_reflectvalue$Record",
				".$registerValue($goReflectType$PointerTo_Named_reflectvalue$Record",
			} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("reflection registrations retain %q", forbidden)
				}
			}
		}
	}
	runtimeSource, err := os.ReadFile(filepath.Join(
		repositoryRoot(),
		"gostdlib",
		"src",
		"internal",
		"portable",
		"reflect",
		"runtime-value.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	measurement.runtimeSize = len(runtimeSource)
	if measurement.bytes == 0 || measurement.nodes == 0 {
		t.Fatalf("reflection registration artifact is absent: %#v", measurement)
	}
	return measurement
}

func encodedReflectionNodes(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded target is %d bytes, want protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize || nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded target has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
