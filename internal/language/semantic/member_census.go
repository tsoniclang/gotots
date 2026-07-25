package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
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

const memberTargetCensusBufferBytes = 32 * 1024

type memberTargetCensusWriter struct {
	digest hash.Hash
	buffer []byte
}

func newMemberTargetCensusWriter() *memberTargetCensusWriter {
	return &memberTargetCensusWriter{
		digest: sha256.New(),
		buffer: make([]byte, 0, memberTargetCensusBufferBytes),
	}
}

func (writer *memberTargetCensusWriter) flush() {
	if len(writer.buffer) == 0 {
		return
	}
	_, _ = writer.digest.Write(writer.buffer)
	writer.buffer = writer.buffer[:0]
}

func (writer *memberTargetCensusWriter) reserve(size int) {
	if size <= cap(writer.buffer)-len(writer.buffer) {
		return
	}
	writer.flush()
	if size > cap(writer.buffer) {
		writer.buffer = make([]byte, 0, size)
	}
}

func (writer *memberTargetCensusWriter) writePart(value string) {
	writer.reserve(len(value) + 24)
	writer.buffer = strconv.AppendInt(
		writer.buffer,
		int64(len(value)),
		10,
	)
	writer.buffer = append(writer.buffer, ':')
	writer.buffer = append(writer.buffer, value...)
	writer.buffer = append(writer.buffer, '|')
}

func (writer *memberTargetCensusWriter) writeBytes(value []byte) {
	writer.reserve(len(value) + 24)
	writer.buffer = strconv.AppendInt(
		writer.buffer,
		int64(len(value)),
		10,
	)
	writer.buffer = append(writer.buffer, ':')
	writer.buffer = append(writer.buffer, value...)
	writer.buffer = append(writer.buffer, '|')
}

func (writer *memberTargetCensusWriter) sum() string {
	writer.flush()
	return hex.EncodeToString(writer.digest.Sum(nil))
}

type memberMethodKey struct {
	pkg  packageRef
	name string
}

func (pkg Package) MemberTargetCensus() (MemberTargetCensus, error) {
	if pkg.memberTargets.count < 0 ||
		len(pkg.memberTargets.digest) != sha256.Size*2 {
		return MemberTargetCensus{}, fmt.Errorf(
			"semantic package has no sealed member-target census",
		)
	}
	return pkg.memberTargets, nil
}

func deriveMemberTargetCensus(
	pkg Package,
) (MemberTargetCensus, error) {
	writer := newMemberTargetCensusWriter()
	writeCensusPart(writer, "semantic-member-target-census/v2")
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
				writeFieldTarget(writer, pkg, owner, *field)
			case method != nil:
				writeMethodTarget(writer, pkg, owner, *method)
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
	writeCensusPart(writer, "count")
	writeCensusInt(writer, int64(count))
	return MemberTargetCensus{
		count:  count,
		digest: writer.sum(),
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
	writer *memberTargetCensusWriter,
	pkg Package,
	owner typeRef,
	field storedTypeField,
) {
	writeCensusPart(writer, "field")
	writeCensusType(writer, pkg, owner)
	writeCensusPackage(writer, pkg, field.pkg)
	writeCensusPart(writer, field.name)
	writeCensusInt(writer, int64(field.ordinal))
	writeCensusType(writer, pkg, field.typeID)
	writeCensusBool(writer, field.embedded)
	writeCensusPart(writer, field.tag)
}

func writeMethodTarget(
	writer *memberTargetCensusWriter,
	pkg Package,
	owner typeRef,
	method storedTypeMethod,
) {
	writeCensusPart(writer, "method")
	writeCensusType(writer, pkg, owner)
	writeCensusPackage(writer, pkg, method.pkg)
	writeCensusPart(writer, method.name)
	writeCensusInt(writer, int64(method.ordinal))
	writeCensusType(writer, pkg, method.signature)
}

func writeCensusType(
	writer *memberTargetCensusWriter,
	pkg Package,
	reference typeRef,
) {
	record, present := componentAt(pkg.identities.types, reference)
	if !present {
		writeCensusPart(writer, "")
		return
	}
	writeCensusPart(writer, record.digest)
}

func writeCensusPackage(
	writer *memberTargetCensusWriter,
	pkg Package,
	reference packageRef,
) {
	if reference == 0 {
		writeCensusBool(writer, false)
		return
	}
	writeCensusBool(writer, true)
	record := pkg.identities.packages[reference-1]
	owner := pkg.identities.owners[record.owner-1]
	writeCensusInt(writer, int64(owner.class))
	if owner.module == 0 {
		writeCensusPart(writer, "")
		writeCensusPart(writer, "")
	} else {
		module := pkg.identities.modules[owner.module-1]
		writeCensusPart(writer, module.path)
		writeCensusPart(writer, module.version)
	}
	writeCensusPart(writer, record.importPath)
}

func writeCensusPart(
	writer *memberTargetCensusWriter,
	value string,
) {
	writer.writePart(value)
}

func writeCensusInt(
	writer *memberTargetCensusWriter,
	value int64,
) {
	var encoded [32]byte
	part := strconv.AppendInt(encoded[:0], value, 10)
	writeCensusBytes(writer, part)
}

func writeCensusBool(
	writer *memberTargetCensusWriter,
	value bool,
) {
	if value {
		writeCensusPart(writer, "1")
		return
	}
	writeCensusPart(writer, "0")
}

func writeCensusBytes(
	writer *memberTargetCensusWriter,
	value []byte,
) {
	writer.writeBytes(value)
}
