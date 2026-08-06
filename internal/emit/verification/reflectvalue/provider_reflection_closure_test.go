package reflectvalue_test

import (
	"strings"
	"testing"
)

func requireProviderStateOperations(
	t *testing.T,
	printed string,
	representation string,
	operations ...string,
) {
	t.Helper()
	alias := ""
	for _, line := range strings.Split(printed, "\n") {
		if !strings.Contains(line, "export const $goProviderState_") ||
			!strings.Contains(line, representation) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			alias = strings.TrimSuffix(fields[2], ":")
			break
		}
	}
	if alias == "" {
		t.Fatalf("provider closure artifact lacks state for %q", representation)
	}
	for _, operation := range operations {
		required := alias + ".$" + operation + "("
		if !strings.Contains(printed, required) {
			t.Fatalf(
				"provider closure artifact lacks %s operation %q",
				representation,
				required,
			)
		}
	}
}

func TestReachedProviderReflectionClosureMatchesGo(t *testing.T) {
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
	typescriptRunner := `const facts = await ProviderReflectionClosure();
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
	runReflectDifferentialInspect(
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
				"provider_bufio_reader.CanonicalBufioReader",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"provider_bufio_writer.CanonicalBufioWriter",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"provider_compress_gzip.CanonicalGzipReader",
				"assign",
				"copy",
			)
			requireProviderStateOperations(
				t,
				artifacts.printed,
				"named_io_fs.CanonicalPathError",
				"assign",
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
					t.Fatalf("provider closure artifact lacks %q", required)
				}
			}
		},
	)
}
