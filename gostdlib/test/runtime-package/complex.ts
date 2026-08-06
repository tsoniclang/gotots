export class GoComplex128 {
    declare private readonly goComplex128Brand: void;
    private constructor(public readonly real: number, public readonly imag: number) {
    }
    public static make(real: number, imag: number): GoComplex128 {
        return new GoComplex128(real, imag);
    }
}
