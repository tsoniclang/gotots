package providerstorage

import (
	"reflect"
	"runtime"
	"runtime/metrics"
	"sync"
	"sync/atomic"
)

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

func LoopConditions() bool {
	type Entry struct{ Flag bool }
	values := []Entry{{true}, {false}}
	seen := false
	stable := true
	for _, value := range values {
		seen = seen || value.Flag
		stable = stable && value.Flag
	}
	return seen && !stable
}

func EmptyAssignments() bool {
	calls := 0
	create := func() struct{} { calls++; return struct{}{} }
	var value struct{}
	pointer := &value
	replace(pointer, create())
	values := make(map[int]struct{})
	values[1] = create()
	elements := []struct{}{{}}
	elements[0] = create()
	return *pointer == struct{}{} && len(values) == 1 && calls == 3
}

func SyncReset() bool {
	var values sync.Map
	retained := &values
	values.Store("key", 1)
	values = sync.Map{}
	if _, found := retained.Load("key"); found {
		return false
	}
	var pool sync.Pool
	poolPointer := &pool
	pool.Put(3)
	pool = sync.Pool{}
	if poolPointer.Get() != nil {
		return false
	}
	var once sync.Once
	oncePointer := &once
	count := 0
	once.Do(func() { count++ })
	once = sync.Once{}
	oncePointer.Do(func() { count++ })
	var group sync.WaitGroup
	groupPointer := &group
	group.Add(1)
	group = sync.WaitGroup{}
	groupPointer.Wait()
	var mutex sync.RWMutex
	mutexPointer := &mutex
	mutex.RLock()
	mutex = sync.RWMutex{}
	mutexPointer.Lock()
	mutexPointer.Unlock()
	return count == 2
}

func AtomicReset() bool {
	var boolean atomic.Bool
	booleanPointer := &boolean
	boolean.Store(true)
	boolean = atomic.Bool{}
	var signed32 atomic.Int32
	signed32Pointer := &signed32
	signed32.Store(3)
	signed32 = atomic.Int32{}
	var signed64 atomic.Int64
	signed64Pointer := &signed64
	signed64.Store(3)
	signed64 = atomic.Int64{}
	var unsigned32 atomic.Uint32
	unsigned32Pointer := &unsigned32
	unsigned32.Store(3)
	unsigned32 = atomic.Uint32{}
	var unsigned64 atomic.Uint64
	unsigned64Pointer := &unsigned64
	unsigned64.Add(3)
	unsigned64 = atomic.Uint64{}
	return !booleanPointer.Load() && signed32Pointer.Load() == 0 &&
		signed64Pointer.Load() == 0 && unsigned32Pointer.Load() == 0 && unsigned64Pointer.Load() == 0
}

func MemStatsFields() bool {
	var stats runtime.MemStats
	pauses := &stats.PauseNs
	first := &stats.PauseNs[0]
	entry := &stats.BySize[0]
	size := &stats.BySize[0].Size
	incoming := runtime.MemStats{}
	incoming.PauseNs[0] = 7
	incoming.BySize[0].Size = 9
	stats = incoming
	return pauses[0] == 7 && *first == 7 && entry.Size == 9 && *size == 9
}

func StructFields() bool {
	original := reflect.TypeOf(struct{ First int }{}).Field(0)
	name := &original.Name
	incoming := reflect.TypeOf(struct{ Second int }{}).Field(0)
	original = incoming
	return *name == "Second" && original.Index[0] == 0
}

func MetricsFields() bool {
	sample := metrics.Sample{Name: "before"}
	name := &sample.Name
	value := &sample.Value
	sample = metrics.Sample{Name: "after"}
	return *name == "after" && value.Kind() == 0
}
