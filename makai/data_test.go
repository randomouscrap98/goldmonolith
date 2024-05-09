package makai

import (
	"path/filepath"
	"testing"

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
	if result.Result != nil {
		t.Fatalf("Expected nothing from empty artist, got %v", result.Result)
	}
}
