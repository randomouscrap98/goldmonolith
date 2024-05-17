package makai

import (
	"log"
	"testing"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func newSudokuUser(username string, password string, t *testing.T, ctx *MakaiContext) int64 {
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
	return uid
}

// Test MANY aspects of creating and using a sudoku user. these sudoku
// tests are expensive, so we do as much as possible in each.
func TestNewSudokuUser_FULL(t *testing.T) {
	ctx := newTestContext("newsudokuuser")
	exists, err := ctx.sudokuUserExists("heck")
	if err != nil {
		t.Fatalf("Error while checking if user exists: %s", err)
	}
	if exists {
		t.Fatalf("User was not supposed to exist yet!")
	}
	uid := newSudokuUser("heck", "somepassword", t, ctx)
	exists, err = ctx.sudokuUserExists("heck")
	if err != nil {
		t.Fatalf("Error while checking if user exists: %s", err)
	}
	if !exists {
		t.Fatalf("User was supposed to exist!")
	}
	// Now test login, since each sudoku test is expensive (we reuse an existing file db)
	goodtoken, err := ctx.LoginSudokuUser("heck", "somepassword")
	if err != nil {
		t.Fatalf("Error logging in user: %s", err)
	}
	if goodtoken == "" {
		t.Fatalf("Token not generated")
	}
	_, err = ctx.LoginSudokuUser("heck2", "somepassword")
	if err == nil {
		t.Fatalf("Expected SOME error from non-existent user")
	}
	_, ok := err.(*utils.ExpectedError)
	if !ok {
		t.Fatalf("Expected an expected error, got %s", err)
	}
	_, err = ctx.LoginSudokuUser("heck", "somepassword2")
	if err == nil {
		t.Fatalf("Expected SOME error from bad password")
	}
	_, ok = err.(*utils.ExpectedError)
	if !ok {
		t.Fatalf("Expected an expected error, got %s", err)
	}
	// Make sure the sudoku user is what we expect
	session, err := ctx.GetSudokuSession(goodtoken)
	if err != nil {
		t.Fatalf("Got error when decoding token: %s", err)
	}
	if session.UserId != uid {
		t.Fatalf("UID from sesion incorrect!")
	}
	origuser, err := ctx.GetSudokuUserById(session.UserId)
	if err != nil {
		t.Fatalf("Got error when looking up user from session: %s", err)
	}
	if origuser.Username != "heck" {
		t.Fatalf("Username from session incorrect!")
	}
	if origuser.UID != uid {
		t.Fatalf("UID from sesion incorrect!")
	}
	// Now we test conversion, this is very important!!
	fulluser := origuser.ToUser(true)
	if fulluser.Username != origuser.Username {
		t.Fatalf("Usernames didn't match: %s vs %s", fulluser.Username, origuser.Username)
	}
	if fulluser.Uid != origuser.UID {
		t.Fatalf("UIDS didn't match: %d vs %d", fulluser.Uid, origuser.UID)
	}
	defoptions := getDefaultSudokuOptions()
	if len(fulluser.Options) != len(defoptions) {
		t.Fatalf("Didn't have all options from defaults: %v", fulluser.Options)
	}
	if len(fulluser.JsonOptions) == 0 {
		t.Fatalf("Didn't set json options")
	}
	log.Printf("Sudoku user options json: %s", fulluser.JsonOptions)
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
