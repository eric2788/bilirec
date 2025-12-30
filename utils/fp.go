package utils

import "strconv"

func NilOrElse[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

func Ptr[T any](v T) *T {
	return &v
}

func EmptyOrElse(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

func Ternary[T any](condition bool, trueValue, falseValue T) T {
	if condition {
		return trueValue
	} else {
		return falseValue
	}
}

func TernaryFunc[T any](condition bool, trueFunc, falseFunc func() T) T {
	if condition {
		return trueFunc()
	} else {
		return falseFunc()
	}
}

func MustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}
