import { Value } from "../../reflect.js";

class InvalidValue extends Value {
  constructor() {
    super();
  }
}

export class ReflectValueOperations {
  static $zero(): Value {
    return new InvalidValue();
  }

  static $copy(source: Value): Value {
    return source;
  }
}
