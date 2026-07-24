package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strconv"

	"github.com/tsoniclang/gotots/internal/identity"
)

type MemberTargetCensus struct {
	count  int
	digest string
}

func (census MemberTargetCensus) Count() int {
	return census.count
}

func (census MemberTargetCensus) Digest() string {
	return census.digest
}

type memberMethodKey struct {
	pkg  packageRef
	name string
}

func (pkg Package) MemberTargetCensus() (MemberTargetCensus, error) {
	digest := sha256.New()
	writeCensusPart(digest, "semantic-member-target-census/v2")
	count := 0
	err := pkg.visitMemberTargets(
		func(
			owner typeRef,
			field *storedTypeField,
			method *storedTypeMethod,
		) error {
			count++
			switch {
			case field != nil:
				writeFieldTarget(digest, pkg, owner, *field)
			case method != nil:
				writeMethodTarget(digest, pkg, owner, *method)
			default:
				return fmt.Errorf(
					"semantic member target has no active payload",
				)
			}
			return nil
		},
	)
	if err != nil {
		return MemberTargetCensus{}, err
	}
	writeCensusPart(digest, "count")
	writeCensusInt(digest, int64(count))
	return MemberTargetCensus{
		count:  count,
		digest: hex.EncodeToString(digest.Sum(nil)),
	}, nil
}

func (pkg Package) visitMemberTargets(
	visit func(
		typeRef,
		*storedTypeField,
		*storedTypeMethod,
	) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic member visitor is required")
	}
	for _, owner := range pkg.types.records {
		if err := pkg.visitTypeMemberTargets(
			owner.id,
			owner,
			visit,
		); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) visitTypeMemberTargets(
	owner typeRef,
	current storedType,
	visit func(
		typeRef,
		*storedTypeField,
		*storedTypeMethod,
	) error,
) error {
	var seenMethods map[memberMethodKey]bool
	for depth := 0; depth <= len(pkg.types.records); depth++ {
		switch current.kind {
		case TypeNamed:
			nominal, err := payloadAt(
				pkg.types.nominal,
				current.payload,
			)
			if err != nil {
				return err
			}
			if nominal.methods.count != 0 &&
				seenMethods == nil {
				seenMethods = map[memberMethodKey]bool{}
			}
			if err := pkg.visitMethodRange(
				owner,
				nominal.methods,
				seenMethods,
				visit,
			); err != nil {
				return err
			}
			next, present := pkg.types.storedType(nominal.target)
			if !present {
				return fmt.Errorf(
					"semantic member owner %s has absent underlying type %s",
					pkg.identities.typeID(owner),
					pkg.identities.typeID(nominal.target),
				)
			}
			current = next
		case TypeStruct:
			relation, err := payloadAt(
				pkg.types.structs,
				current.payload,
			)
			if err != nil {
				return err
			}
			return pkg.visitFieldRange(owner, relation, visit)
		case TypeInterface:
			iface, err := payloadAt(
				pkg.types.interfaces,
				current.payload,
			)
			if err != nil {
				return err
			}
			return pkg.visitMethodRange(
				owner,
				iface.methods,
				seenMethods,
				visit,
			)
		default:
			return nil
		}
	}
	return fmt.Errorf(
		"semantic member owner %s has an underlying cycle",
		pkg.identities.typeID(owner),
	)
}

