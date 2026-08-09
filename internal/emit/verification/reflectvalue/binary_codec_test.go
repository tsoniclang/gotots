package reflectvalue_test

import "testing"

// TestBinaryCodecCanonicalizesWithNativeEvidence covers encoding/binary Read and Write over the
// completed reflection value model: struct and slice payloads across both
// byte orders, exact wire bytes, in-place decode through pointers, and
// the exact Go invalid-type errors for platform-sized integers.
func TestBinaryCodecCanonicalizesWithNativeEvidence(t *testing.T) {
	source := `package reflectvalue

import (
	"encoding/binary"
	"fmt"
)

type Header struct {
	Magic uint32
	Flag  bool
	Depth int16
	Ratio float64
}

type pipe struct {
	data []byte
	at   int
}

func (p *pipe) Write(chunk []byte) (int, error) {
	p.data = append(p.data, chunk...)
	return len(chunk), nil
}

func (p *pipe) Read(target []byte) (int, error) {
	count := copy(target, p.data[p.at:])
	p.at += count
	return count, nil
}

func Transfer() string {
	channel := &pipe{}
	header := Header{Magic: 0xCAFEBABE, Flag: true, Depth: -7, Ratio: 2.5}
	if err := binary.Write(channel, binary.BigEndian, header); err != nil {
		return err.Error()
	}
	encoded := ""
	for _, value := range channel.data {
		encoded += fmt.Sprintf("%02x", value)
	}
	var decoded Header
	if err := binary.Read(channel, binary.BigEndian, &decoded); err != nil {
		return err.Error()
	}
	words := &pipe{}
	if err := binary.Write(words, binary.LittleEndian, []uint16{513, 4}); err != nil {
		return err.Error()
	}
	lanes := make([]uint16, 2)
	if err := binary.Read(words, binary.LittleEndian, lanes); err != nil {
		return err.Error()
	}
	invalid := binary.Write(&pipe{}, binary.BigEndian, 5)
	return fmt.Sprintf(
		"%s %d %t %d %g %d %d %d %q",
		encoded,
		decoded.Magic, decoded.Flag, decoded.Depth, decoded.Ratio,
		len(words.data), int(lanes[0]), int(lanes[1]),
		invalid.Error(),
	)
}
`
	typescriptRunner := `const facts = await Transfer();
console.log(facts);
`
	goRunner := `package main

import (
	"fmt"

	fixture "example.com/reflectvalue"
)

func main() {
	fmt.Println(fixture.Transfer())
}
`
	verifyReflectCanonical(
		t,
		source,
		"Transfer",
		"reflectvalue",
		typescriptRunner,
		goRunner,
	)
}
