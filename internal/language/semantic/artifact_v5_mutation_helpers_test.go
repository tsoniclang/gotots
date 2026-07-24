package semantic

import (
	"bytes"
	"encoding/json"
)

func reverseSemanticTypeIdentities(
	shard map[string]json.RawMessage,
) bool {
	identities, types, present := semanticTypeIdentityTable(shard)
	if !present || len(types) < 2 {
		return false
	}
	types[0], types[1] = types[1], types[0]
	return replaceSemanticIdentityTypes(shard, identities, types)
}

func duplicateSemanticTypeIdentity(
	shard map[string]json.RawMessage,
) bool {
	identities, types, present := semanticTypeIdentityTable(shard)
	if !present || len(types) < 2 {
		return false
	}
	types[1] = append(json.RawMessage(nil), types[0]...)
	return replaceSemanticIdentityTypes(shard, identities, types)
}

func appendUnreferencedSemanticTypeIdentity(
	shard map[string]json.RawMessage,
) bool {
	identities, types, present := semanticTypeIdentityTable(shard)
	if !present {
		return false
	}
	types = append(
		types,
		json.RawMessage(
			`{"digest":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}`,
		),
	)
	if !replaceSemanticIdentityTypes(
		shard,
		identities,
		types,
	) {
		return false
	}
	var counts map[string]json.RawMessage
	if json.Unmarshal(shard["counts"], &counts) != nil {
		return false
	}
	var current uint64
	if json.Unmarshal(counts["types"], &current) != nil {
		return false
	}
	encoded, err := json.Marshal(current + 1)
	if err != nil {
		return false
	}
	counts["types"] = encoded
	encoded, err = json.Marshal(counts)
	if err != nil {
		return false
	}
	shard["counts"] = encoded
	return true
}

func semanticTypeIdentityTable(
	shard map[string]json.RawMessage,
) (
	map[string]json.RawMessage,
	[]json.RawMessage,
	bool,
) {
	var identities map[string]json.RawMessage
	if json.Unmarshal(shard["identities"], &identities) != nil {
		return nil, nil, false
	}
	var types []json.RawMessage
	if json.Unmarshal(identities["types"], &types) != nil {
		return nil, nil, false
	}
	return identities, types, true
}

func replaceSemanticIdentityTypes(
	shard map[string]json.RawMessage,
	identities map[string]json.RawMessage,
	types []json.RawMessage,
) bool {
	encoded, err := json.Marshal(types)
	if err != nil {
		return false
	}
	identities["types"] = encoded
	order := [...]string{
		"modules",
		"owners",
		"packages",
		"files",
		"spans",
		"occurrences",
		"definitions",
		"types",
		"declarations",
		"bindings",
		"operations",
		"unsupported",
	}
	var output bytes.Buffer
	output.WriteByte('{')
	for index, name := range order {
		value, present := identities[name]
		if !present {
			return false
		}
		if index != 0 {
			output.WriteByte(',')
		}
		encodedName, marshalErr := json.Marshal(name)
		if marshalErr != nil {
			return false
		}
		output.Write(encodedName)
		output.WriteByte(':')
		output.Write(value)
	}
	output.WriteByte('}')
	shard["identities"] = output.Bytes()
	return true
}

func shiftCallableDeclarationRange(
	shard map[string]json.RawMessage,
) bool {
	var definitions []map[string]json.RawMessage
	if json.Unmarshal(shard["definitions"], &definitions) != nil {
		return false
	}
	for _, definition := range definitions {
		var payload map[string]json.RawMessage
		var callable map[string]json.RawMessage
		var declarations map[string]json.RawMessage
		if json.Unmarshal(definition["payload"], &payload) != nil ||
			json.Unmarshal(payload["callable"], &callable) != nil ||
			json.Unmarshal(
				callable["declarations"],
				&declarations,
			) != nil {
			continue
		}
		declarations["start"] = json.RawMessage("1")
		encoded, err := json.Marshal(declarations)
		if err != nil {
			return false
		}
		callable["declarations"] = encoded
		encoded, err = json.Marshal(callable)
		if err != nil {
			return false
		}
		payload["callable"] = encoded
		encoded, err = json.Marshal(payload)
		if err != nil {
			return false
		}
		definition["payload"] = encoded
		encoded, err = json.Marshal(definitions)
		if err != nil {
			return false
		}
		shard["definitions"] = encoded
		return true
	}
	return false
}

func changeSignaturePayloadTag(
	shard map[string]json.RawMessage,
) bool {
	return mutateSignatureTypePayload(
		shard,
		func(
			payload map[string]json.RawMessage,
		) bool {
			payload["tag"] = json.RawMessage("1")
			return true
		},
	)
}

func addInactiveTypePayload(
	shard map[string]json.RawMessage,
) bool {
	return mutateSignatureTypePayload(
		shard,
		func(
			payload map[string]json.RawMessage,
		) bool {
			payload["basic"] = json.RawMessage(`{"kind":2}`)
			return true
		},
	)
}

func removeActiveTypePayload(
	shard map[string]json.RawMessage,
) bool {
	return mutateSignatureTypePayload(
		shard,
		func(
			payload map[string]json.RawMessage,
		) bool {
			delete(payload, "signature")
			return true
		},
	)
}

func mutateSignatureTypePayload(
	shard map[string]json.RawMessage,
	mutate func(map[string]json.RawMessage) bool,
) bool {
	var records []map[string]json.RawMessage
	if json.Unmarshal(shard["types"], &records) != nil {
		return false
	}
	for _, record := range records {
		var kind uint8
		var payload map[string]json.RawMessage
		if json.Unmarshal(record["kind"], &kind) != nil ||
			TypeKind(kind) != TypeSignature ||
			json.Unmarshal(record["payload"], &payload) != nil ||
			!mutate(payload) {
			continue
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		record["payload"] = encoded
		encoded, err = json.Marshal(records)
		if err != nil {
			return false
		}
		shard["types"] = encoded
		return true
	}
	return false
}

func addRenderedDefinitionIdentity(
	shard map[string]json.RawMessage,
) bool {
	var definitions []map[string]json.RawMessage
	if json.Unmarshal(shard["definitions"], &definitions) != nil ||
		len(definitions) == 0 {
		return false
	}
	definitions[0]["definition"] = json.RawMessage(`"legacy-full-id"`)
	encoded, err := json.Marshal(definitions)
	if err != nil {
		return false
	}
	shard["definitions"] = encoded
	return true
}
