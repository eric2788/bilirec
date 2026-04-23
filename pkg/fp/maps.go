package fp

func KeysWhere[K comparable, V any](m map[K]V, predicate func(K, V) bool) []K {
	if m == nil {
		return nil
	}
	out := make([]K, 0, len(m))
	for k, v := range m {
		if predicate(k, v) {
			out = append(out, k)
		}
	}
	return out
}

func ValuesWhere[K comparable, V any](m map[K]V, predicate func(K, V) bool) []V {
	if m == nil {
		return nil
	}
	out := make([]V, 0, len(m))
	for k, v := range m {
		if predicate(k, v) {
			out = append(out, v)
		}
	}
	return out
}

func FilterByKey[K comparable, V any](m map[K]V, predicate func(K) bool) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		if predicate(k) {
			out[k] = v
		}
	}
	return out
}

func FilterByValue[K comparable, V any](m map[K]V, predicate func(V) bool) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		if predicate(v) {
			out[k] = v
		}
	}
	return out
}
