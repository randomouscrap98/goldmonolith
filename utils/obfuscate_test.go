package utils

import (
	"testing"
)

func TestObfuscateSimple(t *testing.T) {
	o := GetDefaultObfuscation()

	// First, make sure you get an error if there's no key
	item, err := o.GetFromObfuscatedKey("none")
	if err == nil {
		t.Fatalf("Was supposed to get an error for none, did not")
	}

	// Now add an item
	key := o.GetObfuscatedKey("wow")

	// This should still fail
	item, err = o.GetFromObfuscatedKey("none")
	if err == nil {
		t.Fatalf("Was supposed to get an error for none, did not")
	}

	// But this will pass
	item, err = o.GetFromObfuscatedKey(key)
	if err != nil {
		t.Fatalf("Was not supposed to get an error for %s, got %s", key, err)
	}
	if item != "wow" {
		t.Fatalf("Item didn't match what was put in")
	}
	if key == "wow" {
		t.Fatalf("Key was supposed to be obfuscated, was the same as wow!")
	}

	// Add another item, same deal
	key2 := o.GetObfuscatedKey("wow2")

	// This should still fail
	item, err = o.GetFromObfuscatedKey("none")
	if err == nil {
		t.Fatalf("Was supposed to get an error for none, did not")
	}

	// But this will pass
	item, err = o.GetFromObfuscatedKey(key2)
	if err != nil {
		t.Fatalf("Was not supposed to get an error for %s, got %s", key2, err)
	}
	if item != "wow2" {
		t.Fatalf("Item didn't match what was put in")
	}
	if key2 == "wow" || key2 == "wow2" {
		t.Fatalf("Key was supposed to be obfuscated, was the same as wow!")
	}

	if key == key2 {
		t.Fatalf("Keys weren't supposed to match! Were %s and %s", key, key2)
	}

	// Adding the same item again should produce the same key
	key3 := o.GetObfuscatedKey("wow")
	if key3 != key {
		t.Fatalf("Supposed to get the same key each time! Got %s vs %s", key3, key)
	}
}
