// Hand port B8 — concrete-to-external-interface conversion: ONE boxing
// at the io.Writer boundary via the member-owned constructor; every
// concrete Builder call stays direct.
Base64DataURL(): string {
  const prefix = "data:application/json;base64,";
  const data = this.bytes();
  const sb = new strings.Builder();
  sb.Grow(prefix.length + base64.StdEncoding.EncodedLen(data.length));
  sb.WriteString(prefix);
  const encoder = base64.NewEncoder(base64.StdEncoding, strings.Builder$box$ioWriter(sb));
  encoder.Write(data);
  encoder.Close();
  return sb.String();
}
