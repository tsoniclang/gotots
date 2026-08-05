package runtimecontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
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
	id      uint16
	export  string
	carrier PrimitiveCarrier
}

func (e Entry) ID() uint16 {
	return e.id
}

func (e Entry) Export() string {
	return e.export
}

func (e Entry) ProviderCarrier() PrimitiveCarrier {
	return e.carrier
}

type PrimitiveCarrier uint8

const (
	PrimitiveCarrierInvalid PrimitiveCarrier = iota
	PrimitiveCarrierBoolean
	PrimitiveCarrierNumber
	PrimitiveCarrierBigInt
	PrimitiveCarrierString
)

func (c PrimitiveCarrier) Valid() bool {
	return c >= PrimitiveCarrierBoolean && c <= PrimitiveCarrierString
}

func (c PrimitiveCarrier) String() string {
	switch c {
	case PrimitiveCarrierBoolean:
		return "boolean"
	case PrimitiveCarrierNumber:
		return "number"
	case PrimitiveCarrierBigInt:
		return "bigint"
	case PrimitiveCarrierString:
		return "string"
	default:
		return fmt.Sprintf("primitive-carrier(%d)", c)
	}
}

type Requirements struct {
	profiles          []Profile
	providerProfile   Profile
	providerModule    string
	nativeIntegerBits uint8
	aliases           []Entry
	symbols           []Entry
	valid             bool
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

func (r Requirements) ProviderProfile() Profile {
	if !r.valid {
		return ProfileInvalid
	}
	return r.providerProfile
}

func (r Requirements) ProviderScalarModule() string {
	if !r.valid {
		return ""
	}
	return r.providerModule
}

func (r Requirements) NativeIntegerBits() uint8 {
	if !r.valid {
		return 0
	}
	return r.nativeIntegerBits
}

func (r Requirements) PrimitiveAliases() []Entry {
	return slices.Clone(r.aliases)
}

func (r Requirements) RuntimeSymbols() []Entry {
	return slices.Clone(r.symbols)
}

type entryDocument struct {
	ID              uint16 `json:"id"`
	Export          string `json:"export"`
	ProviderCarrier string `json:"providerCarrier,omitempty"`
}

type document struct {
	SchemaVersion                 uint8           `json:"schemaVersion"`
	IntegerRepresentations        []string        `json:"integerRepresentations"`
	ProviderIntegerRepresentation string          `json:"providerIntegerRepresentation"`
	ProviderScalarModule          string          `json:"providerScalarModule"`
	NativeIntegerBits             uint8           `json:"nativeIntegerBits"`
	PrimitiveAliases              []entryDocument `json:"primitiveAliases"`
	RuntimeSymbols                []entryDocument `json:"runtimeSymbols"`
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
	if source.SchemaVersion != 2 {
		return Requirements{}, contractError(fmt.Sprintf(
			"schema version %d is not 2",
			source.SchemaVersion,
		))
	}
	profiles, err := decodeProfiles(source.IntegerRepresentations)
	if err != nil {
		return Requirements{}, err
	}
	providerProfile, err := decodeProfile(source.ProviderIntegerRepresentation)
	if err != nil {
		return Requirements{}, contractError("provider " + err.Error())
	}
	if !slices.Contains(profiles, providerProfile) {
		return Requirements{}, contractError(
			"provider integer profile is not an admitted product profile",
		)
	}
	if !validProviderScalarModule(source.ProviderScalarModule) {
		return Requirements{}, contractError(fmt.Sprintf(
			"provider scalar module %q is invalid",
			source.ProviderScalarModule,
		))
	}
	if source.NativeIntegerBits != 32 && source.NativeIntegerBits != 64 {
		return Requirements{}, contractError(fmt.Sprintf(
			"native integer width %d is invalid",
			source.NativeIntegerBits,
		))
	}
	aliases, err := decodeEntries("primitive alias", source.PrimitiveAliases, true)
	if err != nil {
		return Requirements{}, err
	}
	symbols, err := decodeEntries("runtime symbol", source.RuntimeSymbols, false)
	if err != nil {
		return Requirements{}, err
	}
	if len(aliases) == 0 && len(symbols) == 0 {
		return Requirements{}, contractError("runtime surface is empty")
	}
	return Requirements{
		profiles:          profiles,
		providerProfile:   providerProfile,
		providerModule:    source.ProviderScalarModule,
		nativeIntegerBits: source.NativeIntegerBits,
		aliases:           aliases,
		symbols:           symbols,
		valid:             true,
	}, nil
}

func validProviderScalarModule(source string) bool {
	return strings.HasPrefix(source, "./internal/") &&
		strings.HasSuffix(source, ".js") &&
		!strings.Contains(source, "\\") &&
		!strings.Contains(source, "/../") &&
		!strings.Contains(source, "/./")
}

func decodeProfiles(source []string) ([]Profile, error) {
	profiles := make([]Profile, 0, len(source))
	for _, spelling := range source {
		profile, err := decodeProfile(spelling)
		if err != nil {
			return nil, contractError(err.Error())
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

func decodeProfile(spelling string) (Profile, error) {
	switch spelling {
	case ProfileNumber.String():
		return ProfileNumber, nil
	case ProfileBigInt.String():
		return ProfileBigInt, nil
	default:
		return ProfileInvalid, fmt.Errorf(
			"integer profile %q is invalid",
			spelling,
		)
	}
}

func decodeEntries(
	kind string,
	source []entryDocument,
	requireCarrier bool,
) ([]Entry, error) {
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
		carrier, err := decodePrimitiveCarrier(item.ProviderCarrier)
		if requireCarrier && err != nil {
			return nil, contractError(kind + " " + err.Error())
		}
		if !requireCarrier && item.ProviderCarrier != "" {
			return nil, contractError(kind + " must not declare a provider carrier")
		}
		entries = append(entries, Entry{
			id:      item.ID,
			export:  item.Export,
			carrier: carrier,
		})
	}
	slices.SortFunc(entries, func(left, right Entry) int {
		return int(left.id) - int(right.id)
	})
	return entries, nil
}

func decodePrimitiveCarrier(spelling string) (PrimitiveCarrier, error) {
	for _, carrier := range []PrimitiveCarrier{
		PrimitiveCarrierBoolean,
		PrimitiveCarrierNumber,
		PrimitiveCarrierBigInt,
		PrimitiveCarrierString,
	} {
		if carrier.String() == spelling {
			return carrier, nil
		}
	}
	return PrimitiveCarrierInvalid, fmt.Errorf(
		"provider carrier %q is invalid",
		spelling,
	)
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