func (pkg Package) visitFieldRange(
	owner typeRef,
	relation typeFieldRange,
	visit func(
		typeRef,
		*storedTypeField,
		*storedTypeMethod,
	) error,
) error {
	fields, present := storedRelation(
		pkg.types.fields,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("semantic member field range is invalid")
	}
	for index := range fields {
		if err := visit(owner, &fields[index], nil); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) visitMethodRange(
	owner typeRef,
	relation typeMethodRange,
	seen map[memberMethodKey]bool,
	visit func(
		typeRef,
		*storedTypeField,
		*storedTypeMethod,
	) error,
) error {
	methods, present := storedRelation(
		pkg.types.methods,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("semantic member method range is invalid")
	}
	for index := range methods {
		method := &methods[index]
		key := memberMethodKey{
			pkg: method.pkg, name: method.name,
		}
		if seen != nil && seen[key] {
			id, _ := identity.NewMemberDeclarationID(
				pkg.identities.typeID(owner),
				pkg.identities.packageID(method.pkg),
				identity.SemanticObjectMethod,
				method.name,
				0,
			)
			return fmt.Errorf(
				"semantic package %s duplicates member target %s",
				pkg.id,
				id,
			)
		}
		if seen != nil {
			seen[key] = true
		}
		if err := visit(owner, nil, method); err != nil {
			return err
		}
	}
	return nil
}

func writeFieldTarget(
	digest hash.Hash,
	pkg Package,
	owner typeRef,
	field storedTypeField,
) {
	writeCensusPart(digest, "field")
	writeCensusType(digest, pkg, owner)
	writeCensusPackage(digest, pkg, field.pkg)
	writeCensusPart(digest, field.name)
	writeCensusInt(digest, int64(field.ordinal))
	writeCensusType(digest, pkg, field.typeID)
	writeCensusBool(digest, field.embedded)
	writeCensusPart(digest, field.tag)
}

func writeMethodTarget(
	digest hash.Hash,
	pkg Package,
	owner typeRef,
	method storedTypeMethod,
) {
	writeCensusPart(digest, "method")
	writeCensusType(digest, pkg, owner)
	writeCensusPackage(digest, pkg, method.pkg)
	writeCensusPart(digest, method.name)
	writeCensusInt(digest, int64(method.ordinal))
	writeCensusType(digest, pkg, method.signature)
}

func writeCensusType(
	digest hash.Hash,
	pkg Package,
	reference typeRef,
) {
	record, present := componentAt(pkg.identities.types, reference)
	if !present {
		writeCensusPart(digest, "")
		return
	}
	writeCensusPart(digest, record.digest)
}

func writeCensusPackage(
	digest hash.Hash,
	pkg Package,
	reference packageRef,
) {
	if reference == 0 {
		writeCensusBool(digest, false)
		return
	}
	writeCensusBool(digest, true)
	record := pkg.identities.packages[reference-1]
	owner := pkg.identities.owners[record.owner-1]
	writeCensusInt(digest, int64(owner.class))
	if owner.module == 0 {
		writeCensusPart(digest, "")
		writeCensusPart(digest, "")
	} else {
		module := pkg.identities.modules[owner.module-1]
		writeCensusPart(digest, module.path)
		writeCensusPart(digest, module.version)
	}
	writeCensusPart(digest, record.importPath)
}

func writeCensusPart(digest hash.Hash, value string) {
	var length [32]byte
	encoded := strconv.AppendInt(
		length[:0],
		int64(len(value)),
		10,
	)
	_, _ = digest.Write(encoded)
	_, _ = io.WriteString(digest, ":")
	_, _ = io.WriteString(digest, value)
	_, _ = io.WriteString(digest, "|")
}

func writeCensusInt(digest hash.Hash, value int64) {
	var encoded [32]byte
	part := strconv.AppendInt(encoded[:0], value, 10)
	writeCensusBytes(digest, part)
}

func writeCensusBool(digest hash.Hash, value bool) {
	if value {
		writeCensusPart(digest, "1")
		return
	}
	writeCensusPart(digest, "0")
}

func writeCensusBytes(digest hash.Hash, value []byte) {
	var length [32]byte
	encoded := strconv.AppendInt(
		length[:0],
		int64(len(value)),
		10,
	)
	_, _ = digest.Write(encoded)
	_, _ = io.WriteString(digest, ":")
	_, _ = digest.Write(value)
	_, _ = io.WriteString(digest, "|")
}
