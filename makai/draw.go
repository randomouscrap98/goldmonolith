package makai

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type DrawingData struct {
	Created        time.Time `json:"created"`
	Modified       time.Time `json:"modified"`
	Size           int64     `json:"size"`
	Name           string    `json:"name"` // drawingName
	Tags           []string  `json:"tags"`
	WriteCount     int       `json:"writecount"`
	PersonalRating int       `json:"personalrating"` // Out of 10
}

type FolderData struct {
	Created  time.Time `json:"created"`
	Name     string    `json:"name"` // folderName
	Children []string  `json:"children"`
}

type ArtistData struct {
	ArtistID   string                  `json:"artistID"`
	Joined     time.Time               `json:"joined"`
	Drawings   map[string]*DrawingData `json:"drawings"`
	Folders    map[string]*FolderData  `json:"folders"`    // Folders aren't properly implemented
	RootFolder string                  `json:"rootfolder"` // Folders aren't properly implemented
}

type ManagerData struct {
	Action    string `schema:"action"`
	ArtistID  string `schema:"artistID"`
	DrawingID string `schema:"drawingID"`
	Drawing   string `schema:"drawing"`
	FolderID  string `schema:"folderID"`
	Name      string `schema:"name"`
}

type ManagerResult struct {
	Errors    []string    `json:"errors"`
	Result    interface{} `json:"result"`
	InputHelp []string    `json:"inputhelp"`
}

func NewFolderData(name string) *FolderData {
	return &FolderData{
		Created:  time.Now(),
		Name:     name,
		Children: make([]string, 0),
	}
}

func NewDrawingData(name string) *DrawingData {
	return &DrawingData{
		Created:        time.Now(),
		Modified:       time.Now(),
		Size:           0,
		Name:           name,
		Tags:           make([]string, 0),
		WriteCount:     0,
		PersonalRating: 0,
	}
}

// Generate some random ID useful for the draw system
func newDrawSystemId() string {
	rawuuid, err := uuid.NewRandom()
	if err != nil {
		return fmt.Sprintf("%s_%d", time.Now().Format(time.RFC3339), rand.Uint32())
	} else {
		return rawuuid.String()
	}
}

// The way this function is used, it's just easier to return the pointer.
// I don't care, whatever, this is ancient history anyway
func NewArtistData(artistId string) *ArtistData {
	rootid := newDrawSystemId()
	folders := make(map[string]*FolderData)
	folders[rootid] = NewFolderData(artistId)
	return &ArtistData{
		ArtistID:   artistId,
		Joined:     time.Now(),
		Drawings:   make(map[string]*DrawingData),
		Folders:    folders,
		RootFolder: rootid,
	}
}

// The path to the data.json for an artist. Returns errors if the string is dumb
func (mctx *MakaiContext) ArtistDataPath(artistId string) (string, error) {
	if artistId == "" || !mctx.drawRegex.Match([]byte(artistId)) {
		return "", fmt.Errorf("Invalid characters in artistId. Regex: %s", mctx.config.DrawSafetyRegex)
	}
	return filepath.Join(mctx.config.DrawingsPath, artistId, ArtistJsonFile), nil
}

// The path to a single drawing for an artist / drawing. Returns errors if either string is dumb
func (mctx *MakaiContext) DrawingPath(artistId string, drawingId string) (string, error) {
	if artistId == "" || !mctx.drawRegex.Match([]byte(artistId)) {
		return "", fmt.Errorf("Invalid characters in artistId. Regex: %s", mctx.config.DrawSafetyRegex)
	}
	if drawingId == "" || !mctx.drawRegex.Match([]byte(drawingId)) {
		return "", fmt.Errorf("Invalid characters in drawingId. Regex: %s", mctx.config.DrawSafetyRegex)
	}
	return filepath.Join(mctx.config.DrawingsPath, artistId, drawingId), nil
}

