// Hand port A5 — direct fields and subtraction. GoInt carrier per the
// accepted integer representation.
Len(): GoInt {
  return this.end - this.pos;
}
