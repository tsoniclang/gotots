export class GoComplex128 {
  constructor(real, imag) {
    this.real = real;
    this.imag = imag;
  }

  static make(real, imag) {
    return new GoComplex128(real, imag);
  }
}
