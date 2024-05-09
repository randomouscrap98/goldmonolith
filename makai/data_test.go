package makai

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/randomouscrap98/goldmonolith/utils"
)

// type ManagerData struct {
// 	Action    string `schema:"action"`
// 	ArtistID  string `schema:"artistID"`
// 	DrawingID string `schema:"drawingID"`
// 	Drawing   string `schema:"drawing"`
// 	FolderID  string `schema:"folderID"`
// 	Name      string `schema:"name"`
// }

func reasonableConfig(name string) *Config {
	config := Config{}
	// Get a baseline config from toml
	rawconfig := GetDefaultConfig_Toml()
	err := toml.Unmarshal([]byte(rawconfig), &config)
	if err != nil {
		panic(err)
	}
	// Set some fields to test values
	config.DrawingsPath = utils.RandomTestFolder(name, false)
	config.TemplatePath = filepath.Join("..", "cmd", config.TemplatePath)
	// WARN: You will need to change the above if the structure of the project changes
	return &config
}

func newTestContext(name string) *MakaiContext {
	context, err := NewMakaiContext(reasonableConfig(name))
	if err != nil {
		panic(err)
	}
	return context
}

func TestEmptyArtist(t *testing.T) {
	ctx := newTestContext("emptyartist")
	mdata := ManagerData{
		Action:   "list",
		ArtistID: "something",
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on missing artist (list), got %v", result.Errors)
	}
	if !utils.IsInterfaceNil(result.Result) {
		t.Fatalf("Expected nothing from empty artist, got %v", result.Result)
	}
	// Listing should NOT write anything
	apath, err := ctx.ArtistDataPath(mdata.ArtistID)
	if err != nil {
		t.Fatalf("Got error while getting artist data path: %s", err)
	}
	_, err = os.Stat(apath)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected no data to be written, but apparently was: %s", err)
	}
}

func TestBadArtist(t *testing.T) {
	ctx := newTestContext("badartist")
	mdata := ManagerData{
		Action:   "list",
		ArtistID: "&*($^^",
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) == 0 {
		t.Fatalf("Expected an error because of bad artist, got none")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Index(e, "Invalid character") >= 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected to find invalid character error, got %v", result.Errors)
	}
}

func TestAddDrawing(t *testing.T) {
	ctx := newTestContext("adddrawing")
	mdata := ManagerData{
		Action:   "save",
		ArtistID: "something",
		Drawing:  "abc123alotofdata wow wow\nwow",
		Name:     "garbo",
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on saving drawing, got %v", result.Errors)
	}
	drawId, ok := result.Result.(string)
	if !ok {
		t.Fatalf("Expected string result from drawing add, got %v", result.Result)
	}
	// Now, go get the drawing data. It should be there
	mdata2 := ManagerData{
		Action:    "load",
		ArtistID:  "something",
		DrawingID: drawId,
	}
	result = ctx.DrawManager(&mdata2)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on looking up drawing, got %v", result.Errors)
	}
	drawing, ok := result.Result.(string)
	if !ok {
		t.Fatalf("Expected string result from drawing load, got %v", result.Result)
	}
	if drawing != mdata.Drawing {
		t.Fatalf("Loading the drawing didn't match: %s vs %s", drawing, mdata.Drawing)
	}
	mdata3 := ManagerData{
		Action:   "list",
		ArtistID: "something",
	}
	result = ctx.DrawManager(&mdata3)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on looking up artist, got %v", result.Errors)
	}
	artist, ok := result.Result.(*ArtistData)
	if !ok {
		t.Fatalf("Expected artist result from artist load, got %v", result.Result)
	}
	if artist.ArtistID != "something" {
		t.Fatalf("Expected artistid something, got %s", artist.ArtistID)
	}
	if time.Now().Sub(artist.Joined).Seconds() > 1 {
		t.Fatalf("Expected artist to join soon, got %s", artist.Joined)
	}
	folder, ok := artist.Folders[artist.RootFolder]
	if !ok {
		t.Fatalf("Artist's root folder %s did not exist in the folder map", artist.RootFolder)
	}
	if slices.Index(folder.Children, drawId) < 0 {
		t.Fatalf("Drawing not added to the root folder")
	}
	ddata, ok := artist.Drawings[drawId]
	if !ok {
		t.Fatalf("Drawing not found in drawing map")
	}
	if time.Now().Sub(ddata.Created).Seconds() > 1 {
		t.Fatalf("Expected drawing to be created soon, got %s", ddata.Created)
	}
	if ddata.Name != "garbo" {
		t.Fatalf("Expected drawing to have name garbo, got %s", ddata.Name)
	}
	if ddata.Size != int64(len(mdata.Drawing)) {
		t.Fatalf("Expected len %d, got %d", len(mdata.Drawing), ddata.Size)
	}
	if ddata.WriteCount != 1 {
		t.Fatalf("Expected writecount to be 1, got %d", ddata.WriteCount)
	}
	mdata4 := ManagerData{
		Action:   "save",
		ArtistID: "something",
		Drawing:  "and now there's even more data for you to find! wow! and it just keeps going!",
		Name:     "garbo",
	}
	result = ctx.DrawManager(&mdata4)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on saving drawing 2, got %v", result.Errors)
	}
	drawId2, ok := result.Result.(string)
	if !ok {
		t.Fatalf("Expected string result from drawing add 2, got %v", result.Result)
	}
	if drawId2 != drawId {
		t.Fatalf("Drawing ids did not match on update: %s vs %s", drawId2, drawId)
	}
	mdata5 := ManagerData{
		Action:   "list",
		ArtistID: "something",
	}
	result = ctx.DrawManager(&mdata5)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on looking up artist 2, got %v", result.Errors)
	}
	artist2, ok := result.Result.(*ArtistData)
	if !ok {
		t.Fatalf("Expected artist result from artist load 2, got %v", result.Result)
	}
	folder2, ok := artist.Folders[artist2.RootFolder]
	if !ok {
		t.Fatalf("Artist's root folder %s did not exist in the folder map", artist2.RootFolder)
	}
	if len(folder2.Children) != 1 {
		t.Fatalf("Expected 1 folder child after drawing update, got %d", len(folder2.Children))
	}
	if len(artist2.Drawings) != 1 {
		t.Fatalf("Expected 1 drawing after drawing update, got %d", len(artist2.Drawings))
	}
	if slices.Index(folder.Children, drawId) < 0 {
		t.Fatalf("Drawing 2 not added to the root folder")
	}
	ddata2, ok := artist2.Drawings[drawId]
	if !ok {
		t.Fatalf("Drawing 2 not found in drawing map")
	}
	if ddata2.Name != "garbo" {
		t.Fatalf("Expected drawing to have name garbo, got %s", ddata2.Name)
	}
	if ddata2.Size != int64(len(mdata4.Drawing)) {
		t.Fatalf("Expected len %d, got %d", len(mdata4.Drawing), ddata2.Size)
	}
	if ddata2.WriteCount != 2 {
		t.Fatalf("Expected writecount to be 2, got %d", ddata2.WriteCount)
	}
}
