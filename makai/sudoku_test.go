package makai

import (
	"log"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := passwordHash("mypassword123!")
	if err != nil {
		t.Fatalf("Error from password hashing: %s", err)
	}
	err = passwordVerify("mypassword123!", hash)
	if err != nil {
		t.Fatalf("Error, same password not verified: %s", err)
	}
	err = passwordVerify("mypassword123", hash)
	if err == nil {
		t.Fatalf("Error expected, but different password accepted")
	} else {
		log.Printf("Verify error (expected): %s", err)
	}
}
