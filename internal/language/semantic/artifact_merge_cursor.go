package semantic

import "fmt"

type orderedSemanticKey[Key any] interface {
	Compare(Key) int
}

type binaryRecordCursor[
	Record any,
	Key orderedSemanticKey[Key],
] struct {
	decoder     *binaryShardDecoder
	authority   Authority
	name        string
	expected    int
	decode      func(*binaryShardDecoder, Authority) (Record, error)
	key         func(Record) Key
	count       int
	previous    Key
	hasPrevious bool
	current     Record
	currentKey  Key
	present     bool
}

func openBinaryRecordCursor[
	Record any,
	Key orderedSemanticKey[Key],
](
	decoder *binaryShardDecoder,
	authority Authority,
	name string,
	expected int,
	decode func(*binaryShardDecoder, Authority) (Record, error),
	key func(Record) Key,
) (*binaryRecordCursor[Record, Key], error) {
	if decoder == nil || !authority.Valid() || expected < 0 ||
		decode == nil || key == nil {
		return nil, fmt.Errorf(
			"semantic binary record cursor %s is invalid", name,
		)
	}
	if _, err := readExpectedRecordCount(
		decoder, name, expected,
	); err != nil {
		return nil, err
	}
	cursor := &binaryRecordCursor[Record, Key]{
		decoder: decoder, authority: authority,
		name: name, expected: expected,
		decode: decode, key: key,
	}
	if err := cursor.advance(); err != nil {
		return nil, err
	}
	return cursor, nil
}

func (cursor *binaryRecordCursor[Record, Key]) advance() error {
	if cursor.count == cursor.expected {
		cursor.present = false
		var zeroKey Key
		cursor.currentKey = zeroKey
		var zeroRecord Record
		cursor.current = zeroRecord
		return nil
	}
	record, err := cursor.decode(cursor.decoder, cursor.authority)
	if err != nil {
		return fmt.Errorf(
			"decode semantic binary %s record %d: %w",
			cursor.name,
			cursor.count,
			err,
		)
	}
	key := cursor.key(record)
	if cursor.hasPrevious && key.Compare(cursor.previous) <= 0 {
		return fmt.Errorf(
			"semantic binary %s records are not canonical at %v",
			cursor.name,
			key,
		)
	}
	cursor.current = record
	cursor.currentKey = key
	cursor.previous = key
	cursor.hasPrevious = true
	cursor.present = true
	cursor.count++
	return nil
}

func mergeBinaryRecords[
	Record any,
	Key orderedSemanticKey[Key],
](
	checker *binaryRecordCursor[Record, Key],
	provider *binaryRecordCursor[Record, Key],
	equal func(Record, Record) bool,
	requireChecker func(Record) bool,
	admit func(Record, Authority) error,
) error {
	for checker.present || provider.present {
		order := 0
		if checker.present && provider.present {
			order = checker.currentKey.Compare(provider.currentKey)
		}
		switch {
		case checker.present && (!provider.present || order < 0):
			return fmt.Errorf(
				"provider authority omits checker %s %v",
				checker.name,
				checker.currentKey,
			)
		case provider.present && (!checker.present || order > 0):
			if requireChecker != nil &&
				requireChecker(provider.current) {
				return fmt.Errorf(
					"checker authority omits selected %s %v",
					provider.name,
					provider.currentKey,
				)
			}
			if err := admit(
				provider.current,
				provider.authority,
			); err != nil {
				return err
			}
			if err := provider.advance(); err != nil {
				return err
			}
		default:
			if !equal(checker.current, provider.current) {
				return fmt.Errorf(
					"semantic authorities conflict on %s %v",
					checker.name,
					checker.currentKey,
				)
			}
			if err := admit(
				checker.current,
				checker.authority,
			); err != nil {
				return err
			}
			if err := checker.advance(); err != nil {
				return err
			}
			if err := provider.advance(); err != nil {
				return err
			}
		}
	}
	return nil
}
