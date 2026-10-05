package when

// Keys is the set of keys an expression may read, with the kind each holds,
// for Check.
type Keys map[string]Kind

// Key is a typed, documented context key. A package that publishes context
// keys declares them as Keys, so their names, kinds and meaning live in one
// place.
type Key[T bool | int | float64 | string | []string] struct{ Name, Doc string }

// NewKey is a key named name.
func NewKey[T bool | int | float64 | string | []string](name, doc string) Key[T] {
	return Key[T]{Name: name, Doc: doc}
}

// Set stores v under k's name in m.
func (k Key[T]) Set(m Map, v T) {
	switch x := any(v).(type) {
	case bool:
		m[k.Name] = BoolValue(x)
	case int:
		m[k.Name] = NumberValue(float64(x))
	case float64:
		m[k.Name] = NumberValue(x)
	case string:
		m[k.Name] = StringValue(x)
	case []string:
		m[k.Name] = ListValue(x)
	}
}

// Get is k's value in c, and whether it is set and of k's kind. An int key
// holds a whole number.
func (k Key[T]) Get(c Context) (T, bool) {
	var zero T
	v, ok := c.Value(k.Name)
	if !ok || v.Kind() != k.Kind() {
		return zero, false
	}
	var out any
	switch any(zero).(type) {
	case bool:
		out = v.b
	case int:
		n := int(v.n)
		if float64(n) != v.n {
			return zero, false
		}
		out = n
	case float64:
		out = v.n
	case string:
		out = v.s
	case []string:
		l, _ := v.List()
		out = l
	}
	t, ok := out.(T)
	return t, ok
}

// Kind is the kind k's values have.
func (k Key[T]) Kind() Kind {
	var zero T
	switch any(zero).(type) {
	case bool:
		return KindBool
	case int, float64:
		return KindNumber
	case string:
		return KindString
	}
	return KindList
}
