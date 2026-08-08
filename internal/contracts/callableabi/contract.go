package callableabi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

type Projection uint8

const (
	ProjectionInvalid Projection = iota
	ProjectionIdentity
)

func (p Projection) Valid() bool {
	return p == ProjectionIdentity
}

func (p Projection) String() string {
	switch p {
	case ProjectionIdentity:
		return "identity"
	default:
		return fmt.Sprintf("projection(%d)", p)
	}
}

type Parameter struct {
	projection Projection
	targetType string
}

func NewParameter(targetType string) (Parameter, error) {
	if targetType == "" {
		return Parameter{}, &Error{Reason: "parameter contract is incomplete"}
	}
	return Parameter{
		projection: ProjectionIdentity,
		targetType: targetType,
	}, nil
}

func (p Parameter) Valid() bool {
	return p.projection == ProjectionIdentity && p.targetType != ""
}

func (p Parameter) Projection() Projection { return p.projection }
func (p Parameter) TargetType() string     { return p.targetType }

type Callable struct {
	identity    string
	parameters  []Parameter
	resultType  string
	fingerprint string
}

func PackageFunctionIdentity(packagePath string, name string) (string, error) {
	if strings.TrimSpace(packagePath) != packagePath || packagePath == "" ||
		strings.TrimSpace(name) != name || name == "" {
		return "", &Error{Reason: "package-function identity is incomplete"}
	}
	return "package-function\x00" + packagePath + "\x00" + name, nil
}

func MethodIdentity(
	packagePath string,
	receiver string,
	name string,
) (string, error) {
	if strings.TrimSpace(packagePath) != packagePath || packagePath == "" ||
		strings.TrimSpace(receiver) != receiver || receiver == "" ||
		strings.TrimSpace(name) != name || name == "" {
		return "", &Error{Reason: "method identity is incomplete"}
	}
	return "method\x00" + packagePath + "\x00" + receiver + "\x00" + name, nil
}

func New(
	identity string,
	parameters []Parameter,
	resultType string,
) (Callable, error) {
	if identity == "" || resultType == "" {
		return Callable{}, &Error{Reason: "callable contract is incomplete"}
	}
	digest := sha256.New()
	digest.Write([]byte(identity))
	digest.Write([]byte{0})
	for index, parameter := range parameters {
		if !parameter.Valid() {
			return Callable{}, &Error{Function: identity, Reason: fmt.Sprintf("parameter %d is invalid", index)}
		}
		digest.Write([]byte{byte(parameter.projection), 0})
		digest.Write([]byte(parameter.targetType))
		digest.Write([]byte{0})
	}
	digest.Write([]byte(resultType))
	digest.Write([]byte{0})
	return Callable{
		identity:    identity,
		parameters:  slices.Clone(parameters),
		resultType:  resultType,
		fingerprint: hex.EncodeToString(digest.Sum(nil)),
	}, nil
}

func (c Callable) Valid() bool {
	return c.identity != "" && c.resultType != "" && c.fingerprint != ""
}

func (c Callable) Identity() string        { return c.identity }
func (c Callable) Parameters() []Parameter { return slices.Clone(c.parameters) }
func (c Callable) ResultType() string      { return c.resultType }
func (c Callable) Fingerprint() string     { return c.fingerprint }
func (c Callable) CanonicalEncoding() []byte {
	if !c.Valid() {
		return nil
	}
	var encoded bytes.Buffer
	writeCanonicalString(&encoded, c.identity)
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(c.parameters)))
	encoded.Write(count[:])
	for _, parameter := range c.parameters {
		encoded.WriteByte(byte(parameter.projection))
		writeCanonicalString(&encoded, parameter.targetType)
	}
	writeCanonicalString(&encoded, c.resultType)
	return encoded.Bytes()
}
func (c Callable) Parameter(index int) (Parameter, bool) {
	if index < 0 || index >= len(c.parameters) {
		return Parameter{}, false
	}
	return c.parameters[index], true
}

func writeCanonicalString(target *bytes.Buffer, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	target.Write(length[:])
	target.WriteString(value)
}

type Error struct {
	Function string
	Reason   string
}

func (e *Error) Error() string {
	if e.Function == "" {
		return "callable ABI: " + e.Reason
	}
	return fmt.Sprintf("callable ABI %q: %s", e.Function, e.Reason)
}
