package conversion

type StoredRecord struct {
	Count uint32
	Inner struct{ Count uint32 }
}

func StructFieldAddressSurvivesReplacement() uint32 {
	value := StoredRecord{Count: 3}
	value.Inner.Count = 5
	count := &value.Count
	nested := &value.Inner.Count
	copy := value
	replacement := StoredRecord{Count: 11}
	replacement.Inner.Count = 13
	value = replacement
	replacement.Count = 17
	return *count + *nested + copy.Count + copy.Inner.Count
}

type GenericStoredRecord[Element any] struct{ Value Element }

func GenericFieldAddressSurvivesReplacement() uint32 {
	value := GenericStoredRecord[uint32]{Value: 2}
	saved := &value.Value
	copy := value
	value = GenericStoredRecord[uint32]{Value: 7}
	return *saved + copy.Value
}
