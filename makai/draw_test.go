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

func mustList(artistId string, ctx *MakaiContext, t *testing.T) *ArtistData {
	mdata := ManagerData{
		Action:   "list",
		ArtistID: artistId,
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on missing artist (list), got %v", result.Errors)
	}
	if utils.IsInterfaceNil(result.Result) {
		return nil
	} else {
		adata, ok := result.Result.(*ArtistData)
		if !ok {
			t.Fatalf("Expected ArtistData from list command, got %v", result.Result)
		}
		return adata
	}
}

func mustSave(artistId string, drawing string, name string, ctx *MakaiContext, t *testing.T) string {
	mdata := ManagerData{
		Action:   "save",
		ArtistID: artistId,
		Drawing:  drawing,
		Name:     name,
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on saving drawing %s, got %v", name, result.Errors)
	}
	drawId, ok := result.Result.(string)
	if !ok {
		t.Fatalf("Expected string result from drawing add %s, got %v", name, result.Result)
	}
	return drawId
}

func mustLoad(artistId string, drawingId string, ctx *MakaiContext, t *testing.T) string {
	mdata := ManagerData{
		Action:    "load",
		ArtistID:  artistId,
		DrawingID: drawingId,
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) > 0 {
		t.Fatalf("Did not expect any errors on looking up drawing %s, got %v", drawingId, result.Errors)
	}
	drawing, ok := result.Result.(string)
	if !ok {
		t.Fatalf("Expected string result from drawing load %s, got %v", drawingId, result.Result)
	}
	return drawing
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

func TestEmptyArtist(t *testing.T) {
	ctx := newTestContext("emptyartist")
	result := mustList("something", ctx, t)
	if result != nil {
		t.Fatalf("Expected nothing from empty artist, got %v", result)
	}
	// Listing should NOT write anything
	apath, err := ctx.ArtistDataPath("something")
	if err != nil {
		t.Fatalf("Got error while getting artist data path: %s", err)
	}
	_, err = os.Stat(apath)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected no data to be written, but apparently was: %s", err)
	}
}

