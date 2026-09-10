package providerstorage

import "reflect"

type Holder struct {
	Value reflect.Value
}

func replace[Element any](target *Element, value Element) {
	*target = value
}

func Descriptors() bool {
	original := reflect.ValueOf(1)
	copied := original
	pointer := &copied
	replace(pointer, reflect.ValueOf(2))
	if original.Int() != 1 || copied.Int() != 2 || pointer.Int() != 2 {
		return false
	}
	holder := Holder{Value: original}
	field := &holder.Value
	holder = Holder{Value: copied}
	if holder.Value.Int() != 2 || field.Int() != 2 {
		return false
	}
	array := [1]reflect.Value{original}
	entry := &array[0]
	array = [1]reflect.Value{copied}
	if entry.Int() != 2 || original.Int() != 1 {
		return false
	}
	*pointer = reflect.Value{}
	return !copied.IsValid() && original.Int() == 1 && entry.Int() == 2
}

func LiveLocations() bool {
	private := struct{ hidden int }{hidden: 1}
	privateField := reflect.ValueOf(&private).Elem().Field(0)
	if !privateField.CanAddr() || privateField.CanSet() {
		return false
	}
	number := 3
	original := reflect.ValueOf(&number).Elem()
	copied := original
	copied.SetInt(9)
	if number != 9 || original.Int() != 9 || copied.Int() != 9 {
		return false
	}
	replace(&copied, reflect.ValueOf(4))
	original.SetInt(7)
	return number == 7 && original.Int() == 7 && copied.Int() == 4 &&
		original.CanAddr() && !copied.CanAddr()
}

func MutationConditions() bool {
	number := 3
	change := func() bool { number = 7; return true }
	if number != 3 {
		return false
	}
	if !(number == 3 && change() && number == 7) {
		return false
	}
	flag := true
	flip := func() { flag = false }
	if !flag {
		return false
	}
	flip()
	if flag == true {
		return false
	}
	switch number {
	case 7:
		replace(&number, 9)
		return number == 9
	default:
		return false
	}
}
