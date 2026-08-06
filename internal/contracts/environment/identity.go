package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/types"
	"strconv"
	"strings"
)

func ToolchainKey(profile BuildProfile) (string, error) {
	if !profile.Valid() {
		return "", &ContractError{Reason: "toolchain identity is incomplete"}
	}
	cgo := "0"
	if profile.CgoEnabled() {
		cgo = "1"
	}
	digest := sha256.Sum256([]byte(
		profile.ToolchainVersion() + "\x00" +
			profile.GOOS() + "\x00" +
			profile.GOARCH() + "\x00" +
			cgo + "\x00" +
			strings.Join(profile.Tags(), "\x00"),
	))
	return hex.EncodeToString(digest[:]), nil
}

type ObjectKind uint8

const (
	ObjectInvalid ObjectKind = iota
	ObjectConstant
	ObjectType
	ObjectVariable
	ObjectFunction
	ObjectBuiltin
)

func (k ObjectKind) Valid() bool {
	return k >= ObjectConstant && k <= ObjectBuiltin
}

type ObjectContract struct {
	kind      ObjectKind
	identity  string
	receiver  string
	signature string
	value     string
}

func (c ObjectContract) Kind() ObjectKind {
	return c.kind
}

func (c ObjectContract) Identity() string {
	return c.identity
}

func (c ObjectContract) Receiver() string {
	return c.receiver
}

func (c ObjectContract) Signature() string {
	return c.signature
}

func (c ObjectContract) Value() string {
	return c.value
}

type ContractError struct {
	Object string
	Reason string
}

func (e *ContractError) Error() string {
	if e.Object == "" {
		return "describe environment declaration: " + e.Reason
	}
	return fmt.Sprintf(
		"describe environment declaration %q: %s",
		e.Object,
		e.Reason,
	)
}

func Describe(object types.Object) (ObjectContract, error) {
	if object == nil {
		return ObjectContract{}, &ContractError{Reason: "object is nil"}
	}
	if object.Pkg() == nil || object.Pkg().Path() == "" {
		return ObjectContract{}, &ContractError{
			Object: object.Name(),
			Reason: "object has no package identity",
		}
	}
	kind, err := objectKind(object)
	if err != nil {
		return ObjectContract{}, err
	}
	receiver, method := methodReceiver(object)
	if object.Parent() != object.Pkg().Scope() && !method {
		return ObjectContract{}, &ContractError{
			Object: object.Name(),
			Reason: "object is not a package declaration",
		}
	}
	signature, err := sourceSignature(object)
	if err != nil {
		return ObjectContract{}, err
	}
	value := ""
	if selected, ok := object.(*types.Const); ok && selected.Val() != nil {
		value = selected.Val().ExactString()
	}
	return ObjectContract{
		kind:      kind,
		identity:  Identity(object.Pkg().Path(), kind, receiver, object.Name()),
		receiver:  receiver,
		signature: signature,
		value:     value,
	}, nil
}

func Identity(
	packagePath string,
	kind ObjectKind,
	receiver string,
	name string,
) string {
	return packagePath + "|kind=" + strconv.Itoa(int(kind)) +
		"|receiver=" + receiver + "|name=" + name
}

func StableTypeString(source types.Type) string {
	return types.TypeString(source, func(sourcePackage *types.Package) string {
		if sourcePackage == nil {
			return ""
		}
		return sourcePackage.Path()
	})
}

func objectKind(object types.Object) (ObjectKind, error) {
	switch object.(type) {
	case *types.Const:
		return ObjectConstant, nil
	case *types.TypeName:
		return ObjectType, nil
	case *types.Var:
		return ObjectVariable, nil
	case *types.Func:
		return ObjectFunction, nil
	case *types.Builtin:
		return ObjectBuiltin, nil
	default:
		return ObjectInvalid, &ContractError{
			Object: object.Name(),
			Reason: "object kind is unsupported",
		}
	}
}

func methodReceiver(object types.Object) (string, bool) {
	function, ok := object.(*types.Func)
	if !ok {
		return "", false
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return "", false
	}
	receiverType := signature.Recv().Type()
	declaringType := receiverType
	if pointer, ok := types.Unalias(receiverType).(*types.Pointer); ok {
		declaringType = pointer.Elem()
	}
	named, ok := types.Unalias(declaringType).(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() != object.Pkg() {
		return "", false
	}
	return StableTypeString(receiverType), true
}

func sourceSignature(object types.Object) (string, error) {
	switch selected := object.(type) {
	case *types.Builtin:
		return "builtin", nil
	case *types.TypeName:
		return "defined=" + StableTypeString(selected.Type()) +
			"|underlying=" + StableTypeString(selected.Type().Underlying()), nil
	case *types.Func:
		signature, ok := selected.Type().(*types.Signature)
		if !ok {
			return "", &ContractError{
				Object: selected.Name(),
				Reason: "function has no signature",
			}
		}
		return StableTypeString(signature) +
			"|params=" + tupleNames(signature.Params()) +
			"|results=" + tupleNames(signature.Results()), nil
	default:
		if object.Type() == nil {
			return "", &ContractError{
				Object: object.Name(),
				Reason: "object has no type",
			}
		}
		return StableTypeString(object.Type()), nil
	}
}

func tupleNames(tuple *types.Tuple) string {
	if tuple == nil {
		return ""
	}
	names := make([]string, tuple.Len())
	for index := range tuple.Len() {
		names[index] = tuple.At(index).Name()
	}
	return strings.Join(names, ",")
}