// The REALLY BIG test that does way too much
func TestAddDrawing(t *testing.T) {
	ctx := newTestContext("adddrawing")

	// Add a drawing and see if it was added
	origDrawing := "abc123alotofdata wow wow\nwow"
	drawId := mustSave("something", origDrawing, "garbo", ctx, t)
	drawing := mustLoad("something", drawId, ctx, t)
	if drawing != origDrawing {
		t.Fatalf("Loading the drawing didn't match: %s vs %s", drawing, origDrawing)
	}

	// Go lookup the artist data now and check a bunch of stuff
	artist := mustList("something", ctx, t)
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
	if ddata.Size != int64(len(origDrawing)) {
		t.Fatalf("Expected len %d, got %d", len(origDrawing), ddata.Size)
	}
	if ddata.WriteCount != 1 {
		t.Fatalf("Expected writecount to be 1, got %d", ddata.WriteCount)
	}

	// Now update the existing drawing
	updatedDrawing := "and now there's even more data for you to find! wow! and it just keeps going!"
	newId := mustSave("something", updatedDrawing, "garbo", ctx, t)
	if newId != drawId {
		t.Fatalf("Drawing ids did not match on update: %s vs %s", newId, drawId)
	}

	// Check the artist data again. It must all be good still
	artist2 := mustList("something", ctx, t)
	folder2, ok := artist2.Folders[artist2.RootFolder]
	if !ok {
		t.Fatalf("Artist's root folder %s did not exist in the folder map", artist2.RootFolder)
	}
	if len(folder2.Children) != 1 {
		t.Fatalf("Expected 1 folder child after drawing update, got %d", len(folder2.Children))
	}
	if len(artist2.Drawings) != 1 {
		t.Fatalf("Expected 1 drawing after drawing update, got %d", len(artist2.Drawings))
	}
	if slices.Index(folder2.Children, drawId) < 0 {
		t.Fatalf("Drawing 2 not added to the root folder")
	}
	ddata2, ok := artist2.Drawings[drawId]
	if !ok {
		t.Fatalf("Drawing 2 not found in drawing map")
	}
	if ddata2.Name != "garbo" {
		t.Fatalf("Expected drawing to have name garbo, got %s", ddata2.Name)
	}
	if ddata2.Size != int64(len(updatedDrawing)) {
		t.Fatalf("Expected len %d, got %d", len(updatedDrawing), ddata2.Size)
	}
	if ddata2.WriteCount != 2 {
		t.Fatalf("Expected writecount to be 2, got %d", ddata2.WriteCount)
	}

	// Now add an actual new file. Make sure it shows up in the appropriate places
	newDrawing := "and then there were two"
	secondId := mustSave("something", newDrawing, "garbo2", ctx, t)
	if secondId == drawId {
		t.Fatalf("Drawing did not get a new id")
	}
	checkNewDrawing := mustLoad("something", secondId, ctx, t)
	if checkNewDrawing != newDrawing {
		t.Fatalf("Couldn't load new drawing: not the same")
	}

	artist3 := mustList("something", ctx, t)
	folder3, ok := artist3.Folders[artist3.RootFolder]
	if !ok {
		t.Fatalf("Artist's root folder %s did not exist in the folder map", artist3.RootFolder)
	}
	if len(folder3.Children) != 2 {
		t.Fatalf("Expected 2 children of root folder after drawing update, got %d", len(folder3.Children))
	}
	if len(artist3.Drawings) != 2 {
		t.Fatalf("Expected 2 drawings after drawing update, got %d", len(artist3.Drawings))
	}
	if slices.Index(folder3.Children, drawId) < 0 {
		t.Fatalf("Drawing 1 not persisting in the root folder")
	}
	if slices.Index(folder3.Children, secondId) < 0 {
		t.Fatalf("Drawing 3 not added to the root folder")
	}
	ddata3, ok := artist3.Drawings[secondId]
	if !ok {
		t.Fatalf("Drawing 3 not found in drawing map")
	}
	if ddata3.Name != "garbo2" {
		t.Fatalf("Expected drawing to have name garbo, got %s", ddata3.Name)
	}
	if ddata3.Size != int64(len(newDrawing)) {
		t.Fatalf("Expected len %d, got %d", len(newDrawing), ddata3.Size)
	}
	if ddata3.WriteCount != 1 {
		t.Fatalf("Expected writecount to be 1, got %d", ddata3.WriteCount)
	}
}

func TestOutOfSpace(t *testing.T) {
	ctx := newTestContext("outofspace")

	// First save should work
	_ = mustSave("nospace", "hecking whatever dude", "test", ctx, t)

	// Now greatly limit the file count
	ctx.config.MaxDrawingFiles = 2

	// Remaining saves will not work
	mdata := ManagerData{
		Action:   "save",
		ArtistID: "nospace",
		Drawing:  "hecking whatever dude",
		Name:     "test2",
	}
	result := ctx.DrawManager(&mdata)
	if len(result.Errors) < 1 {
		t.Fatalf("Expected at least 1 error due to file count limit")
	}
	if strings.Index(result.Errors[0], "space") < 0 || strings.Index(result.Errors[0], "2") < 0 {
		t.Fatalf("Didn't find expected strings in out of space output: %v", result.Errors)
	}

	// Now greatly limit the file size
	ctx.config.MaxDrawingFiles = 20
	ctx.config.MaxDrawingData = 100

	// Remaining saves will not work
	result = ctx.DrawManager(&mdata)
	if len(result.Errors) < 1 {
		t.Fatalf("Expected at least 1 error due to file size limit")
	}
	if strings.Index(result.Errors[0], "space") < 0 || strings.Index(result.Errors[0], "100") < 0 {
		t.Fatalf("Didn't find expected strings in out of space output: %v", result.Errors)
	}
}
