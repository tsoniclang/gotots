// Hand port B4 — byte-observable string scanning: the byte-faithful
// carrier makes indexOf/slicing exactly Go's byte operations, so the
// scanner body stays source-shaped; the exception is the CARRIER
// decision, not per-line machinery.
scanString(jsxAttributeString: boolean): string {
  const quote = this.char();
  if (quote === CharCodeSingleQuote) {
    this.tokenFlags |= ast.TokenFlags.SingleQuote;
  }
  this.pos++;
  // Fast path for simple strings without escape sequences.
  const strLen = this.text.indexOf(String.fromCharCode(quote), this.pos) - this.pos;
  if (strLen === 0) {
    this.pos++;
    return "";
  }
  if (strLen > 0) {
    const str = this.text.substring(this.pos, this.pos + strLen);
    if (jsxAttributeString ||
      (str.indexOf("\\") < 0 && str.indexOf("\r") < 0 && str.indexOf("\n") < 0)) {
      this.pos += strLen + 1;
      return str;
    }
  }
  let sb = "";
  let start = this.pos;
  for (;;) {
    const ch = this.char();
    if (ch < 0) {
      sb += this.text.substring(start, this.pos);
      this.tokenFlags |= ast.TokenFlags.Unterminated;
      this.error(diagnostics.Unterminated_string_literal);
      break;
    }
    if (ch === quote) {
      sb += this.text.substring(start, this.pos);
      this.pos++;
      break;
    }
    if (ch === CharCodeBackslash && !jsxAttributeString) {
      sb += this.text.substring(start, this.pos);
      sb += this.scanEscapeSequence(EscapeSequenceScanningFlags.String | EscapeSequenceScanningFlags.ReportErrors);
      start = this.pos;
      continue;
    }
    if ((ch === CharCodeLineFeed || ch === CharCodeCarriageReturn) && !jsxAttributeString) {
      sb += this.text.substring(start, this.pos);
      this.tokenFlags |= ast.TokenFlags.Unterminated;
      this.error(diagnostics.Unterminated_string_literal);
      break;
    }
    this.pos++;
  }
  return sb;
}
