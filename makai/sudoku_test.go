package makai

import (
	"testing"
)

func newSudokuUser(t *testing.T, ctx *MakaiContext) {
	uid, err := ctx.RegisterSudokuUser("heck", "somepassword")
	if err != nil {
		t.Fatalf("Can't create sudoku user: %s", err)
	}
	if uid <= 0 {
		t.Fatalf("Didn't return a valid uid: %d!", uid)
	}
	// Go lookup the user to make sure it worked
	user, err := ctx.GetSudokuUserById(uid)
	if err != nil {
		t.Fatalf("Can't find created sudoku user: %s", err)
	}
	if user.UID != uid {
		t.Fatalf("Same user not found (uid)!")
	}
	if user.Username != "heck" {
		t.Fatalf("Same user not found (username)!")
	}
}

func TestNewSudokuUser(t *testing.T) {
	ctx := newTestContext("newsudokuuser")
	newSudokuUser(t, ctx)
}

// import (
// 	"log"
// 	"testing"
// )

// func TestPasswordHashing(t *testing.T) {
// 	hash, err := passwordHash("mypassword123!")
// 	if err != nil {
// 		t.Fatalf("Error from password hashing: %s", err)
// 	}
// 	err = passwordVerify("mypassword123!", hash)
// 	if err != nil {
// 		t.Fatalf("Error, same password not verified: %s", err)
// 	}
// 	err = passwordVerify("mypassword123", hash)
// 	if err == nil {
// 		t.Fatalf("Error expected, but different password accepted")
// 	} else {
// 		log.Printf("Verify error (expected): %s", err)
// 	}
// }
