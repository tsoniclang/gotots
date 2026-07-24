package structure

import "fmt"

type occurrenceDisplayRecord struct {
	startFile displayFileIndex
	endFile   displayFileIndex

	startLine   int
	startColumn int
	endLine     int
	endColumn   int
}

func (builder *OccurrenceStoreBuilder) displaySpan(
	display DisplaySpan,
	physical Span,
) (occurrenceDisplayIndex, error) {
	if builder == nil || builder.store == nil {
		return 0, fmt.Errorf(
			"occurrence display span requires mutable storage",
		)
	}
	if display.Start.Filename == builder.store.displayFile &&
		display.End.Filename == builder.store.displayFile &&
		display.Start.Line == physical.Start.Line &&
		display.Start.Column == physical.Start.Column &&
		display.End.Line == physical.End.Line &&
		display.End.Column == physical.End.Column {
		return 0, nil
	}
	startFile, err := builder.displayFile(
		display.Start.Filename,
	)
	if err != nil {
		return 0, err
	}
	endFile, err := builder.displayFile(
		display.End.Filename,
	)
	if err != nil {
		return 0, err
	}
	if uint64(len(builder.store.displaySpans)) >=
		uint64(^uint32(0)) {
		return 0, fmt.Errorf(
			"occurrence display-span table overflows uint32",
		)
	}
	builder.store.displaySpans = append(
		builder.store.displaySpans,
		occurrenceDisplayRecord{
			startFile:   startFile,
			endFile:     endFile,
			startLine:   display.Start.Line,
			startColumn: display.Start.Column,
			endLine:     display.End.Line,
			endColumn:   display.End.Column,
		},
	)
	return occurrenceDisplayIndex(
		len(builder.store.displaySpans),
	), nil
}

func (store *OccurrenceStore) displaySpan(
	record occurrenceStoreRecord,
) DisplaySpan {
	if store == nil {
		return DisplaySpan{}
	}
	if record.display == 0 {
		return DisplaySpan{
			Start: DisplayPosition{
				Filename: store.displayFile,
				Line:     record.startLine,
				Column:   record.startColumn,
			},
			End: DisplayPosition{
				Filename: store.displayFile,
				Line:     record.endLine,
				Column:   record.endColumn,
			},
		}
	}
	index := int(record.display - 1)
	if index < 0 ||
		index >= len(store.displaySpans) {
		return DisplaySpan{}
	}
	display := store.displaySpans[index]
	return DisplaySpan{
		Start: DisplayPosition{
			Filename: store.displayFileName(display.startFile),
			Line:     display.startLine,
			Column:   display.startColumn,
		},
		End: DisplayPosition{
			Filename: store.displayFileName(display.endFile),
			Line:     display.endLine,
			Column:   display.endColumn,
		},
	}
}
