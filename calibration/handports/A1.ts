// Hand port A1 — ordinary package/generic call chain. Names, receiver,
// argument counts, and control flow mirror the Go source exactly.
getSpellingSuggestionForUnicodePropertyValue(propertyName: string, value: string): string {
  const values = valuesOfNonBinaryUnicodeProperties.get(propertyName);
  if (values === undefined) {
    return "";
  }
  return core.GetSpellingSuggestionForStrings(value, maps.Keys(values.Keys()));
}
