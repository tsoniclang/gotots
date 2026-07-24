package compiler

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type mutableSemanticCounts struct {
	DeclarationRecords uint64 `json:"declarationRecords"`
}

type mutableSemanticIdentityTables struct {
	Declarations []mutableSemanticDeclarationIdentity `json:"declarations"`
	Types        []struct {
		Digest string `json:"digest"`
	} `json:"types"`
}

type mutableSemanticDeclarationIdentity struct {
	Form      uint8  `json:"form"`
	OwnerType uint64 `json:"ownerType"`
	MemberPkg uint64 `json:"memberPackage"`
	Class     uint8  `json:"class"`
	Name      string `json:"name"`
	Ordinal   int    `json:"ordinal"`
}

type mutableSemanticMemberRange struct {
	Start  uint64                       `json:"start"`
	Count  uint64                       `json:"count"`
	Values []map[string]json.RawMessage `json:"values"`
}

func removeReferencedSemanticDeclaration(
	shard map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	referenced := referencedDeclarationIndexes(shard["definitions"])
	var declarations []map[string]json.RawMessage
	if json.Unmarshal(shard["declarations"], &declarations) != nil {
		return false
	}
	for index, declaration := range declarations {
		var reference uint64
		if json.Unmarshal(declaration["id"], &reference) != nil ||
			!referenced[reference] {
			continue
		}
		declarations = append(
			declarations[:index], declarations[index+1:]...,
		)
		if !replaceSemanticShardField(
			shard, "declarations", declarations,
		) || !adjustDeclarationRecordCount(shard, -1) {
			return false
		}
		entry.DeclarationCount--
		if index < len(entry.Declarations) {
			entry.Declarations = append(
				entry.Declarations[:index],
				entry.Declarations[index+1:]...,
			)
		}
		return true
	}
	return false
}

