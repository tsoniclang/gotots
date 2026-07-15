package census

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// The external declaration contract is the exact typed surface that owned
// source demands from every external package: the deterministic input for
// stub generation and the emulation layer. It is built from go/types
// export evidence only — external bodies are never read.

// ExternalObjectShape is one external object's exact contract.
type ExternalObjectShape struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // func|var|const|type|field
	// Signature for funcs; declared type for vars/consts/fields.
	Signature string `json:"signature,omitempty"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value,omitempty"` // consts: exact value
	// For kind=type: the underlying shape and the complete exported
	// method set (assignability at the boundary depends on it).
	Underlying string        `json:"underlying,omitempty"`
	TypeParams []string      `json:"typeParams,omitempty"`
	Methods    []MethodShape `json:"methods,omitempty"`
	// Referenced is true when owned source names this object directly;
	// closure entries exist only because a referenced contract needs them.
	Referenced bool `json:"referenced"`
}

// ExternalPackageContract is the complete demanded surface of one package.
type ExternalPackageContract struct {
	Path    string                `json:"path"`
	Objects []ExternalObjectShape `json:"objects"`
}

// ExternalContract is published as externals.json in the census bundle.
type ExternalContract struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Packages      []ExternalPackageContract `json:"packages"`
}

// buildExternalContract renders the exact contract for every directly
// referenced external object plus the transitive closure of external
// named types their contracts mention.
func buildExternalContract(referenced map[types.Object]bool, isExternal func(string) bool) (*ExternalContract, error) {
	closure := newTypeClosure(isExternal)
	for object := range referenced {
		closure.addObject(object)
	}
	if err := closure.solve(); err != nil {
		return nil, err
	}

	// Distinct types.Object instances can render to identical contracts
	// (the same object reached from several root packages); identical
	// renderings merge, and the ordering is total so equal names with
	// different contracts remain deterministic.
	byPackage := map[string]map[string]ExternalObjectShape{}
	emit := func(pkg string, shape ExternalObjectShape) {
		identity := shapeIdentity(shape)
		if byPackage[pkg] == nil {
			byPackage[pkg] = map[string]ExternalObjectShape{}
		}
		if existing, ok := byPackage[pkg][identity]; ok {
			existing.Referenced = existing.Referenced || shape.Referenced
			byPackage[pkg][identity] = existing
			return
		}
		byPackage[pkg][identity] = shape
	}
	for object := range referenced {
		shape, err := objectShape(object, true)
		if err != nil {
			return nil, err
		}
		emit(object.Pkg().Path(), shape)
	}
	for _, typeName := range closure.closureTypes() {
		if referenced[typeName] {
			continue
		}
		shape, err := objectShape(typeName, false)
		if err != nil {
			return nil, err
		}
		emit(typeName.Pkg().Path(), shape)
	}

	contract := &ExternalContract{SchemaVersion: 1}
	paths := make([]string, 0, len(byPackage))
	for path := range byPackage {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		identities := make([]string, 0, len(byPackage[path]))
		for identity := range byPackage[path] {
			identities = append(identities, identity)
		}
		sort.Strings(identities)
		objects := make([]ExternalObjectShape, 0, len(identities))
		for _, identity := range identities {
			objects = append(objects, byPackage[path][identity])
		}
		contract.Packages = append(contract.Packages, ExternalPackageContract{Path: path, Objects: objects})
	}
	return contract, nil
}

// shapeIdentity is the total ordering and merge key for one object shape:
// every contract-bearing field participates, so distinct contracts never
// collide and identical ones merge.
func shapeIdentity(shape ExternalObjectShape) string {
	parts := []string{shape.Name, shape.Kind, shape.Signature, shape.Type, shape.Value, shape.Underlying}
	for _, param := range shape.TypeParams {
		parts = append(parts, param)
	}
	for _, method := range shape.Methods {
		parts = append(parts, method.Name, method.Signature)
	}
	return strings.Join(parts, "\x00")
}

