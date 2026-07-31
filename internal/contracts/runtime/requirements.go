package runtimecontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

type Profile uint8

const (
	ProfileInvalid Profile = iota
	ProfileNumber
	ProfileBigInt
)

func (p Profile) Valid() bool {
	return p == ProfileNumber || p == ProfileBigInt
}

func (p Profile) String() string {
	switch p {
	case ProfileNumber:
		return "number"
	case ProfileBigInt:
		return "bigint"
	default:
		return fmt.Sprintf("profile(%d)", p)
	}
}

type Entry struct {
	id     uint16
	export string
}

func (e Entry) ID() uint16 {
	return e.id
}

func (e Entry) Export() string {
	return e.export
}

type Requirements struct {
	profiles []Profile
	aliases  []Entry
	symbols  []Entry
	valid    bool
}

func (r Requirements) Valid() bool {
	return r.valid
}

func (r Requirements) AllowsProfile(profile Profile) bool {
	return r.valid && slices.Contains(r.profiles, profile)
}

func (r Requirements) Profiles() []Profile {
	return slices.Clone(r.profiles)
}

func (r Requirements) PrimitiveAliases() []Entry {
	return slices.Clone(r.aliases)
}

func (r Requirements) RuntimeSymbols() []Entry {
	return slices.Clone(r.symbols)
}

type entryDocument struct {
	ID     uint16 `json:"id"`
	Export string `json:"export"`
}

type document struct {
	SchemaVersion          uint8           `json:"schemaVersion"`
	IntegerRepresentations []string        `json:"integerRepresentations"`
	PrimitiveAliases       []entryDocument `json:"primitiveAliases"`
	RuntimeSymbols         []entryDocument `json:"runtimeSymbols"`
}

func Decode(data []byte) (Requirements, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var source document
	if err := decoder.Decode(&source); err != nil {
		return Requirements{}, contractError("decode: " + err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Requirements{}, contractError("decode: " + err.Error())
	}
	if source.SchemaVersion != 1 {
		return Requirements{}, contractError(fmt.Sprintf(
			"schema version %d is not 1",
			source.SchemaVersion,
		))
	}
	profiles, err := decodeProfiles(source.IntegerRepresentations)
	if err != nil {
		return Requirements{}, err
	}
	aliases, err := decodeEntries("primitive alias", source.PrimitiveAliases)
	if err != nil {
		return Requirements{}, err
	}
	symbols, err := decodeEntries("runtime symbol", source.RuntimeSymbols)
	if err != nil {
		return Requirements{}, err
	}
	if len(aliases) == 0 && len(symbols) == 0 {
		return Requirements{}, contractError("runtime surface is empty")
	}
	return Requirements{
		profiles: profiles,
		aliases:  aliases,
		symbols:  symbols,
		valid:    true,
	}, nil
}

func decodeProfiles(source []string) ([]Profile, error) {
	profiles := make([]Profile, 0, len(source))
	for _, spelling := range source {
		var profile Profile
		switch spelling {
		case ProfileNumber.String():
			profile = ProfileNumber
		case ProfileBigInt.String():
			profile = ProfileBigInt
		default:
			return nil, contractError(
				"integer profile " + fmt.Sprintf("%q", spelling) + " is invalid",
			)
		}
		if slices.Contains(profiles, profile) {
			return nil, contractError("integer profile " + spelling + " is duplicated")
		}
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return nil, contractError("integer profiles are empty")
	}
	slices.Sort(profiles)
	return profiles, nil
}

func decodeEntries(kind string, source []entryDocument) ([]Entry, error) {
	entries := make([]Entry, 0, len(source))
	ids := make(map[uint16]struct{}, len(source))
	exports := make(map[string]struct{}, len(source))
	for _, item := range source {
		if item.ID == 0 || item.Export == "" {
			return nil, contractError(kind + " identity is empty")
		}
		if _, duplicate := ids[item.ID]; duplicate {
			return nil, contractError(fmt.Sprintf(
				"%s id %d is duplicated",
				kind,
				item.ID,
			))
		}
		if _, duplicate := exports[item.Export]; duplicate {
			return nil, contractError(
				kind + " export " + fmt.Sprintf("%q", item.Export) + " is duplicated",
			)
		}
		ids[item.ID] = struct{}{}
		exports[item.Export] = struct{}{}
		entries = append(entries, Entry{id: item.ID, export: item.Export})
	}
	slices.SortFunc(entries, func(left, right Entry) int {
		return int(left.id) - int(right.id)
	})
	return entries, nil
}

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return "decode generated runtime contract: " + e.Reason
}

func contractError(reason string) error {
	return &Error{Reason: reason}
}