func referencedDeclarationIndexes(
	encoded json.RawMessage,
) map[uint64]bool {
	var definitions []struct {
		Payload map[string]json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(encoded, &definitions) != nil {
		return nil
	}
	referenced := map[uint64]bool{}
	for _, definition := range definitions {
		for _, form := range []string{
			"callable", "initializer",
		} {
			var payload map[string]json.RawMessage
			if json.Unmarshal(
				definition.Payload[form], &payload,
			) != nil {
				continue
			}
			var values struct {
				Values []uint64 `json:"values"`
			}
			if json.Unmarshal(
				payload["declarations"], &values,
			) == nil {
				for _, reference := range values.Values {
					referenced[reference] = true
				}
			}
		}
		for _, form := range []string{"bodyless", "synthetic"} {
			var payload struct {
				Declaration uint64 `json:"declaration"`
			}
			if json.Unmarshal(
				definition.Payload[form], &payload,
			) == nil && payload.Declaration != 0 {
				referenced[payload.Declaration] = true
			}
		}
	}
	return referenced
}

func injectTargetSpecificSemanticField(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	if _, exists := shard["typescriptShape"]; exists {
		return false
	}
	shard["typescriptShape"] = json.RawMessage(`"class"`)
	return true
}

func exceedSemanticManifestCapacity(
	_ map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	entry.BindingCount = int(entry.ShardBytes) + 1
	return true
}

func duplicateSemanticResolutionWithoutCount(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	var records []json.RawMessage
	if json.Unmarshal(
		shard["resolutions"], &records,
	) != nil || len(records) == 0 {
		return false
	}
	records = append(records, records[0])
	return replaceSemanticShardField(shard, "resolutions", records)
}

func dropUnexportedMemberPackage(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	var records []map[string]json.RawMessage
	if json.Unmarshal(shard["types"], &records) != nil {
		return false
	}
	for _, record := range records {
		var payload map[string]json.RawMessage
		if json.Unmarshal(record["payload"], &payload) != nil {
			continue
		}
		if mutateUnexportedMemberPackage(payload) {
			encoded, err := json.Marshal(payload)
			if err != nil {
				return false
			}
			record["payload"] = encoded
			return replaceSemanticShardField(
				shard, "types", records,
			)
		}
	}
	return false
}

func mutateUnexportedMemberPackage(
	payload map[string]json.RawMessage,
) bool {
	for _, owner := range []string{"struct", "interface", "nominal"} {
		var shape map[string]json.RawMessage
		if json.Unmarshal(payload[owner], &shape) != nil {
			continue
		}
		for _, memberClass := range []string{"fields", "methods"} {
			var members mutableSemanticMemberRange
			if json.Unmarshal(
				shape[memberClass], &members,
			) != nil {
				continue
			}
			for _, member := range members.Values {
				var name string
				var packageReference uint64
				if json.Unmarshal(member["name"], &name) != nil ||
					json.Unmarshal(
						member["package"], &packageReference,
					) != nil ||
					packageReference == 0 ||
					semanticNameIsExported(name) {
					continue
				}
				member["package"] = json.RawMessage("0")
				encoded, err := json.Marshal(members)
				if err != nil {
					return false
				}
				shape[memberClass] = encoded
				encoded, err = json.Marshal(shape)
				if err != nil {
					return false
				}
				payload[owner] = encoded
				return true
			}
		}
	}
	return false
}

func introduceCircularInterfaceMethodIdentity(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	var records []map[string]json.RawMessage
	if json.Unmarshal(shard["types"], &records) != nil {
		return false
	}
	for _, record := range records {
		var kind uint8
		var reference uint64
		var payload map[string]json.RawMessage
		if json.Unmarshal(record["kind"], &kind) != nil ||
			semantic.TypeKind(kind) != semantic.TypeInterface ||
			json.Unmarshal(record["id"], &reference) != nil ||
			json.Unmarshal(record["payload"], &payload) != nil {
			continue
		}
		var shape map[string]json.RawMessage
		var methods mutableSemanticMemberRange
		if json.Unmarshal(payload["interface"], &shape) != nil ||
			json.Unmarshal(shape["methods"], &methods) != nil ||
			len(methods.Values) == 0 {
			continue
		}
		encodedReference, err := json.Marshal(reference)
		if err != nil {
			return false
		}
		methods.Values[0]["signature"] = encodedReference
		encodedMethods, err := json.Marshal(methods)
		if err != nil {
			return false
		}
		shape["methods"] = encodedMethods
		encodedShape, err := json.Marshal(shape)
		if err != nil {
			return false
		}
		payload["interface"] = encodedShape
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		record["payload"] = encodedPayload
		return replaceSemanticShardField(shard, "types", records)
	}
	return false
}

func serializeSemanticMemberDeclaration(
	shard map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	var identities mutableSemanticIdentityTables
	if json.Unmarshal(shard["identities"], &identities) != nil {
		return false
	}
	var types []map[string]json.RawMessage
	if json.Unmarshal(shard["types"], &types) != nil {
		return false
	}
	var packageReference uint64
	if json.Unmarshal(shard["package"], &packageReference) != nil {
		return false
	}
	for index, declaration := range identities.Declarations {
		if identity.SemanticDeclarationForm(declaration.Form) !=
			identity.SemanticDeclarationMember {
			continue
		}
		memberType, found := semanticMemberType(
			types, declaration,
		)
		if !found {
			continue
		}
		var records []json.RawMessage
		if json.Unmarshal(shard["declarations"], &records) != nil {
			return false
		}
		record, err := json.Marshal(map[string]any{
			"id":       uint64(index + 1),
			"package":  packageReference,
			"class":    declaration.Class,
			"name":     declaration.Name,
			"type":     memberType,
			"exported": semanticNameIsExported(declaration.Name),
		})
		if err != nil {
			return false
		}
		records = append(records, record)
		if !replaceSemanticShardField(
			shard, "declarations", records,
		) || !adjustDeclarationRecordCount(shard, 1) {
			return false
		}
		entry.DeclarationCount++
		if declaration.OwnerType == 0 ||
			declaration.OwnerType > uint64(len(identities.Types)) {
			return false
		}
		owner, err := identity.ParseSemanticTypeID(
			"semantic-type/sha256:" +
				identities.Types[declaration.OwnerType-1].Digest,
		)
		if err != nil {
			return false
		}
		memberPackage := identity.PackageID{}
		if declaration.MemberPkg != 0 {
			memberPackage, err = identity.ParsePackageID(entry.Package)
			if err != nil {
				return false
			}
		}
		memberID, err := identity.NewMemberDeclarationID(
			owner,
			memberPackage,
			identity.SemanticObjectClass(declaration.Class),
			declaration.Name,
			declaration.Ordinal,
		)
		if err != nil {
			return false
		}
		entry.Declarations = append(
			entry.Declarations, memberID.String(),
		)
		sort.Slice(
			entry.Declarations,
			func(left int, right int) bool {
				leftID, leftErr :=
					identity.ParseSemanticDeclarationID(
						entry.Declarations[left],
					)
				rightID, rightErr :=
					identity.ParseSemanticDeclarationID(
						entry.Declarations[right],
					)
				return leftErr == nil && rightErr == nil &&
					leftID.Compare(rightID) < 0
			},
		)
		return true
	}
	return false
}

func semanticMemberType(
	types []map[string]json.RawMessage,
	declaration mutableSemanticDeclarationIdentity,
) (uint64, bool) {
	for _, record := range types {
		var reference uint64
		var payload map[string]json.RawMessage
		if json.Unmarshal(record["id"], &reference) != nil ||
			reference != declaration.OwnerType ||
			json.Unmarshal(record["payload"], &payload) != nil {
			continue
		}
		for _, owner := range []string{
			"struct", "interface", "nominal",
		} {
			var shape map[string]json.RawMessage
			if json.Unmarshal(payload[owner], &shape) != nil {
				continue
			}
			memberClass := "methods"
			typeField := "signature"
			if identity.SemanticObjectClass(declaration.Class) ==
				identity.SemanticObjectField {
				memberClass = "fields"
				typeField = "type"
			}
			var members mutableSemanticMemberRange
			if json.Unmarshal(
				shape[memberClass], &members,
			) != nil {
				continue
			}
			for _, member := range members.Values {
				var name string
				var ordinal int
				var memberType uint64
				if json.Unmarshal(member["name"], &name) == nil &&
					json.Unmarshal(
						member["ordinal"], &ordinal,
					) == nil &&
					name == declaration.Name &&
					ordinal == declaration.Ordinal &&
					json.Unmarshal(
						member[typeField], &memberType,
					) == nil {
					return memberType, true
				}
			}
		}
	}
	return 0, false
}

func adjustDeclarationRecordCount(
	shard map[string]json.RawMessage,
	delta int,
) bool {
	var counts map[string]json.RawMessage
	if json.Unmarshal(shard["counts"], &counts) != nil {
		return false
	}
	var current uint64
	if json.Unmarshal(
		counts["declarationRecords"], &current,
	) != nil || delta < 0 && current < uint64(-delta) {
		return false
	}
	next := int64(current) + int64(delta)
	encoded, err := json.Marshal(next)
	if err != nil {
		return false
	}
	counts["declarationRecords"] = encoded
	return replaceSemanticShardField(shard, "counts", counts)
}

func replaceSemanticShardField(
	shard map[string]json.RawMessage,
	name string,
	value any,
) bool {
	encoded, err := json.Marshal(value)
	if err != nil {
		return false
	}
	shard[name] = encoded
	return true
}

func alterSemanticMemberTargetCount(
	_ map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	entry.MemberTargetCount++
	return true
}

func alterSemanticMemberTargetDigest(
	_ map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	entry.MemberTargetDigest = strings.Repeat("0", 64)
	return true
}

func semanticNameIsExported(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}