func objectShape(object types.Object, referenced bool) (ExternalObjectShape, error) {
	shape := ExternalObjectShape{Name: object.Name(), Referenced: referenced}
	switch concrete := object.(type) {
	case *types.Func:
		shape.Kind = "func"
		if qualified, isMethod := methodQualifiedName(concrete); isMethod {
			shape.Kind = "method"
			shape.Name = qualified
		}
		shape.Signature = typeString(concrete.Type())
	case *types.Var:
		if concrete.IsField() {
			shape.Kind = "field"
		} else {
			shape.Kind = "var"
		}
		shape.Type = typeString(concrete.Type())
	case *types.Const:
		shape.Kind = "const"
		shape.Type = typeString(concrete.Type())
		shape.Value = concrete.Val().ExactString()
	case *types.TypeName:
		shape.Kind = "type"
		named, ok := concrete.Type().(*types.Named)
		if !ok {
			// External aliases surface as their target type.
			shape.Underlying = typeString(types.Unalias(concrete.Type()))
			break
		}
		shape.Underlying = typeString(named.Underlying())
		if params := named.TypeParams(); params != nil {
			for i := range params.Len() {
				shape.TypeParams = append(shape.TypeParams,
					params.At(i).Obj().Name()+" "+typeString(params.At(i).Constraint()))
			}
		}
		for i := range named.NumMethods() {
			method := named.Method(i)
			if !method.Exported() {
				continue
			}
			signature := method.Type().(*types.Signature)
			pointerReceiver := false
			if recv := signature.Recv(); recv != nil {
				_, pointerReceiver = recv.Type().(*types.Pointer)
			}
			shape.Methods = append(shape.Methods, MethodShape{
				Name:            method.Name(),
				Signature:       typeString(signature),
				PointerReceiver: pointerReceiver,
			})
		}
	default:
		return shape, fmt.Errorf("unreviewed external object class %T for %s", object, object.Name())
	}
	return shape, nil
}

// typeClosure computes the transitive set of external named types
// mentioned by the referenced contracts.
type typeClosure struct {
	isExternal func(string) bool
	pending    []types.Type
	seenTypes  map[types.Type]bool
	named      map[*types.TypeName]bool
}

func newTypeClosure(isExternal func(string) bool) *typeClosure {
	return &typeClosure{
		isExternal: isExternal,
		seenTypes:  map[types.Type]bool{},
		named:      map[*types.TypeName]bool{},
	}
}

func (c *typeClosure) addObject(object types.Object) {
	c.pending = append(c.pending, object.Type())
}

func (c *typeClosure) addType(t types.Type) {
	if t != nil && !c.seenTypes[t] {
		c.seenTypes[t] = true
		c.pending = append(c.pending, t)
	}
}

// solve walks types breadth-first, collecting external named types. The
// walk is exhaustive over type shapes; an unreviewed shape fails.
func (c *typeClosure) solve() error {
	for len(c.pending) > 0 {
		current := c.pending[len(c.pending)-1]
		c.pending = c.pending[:len(c.pending)-1]

		switch t := types.Unalias(current).(type) {
		case *types.Named:
			obj := t.Obj()
			if obj.Pkg() != nil && c.isExternal(obj.Pkg().Path()) && !c.named[obj] {
				c.named[obj] = true
				c.addType(t.Underlying())
				for i := range t.NumMethods() {
					if t.Method(i).Exported() {
						c.addType(t.Method(i).Type())
					}
				}
			}
			if args := t.TypeArgs(); args != nil {
				for i := range args.Len() {
					c.addType(args.At(i))
				}
			}
		case *types.Pointer:
			c.addType(t.Elem())
		case *types.Slice:
			c.addType(t.Elem())
		case *types.Array:
			c.addType(t.Elem())
		case *types.Map:
			c.addType(t.Key())
			c.addType(t.Elem())
		case *types.Chan:
			c.addType(t.Elem())
		case *types.Signature:
			c.addType(t.Params())
			c.addType(t.Results())
		case *types.Tuple:
			for i := range t.Len() {
				c.addType(t.At(i).Type())
			}
		case *types.Struct:
			for i := range t.NumFields() {
				c.addType(t.Field(i).Type())
			}
		case *types.Interface:
			for i := range t.NumExplicitMethods() {
				c.addType(t.ExplicitMethod(i).Type())
			}
			for i := range t.NumEmbeddeds() {
				c.addType(t.EmbeddedType(i))
			}
		case *types.Union:
			for i := range t.Len() {
				c.addType(t.Term(i).Type())
			}
		case *types.TypeParam:
			c.addType(t.Constraint())
		case *types.Basic:
			// terminal
		default:
			return fmt.Errorf("unreviewed type shape %T in external closure (%s)", t, current)
		}
	}
	return nil
}

// closureTypes returns the collected external type names sorted by
// package and name for deterministic emission.
func (c *typeClosure) closureTypes() []*types.TypeName {
	names := make([]*types.TypeName, 0, len(c.named))
	for typeName := range c.named {
		names = append(names, typeName)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i].Pkg().Path() != names[j].Pkg().Path() {
			return names[i].Pkg().Path() < names[j].Pkg().Path()
		}
		return names[i].Name() < names[j].Name()
	})
	return names
}
