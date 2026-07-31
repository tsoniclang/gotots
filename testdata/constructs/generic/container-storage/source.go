package genericcontainerstorage

type Item struct {
	Value int32
}

type PlainItem struct {
	Value int32
}

type Bag[T any] struct {
	data []T
}

func (bag *Bag[T]) Add(value T) {
	bag.data = append(bag.data, value)
}

func (bag *Bag[T]) First() T {
	return bag.data[0]
}

type Arena[T any] struct {
	data []T
}

type ArenaHolder[T any] struct {
	value Arena[T]
}

func (arena *Arena[T]) Add(value T) *T {
	arena.data = append(arena.data, value)
	return &arena.data[len(arena.data)-1]
}

func (arena *Arena[T]) Size() int32 {
	return int32(len(arena.data))
}

func (arena *Arena[T]) Replace(index int, value T) {
	arena.data[index] = value
}

func ReuseArena[T any](target *Arena[T], source *Arena[T]) {
	target.data = source.data
}

func ReuseConcreteArena() int32 {
	holder := ArenaHolder[Item]{
		value: Arena[Item]{
			data: []Item{{Value: 13}},
		},
	}
	var target Arena[Item]
	ReuseArena(&target, &holder.value)
	return target.data[0].Value
}

func ArrayAddress[T any](first T, second T) T {
	values := [1]T{first}
	pointer := &values[0]
	values[0] = second
	return *pointer
}

func FirstVariadic[T any](values ...T) T {
	return values[0]
}

func Audit() []int32 {
	var bag Bag[PlainItem]
	bag.Add(PlainItem{Value: 7})

	var arena Arena[Item]
	item := arena.Add(Item{Value: 40})
	arena.Replace(0, Item{Value: 41})
	item.Value++

	var scalar Arena[int32]
	number := scalar.Add(40)
	scalar.Replace(0, 41)
	*number++

	return []int32{
		bag.First().Value,
		arena.Size(),
		arena.data[0].Value,
		scalar.Size(),
		scalar.data[0],
		ArrayAddress(
			Item{Value: 10},
			Item{Value: 11},
		).Value,
		ArrayAddress(int32(10), int32(11)),
		FirstVariadic(Item{Value: 12}).Value,
		ReuseConcreteArena(),
	}
}