// Retrieve existing artist data from the filesystem. if no artist data exists,
// does NOT return error: specifically returns nil (this is how the old system worked)
func (mctx *MakaiContext) GetArtistData(artistId string) (*ArtistData, error) {
	apath, err := mctx.ArtistDataPath(artistId)
	if err != nil {
		return nil, err
	}
	file, err := os.ReadFile(apath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // This is OK, there's just no artist data yet
		} else {
			return nil, err
		}
	}
	var result ArtistData
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Return the raw drawing data. Apparently, the javascript system just stores arbitrary strings,
// which we DON'T parse. The old system returned null on error, we ACTUALLY return an error
func (mctx *MakaiContext) GetDrawingData(artistId string, drawingId string) ([]byte, error) {
	dpath, err := mctx.DrawingPath(artistId, drawingId)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(dpath)
}

// Add or update the drawing data statistics for the drawing with the given name in the given
// folder. This means updating statistics, or creating all the links appropriate in the artist data
func (mctx *MakaiContext) UpdateDrawingData(name string, size int64, folderId string, artist *ArtistData) (string, error) {
	// Folder must exist (why don't we just create it for them though?)
	fdata, ok := artist.Folders[folderId]
	if !ok {
		return "", fmt.Errorf("Folder %s not found for artist %s", folderId, artist.ArtistID)
	}
	var ddata *DrawingData
	var drawingId string
	// Scan over the children of the folder, which are the IDS we can use to index
	// into the drawings map. We want a reference to the existing drawing and the ID
	for _, child := range fdata.Children {
		ddata, ok = artist.Drawings[child]
		if ok && ddata.Name == name {
			drawingId = child
			break
		}
	}
	// This is a new drawing, because we couldn't find it above.
	if drawingId == "" {
		log.Printf("Creating new drawing data '%s' for %s", name, artist.ArtistID)
		drawingId = newDrawSystemId()
		ddata = NewDrawingData(name)
		artist.Folders[folderId].Children = append(artist.Folders[folderId].Children, drawingId)
		artist.Drawings[drawingId] = ddata
	}
	// Update some fields in the drawing data. It's a pointer, so it should update
	// the struct we expect.
	ddata.Modified = time.Now()
	ddata.WriteCount += 1
	ddata.Size = size
	return drawingId, nil
}

func (mctx *MakaiContext) DrawManager(data *ManagerData) *ManagerResult {
	result := ManagerResult{
		InputHelp: []string{"action", "artistID", "drawing", "drawingID", "folderID", "name"},
		Errors:    make([]string, 0),
	}

	addError := func(err string) *ManagerResult {
		result.Errors = append(result.Errors, err)
		return &result
	}
	addErr := func(err error) *ManagerResult {
		return addError(err.Error())
	}

	if data.ArtistID == "" {
		return addError("No artist ID given!")
	}

	// Now actually do stuff based on the action
	if data.Action == "list" {
		adata, err := mctx.GetArtistData(data.ArtistID)
		if err != nil {
			return addErr(err)
		}
		result.Result = adata
	} else if data.Action == "load" {
		ddata, err := mctx.GetDrawingData(data.ArtistID, data.DrawingID)
		if err != nil {
			return addErr(err)
		}
		result.Result = string(ddata)
	} else if data.Action == "save" {
		if data.Drawing == "" {
			return addError("Did not supply drawing data!")
		} else if data.Name == "" {
			return addError("No name supplied for the drawing!")
		}
		artist, err := mctx.GetArtistData(data.ArtistID)
		if err != nil {
			return addErr(err)
		}
		if artist == nil {
			log.Printf("Creating new artist %s while saving drawing %s", data.ArtistID, data.Name)
			artist = NewArtistData(data.ArtistID)
		}
		if artist.RootFolder == "" {
			return addError("Artist root folder not set (programming error!)")
		}
		drawingId, err := mctx.UpdateDrawingData(data.Name, int64(len(data.Drawing)), artist.RootFolder, artist)
	} else {
		return addError("Unknown action!")
	}

	return &result
}
