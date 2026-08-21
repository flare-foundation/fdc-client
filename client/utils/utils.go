package utils

import (
	"bytes"
)

// Prepend places the element at the beginning of the slice and moves the potentially replaced element to the end.
func Prepend[T any](slice []T, element T) []T {
	if len(slice) == 0 {
		slice = append(slice, element)
		return slice
	}

	slice = append(slice, slice[0])

	slice[0] = element

	return slice
}

// Invert returns a new map whose keys and values are swapped.
// If m has duplicate values, only one of the corresponding keys is preserved.
func Invert[K comparable, V comparable](m map[K]V) map[V]K {
	invertedMap := make(map[V]K)
	for k, v := range m {
		invertedMap[v] = k
	}

	return invertedMap
}

// Bytes32ToString returns b as a string with trailing zero bytes stripped.
func Bytes32ToString(b [32]byte) string {
	return string(bytes.Trim(b[:], "\x00"))
}
