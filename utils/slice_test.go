package utils

import (
	"slices"
	"testing"
)

func TestSliceToPlaceholder(t *testing.T) {
	placeholder := SliceToPlaceholder([]string{"a"})
	if placeholder != "?" {
		t.Fatalf("Expected ?, got %s", placeholder)
	}
	placeholder = SliceToPlaceholder([]string{"a", "b", "c"})
	if placeholder != "?,?,?" {
		t.Fatalf("Expected ?,?,?, got %s", placeholder)
	}
}

func TestSliceToAny(t *testing.T) {
	arr := SliceToAny([]string{"abc 123"})
	if len(arr) != 1 {
		t.Fatalf("Expected length 1, got %d", len(arr))
	}
	v, ok := arr[0].(string)
	if !ok {
		t.Fatalf("Underlying value wasn't string in any list")
	}
	if v != "abc 123" {
		t.Fatalf("Underlying value wasn't expected: %s vs abc 123", v)
	}
	arr = SliceToAny([]string{"abc", "123", "heck"})
	if len(arr) != 3 {
		t.Fatalf("Expected length 3, got %d", len(arr))
	}
}

func TestSliceDistinct(t *testing.T) {
	original := []int{1, 3, 2, 5, 3, 4, 5, 5, 5, 10}
	expected := []int{1, 2, 3, 4, 5, 10}
	distinct := SliceDistinct(original)
	slices.Sort(distinct)
	if !slices.Equal(distinct, expected) {
		t.Fatalf("Unexpected distinct list: %v", distinct)
	}
	soriginal := []string{"apple", "banana", "apple", "orange", "banana", "banana", "apple"}
	sexpected := []string{"apple", "banana", "orange"}
	sdistinct := SliceDistinct(soriginal)
	slices.Sort(sdistinct)
	if !slices.Equal(sdistinct, sexpected) {
		t.Fatalf("Unexpected string distinct list: %v", sdistinct)
	}
}
