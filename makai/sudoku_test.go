package makai

import (
	"testing"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func newSudokuUser(username string, password string, t *testing.T, ctx *MakaiContext) {
	uid, err := ctx.RegisterSudokuUser(username, password)
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
	if user.Username != username {
		t.Fatalf("Same user not found (username)!")
	}
	if time.Now().Sub(user.Created).Seconds() > 1 {
		t.Fatalf("Create date not close enough: %s!", user.Created)
	}
}

func TestNewSudokuUser(t *testing.T) {
	ctx := newTestContext("newsudokuuser")
	exists, err := ctx.sudokuUserExists("heck")
	if err != nil {
		t.Fatalf("Error while checking if user exists: %s", err)
	}
	if exists {
		t.Fatalf("User was not supposed to exist yet!")
	}
	newSudokuUser("heck", "somepassword", t, ctx)
	exists, err = ctx.sudokuUserExists("heck")
	if err != nil {
		t.Fatalf("Error while checking if user exists: %s", err)
	}
	if !exists {
		t.Fatalf("User was supposed to exist!")
	}
	// Now test login, since each sudoku test is expensive (we reuse an existing file db)
	token, err := ctx.LoginSudokuUser("heck", "somepassword")
	if err != nil {
		t.Fatalf("Error logging in user: %s", err)
	}
	if token == "" {
		t.Fatalf("Token not generated")
	}
	token, err = ctx.LoginSudokuUser("heck2", "somepassword")
	if err == nil {
		t.Fatalf("Expected SOME error from non-existent user")
	}
	_, ok := err.(*utils.ExpectedError)
	if !ok {
		t.Fatalf("Expected an expected error, got %s", err)
	}
	token, err = ctx.LoginSudokuUser("heck", "somepassword2")
	if err == nil {
		t.Fatalf("Expected SOME error from bad password")
	}
	_, ok = err.(*utils.ExpectedError)
	if !ok {
		t.Fatalf("Expected an expected error, got %s", err)
	}
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
