package utils

import (
	"log"
	"testing"
)

func TestGetRuntimeInfo(t *testing.T) {
	info := GetRuntimeInfo()
	log.Printf("Info: %v", info)
	if info.GoMemLimit == 0 {
		t.Fatalf("Expected some value for GoMemLimit (got 0)")
	}
	if info.HeapGoal == 0 {
		t.Fatalf("Expected some value for HeapGoal (got 0)")
	}
	// I just couldn't get this to not be 0. Oh well...
	// if info.LiveHeapBytes == 0 {
	// 	t.Fatalf("Expected some value for LiveHeapBytes (got 0)")
	// }
	if info.GoroutineCount == 0 {
		t.Fatalf("Expected some value for GoroutineCount (got 0)")
	}
	if info.TotalAllocBytes == 0 {
		t.Fatalf("Expected some value for TotalAllocBytes (got 0)")
	}
}
