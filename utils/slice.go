package utils

import (
	"strings"
)

// Function to generate placeholders for SQL query
func SliceToPlaceholder[T any](slice []T) string {
	var sb strings.Builder
	ph := []byte("?,")
	phlast := []byte("?")
	for i := range slice {
		if i == len(slice)-1 {
			sb.Write(phlast)
		} else {
			sb.Write(ph)
		}
	}
	return sb.String()
}

// Convert a slice to an any slice, useful for db query params
func SliceToAny[T any](slice []T) []any {
	anys := make([]any, len(slice))
	for i := range anys {
		anys[i] = slice[i]
	}
	return anys
}

// A very slow and memory ineficient way to get the distinct
// set of items from a slice. Order is not preserved
func SliceDistinct[T comparable](slice []T) []T {
	set := make(map[T]bool)
	for _, item := range slice {
		set[item] = true
	}
	result := make([]T, len(set))
	i := 0
	for k := range set {
		result[i] = k
		i += 1
	}
	return result
}

// Get pointer to first element or throw error. Also accept an error
// because this is usually used with database lookups
func FirstErr[T any](t []T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	if len(t) < 1 {
		return nil, &NotFoundError{}
	}
	return &t[0], nil
}
