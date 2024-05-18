package makai

import (
	"encoding/json"
	"log"
	"strings"
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
	log.Printf("(TEST) Sudoku user options json: %s", fulluser.JsonOptions)
	// Now we test user settings update
	newsettings := `{"somegarbage":"nothing"}`
	err = ctx.UpdateUserSettings(uid, newsettings)
	if err != nil {
		t.Fatalf("Couldn't update user settings: %s", err)
	}
	// Go lookup the user; our settings should be there verbatim
	user, err := ctx.GetSudokuUserById(uid)
	if err != nil {
		t.Fatalf("Error getting user by id: %s", err)
	}
	if user.SettingsJson != newsettings {
		t.Fatalf("User settings don't match: %s vs %s", user.SettingsJson, newsettings)
	}
	// Now even with garbage, our settings shouldn't get messed up on convert
	fulluser = user.ToUser(true)
	for k := range defoptions {
		_, ok := fulluser.Options[k]
		if !ok {
			t.Fatalf("Missing setting in full user: %s", k)
		}
	}
	// Set the user limit to something very low, then try creating a user again
	ctx.config.SudokuMaxUsers = 1
	_, err = ctx.RegisterSudokuUser("toomany", "toomany5")
	if err == nil {
		t.Fatalf("Expected error from registration for too many users")
	}
	realerr, ok := err.(*utils.OutOfSpaceError)
	if !ok {
		t.Fatalf("Error was not of type OutOfSpaceError")
	}
	if realerr.Allowed != 1 {
		t.Fatalf("Allowed should be 1, was %d", realerr.Allowed)
	}
}

func TestRetrieveData(t *testing.T) {
	ctx := newTestContext("retrievesudokudata")
	sets, err := ctx.GetPuzzleSets(-1)
	if err != nil {
		t.Fatalf("Error retrieving puzzle sets: %s", err)
	}
	if len(sets) == 0 {
		t.Fatalf("Didn't get any puzzle sets!")
	}
	found := false
	for _, ps := range sets {
		if strings.Index(ps.PuzzleSet, "Medium") >= 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("No puzzle sets with 'Medium' in name: %v", sets)
	}
	// Now some other data retrieval functions
	puzzleset, err := ctx.GetPuzzlesetData("Medium Pack 1", -1)
	if err != nil {
		t.Fatalf("Couldn't pull puzzleset data: %s", err)
	}
	if len(puzzleset) != 9 {
		t.Fatalf("Expected 9 puzzles, got %d", len(puzzleset))
	}
	// Need a user for this test
	uid := newSudokuUser("someuser", "garbage", t, ctx)
	puzzle, err := ctx.GetPuzzle(puzzleset[0].Pid, uid)
	if err != nil {
		t.Fatalf("Couldn't pull puzzle: %s", err)
	}
	if puzzle.PuzzleSet != "Medium Pack 1" {
		t.Fatalf("Puzzle in wrong set: %s vs Medium Pack 1", puzzle.PuzzleSet)
	}
	if len(puzzle.Puzzle) != 81 {
		t.Fatalf("Puzzle doesn't contain a proper puzzle: %s", puzzle.Puzzle)
	}
	if len(puzzle.Solution) != 81 {
		t.Fatalf("Puzzle doesn't contain a proper solution: %s", puzzle.Solution)
	}
	if puzzle.Seconds != 0 {
		t.Fatalf("Expected 0 seconds on new puzzle, got %d", puzzle.Seconds)
	}
}

func TestSudokuProgress(t *testing.T) {
	const PuzzleSetName = "Medium Pack 1"
	ctx := newTestContext("sudokuprogress")
	// Need a user for this test
	uid := newSudokuUser("someuser", "garbage", t, ctx)
	// Need to find SOME puzzle to work with (need valid pid)
	puzzleset, err := ctx.GetPuzzlesetData(PuzzleSetName, -1)
	if err != nil {
		t.Fatalf("Couldn't pull puzzleset data: %s", err)
	}
	puzzle, err := ctx.GetPuzzle(puzzleset[0].Pid, uid)
	if err != nil {
		t.Fatalf("Couldn't pull puzzle data: %s", err)
	}
	// Now, let's run through a normal scenario. Let's save some progress.
	data := SudokuSaveData{Puzzle: "this is not a puzzle"}
	jdata, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Couldn't create json: %s", err)
	}
	solved, err := ctx.UpdateSudokuProgress(puzzle.Pid, uid, string(jdata), 55)
	if err != nil {
		t.Fatalf("Couldn't update sudoku progress: %s", err)
	}
	if solved {
		t.Fatalf("Not supposed to be solved yet!")
	}
	// Go get the puzzle data again
	puzzle2, err := ctx.GetPuzzle(puzzle.Pid, uid)
	if puzzle2.PlayerSolution != string(jdata) {
		t.Fatalf("Progress not saved: %s vs %s", puzzle2.PlayerSolution, string(jdata))
	}
	if puzzle2.Seconds != 55 {
		t.Fatalf("Expected 55 seconds on pause, got %d", puzzle2.Seconds)
	}
	puzzleset2, err := ctx.GetPuzzlesetData(PuzzleSetName, uid)
	if err != nil {
		t.Fatalf("Couldn't pull puzzleset data: %s", err)
	}
	if !puzzleset2[0].Paused {
		t.Fatalf("Puzzle was supposed to be paused!")
	}
	if time.Now().Sub(*puzzleset2[0].PausedOn).Seconds() > 5 {
		t.Fatalf("Paused time not set correctly!")
	}
	if puzzleset2[0].Completed {
		t.Fatalf("Not supposed to be completed yet!")
	}
	// Update the puzzle with the solution. It should be solved now!
	data.Puzzle = puzzle.Solution
	jdata, err = json.Marshal(data)
	if err != nil {
		t.Fatalf("Couldn't create json: %s", err)
	}
	solved, err = ctx.UpdateSudokuProgress(puzzle.Pid, uid, string(jdata), 108)
	if err != nil {
		t.Fatalf("Couldn't update sudoku progress: %s", err)
	}
	if !solved {
		t.Fatalf("Supposed to be solved!")
	}
	puzzle3, err := ctx.GetPuzzle(puzzle.Pid, uid)
	if puzzle3.PlayerSolution != "" { //string(jdata) {
		t.Fatalf("Progress not deleted on completion: %s", puzzle3.PlayerSolution)
		//t.Fatalf("Progress not saved: %s vs %s", puzzle2.PlayerSolution, string(jdata))
	}
	puzzleset3, err := ctx.GetPuzzlesetData(PuzzleSetName, uid)
	if err != nil {
		t.Fatalf("Couldn't pull puzzleset data: %s", err)
	}
	if puzzleset3[0].Paused {
		t.Fatalf("Completed puzzle is not supposed to be paused!")
	}
	if time.Now().Sub(*puzzleset3[0].CompletedOn).Seconds() > 5 {
		t.Fatalf("Completed time not set correctly!")
	}
	if !puzzleset3[0].Completed {
		t.Fatalf("Puzzle supposed to be completed!")
	}
}
