package callableabi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

type Projection uint8

const (
	ProjectionInvalid Projection = iota
	ProjectionIdentity
	ProjectionPointeeValue
	ProjectionDirectObjectReference
	ProjectionMutableScalarLocation
	ProjectionOwnerLocation
	ProjectionUnsafeLocation
)

func (p Projection) Valid() bool {
	return p >= ProjectionIdentity && p <= ProjectionUnsafeLocation
}

func (p Projection) String() string {
	switch p {
	case ProjectionIdentity:
		return "identity"
	case ProjectionPointeeValue:
		return "pointee-value"
	case ProjectionDirectObjectReference:
		return "direct-object-reference"
	case ProjectionMutableScalarLocation:
		return "mutable-scalar-location"
	case ProjectionOwnerLocation:
		return "owner-location"
	case ProjectionUnsafeLocation:
		return "unsafe-location"
	default:
		return fmt.Sprintf("projection(%d)", p)
	}
}

type Parameter struct {
	projection Projection
	nilPolicy  NilPolicy
	targetType string
}

type NilPolicy uint8

const (
	NilPolicyInvalid NilPolicy = iota
	NilPolicyNotApplicable
	NilPolicyRejectAtBoundary
	NilPolicyPreserve
)

func (p NilPolicy) Valid() bool {
	return p >= NilPolicyNotApplicable && p <= NilPolicyPreserve
}

func NewParameter(
	projection Projection,
	nilPolicy NilPolicy,
	targetType string,
) (Parameter, error) {
	if !projection.Valid() || !nilPolicy.Valid() || targetType == "" ||
		(projection == ProjectionPointeeValue) ==
			(nilPolicy == NilPolicyNotApplicable) {
		return Parameter{}, &Error{Reason: "parameter projection is incomplete"}
	}
	return Parameter{
		projection: projection,
		nilPolicy:  nilPolicy,
		targetType: targetType,
	}, nil
}

func (p Parameter) Valid() bool {
	return p.projection.Valid() && p.nilPolicy.Valid() && p.targetType != "" &&
		(p.projection == ProjectionPointeeValue) !=
			(p.nilPolicy == NilPolicyNotApplicable)
}

func (p Parameter) Projection() Projection { return p.projection }
func (p Parameter) NilPolicy() NilPolicy   { return p.nilPolicy }
func (p Parameter) TargetType() string     { return p.targetType }

type Callable struct {
	identity    string
	parameters  []Parameter
	fingerprint string
}

func PackageFunctionIdentity(packagePath string, name string) (string, error) {
	if strings.TrimSpace(packagePath) != packagePath || packagePath == "" ||
		strings.TrimSpace(name) != name || name == "" {
		return "", &Error{Reason: "package-function identity is incomplete"}
	}
	return "package-function\x00" + packagePath + "\x00" + name, nil
}

func New(identity string, parameters []Parameter) (Callable, error) {
	if identity == "" {
		return Callable{}, &Error{Reason: "callable identity is absent"}
	}
	digest := sha256.New()
	digest.Write([]byte(identity))
	digest.Write([]byte{0})
	for index, parameter := range parameters {
		if !parameter.Valid() {
			return Callable{}, &Error{Function: identity, Reason: fmt.Sprintf("parameter %d is invalid", index)}
		}
		digest.Write([]byte{byte(parameter.projection), byte(parameter.nilPolicy), 0})
		digest.Write([]byte(parameter.targetType))
		digest.Write([]byte{0})
	}
	return Callable{
		identity:    identity,
		parameters:  slices.Clone(parameters),
		fingerprint: hex.EncodeToString(digest.Sum(nil)),
	}, nil
}

func (c Callable) Valid() bool {
	return c.identity != "" && c.fingerprint != ""
}

func (c Callable) Identity() string        { return c.identity }
func (c Callable) Parameters() []Parameter { return slices.Clone(c.parameters) }
func (c Callable) Fingerprint() string     { return c.fingerprint }
func (c Callable) Parameter(index int) (Parameter, bool) {
	if index < 0 || index >= len(c.parameters) {
		return Parameter{}, false
	}
	return c.parameters[index], true
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
