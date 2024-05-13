package makai

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"

	"github.com/randomouscrap98/goldmonolith/utils"
)

var mtestsudmu sync.Mutex

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
	config.SudokuDbPath = filepath.Join(config.DrawingsPath, "sudoku.db")
	// WARN: this is HORRIBLE but like... yeah...
	if strings.Index(name, "sudoku") >= 0 {
		err := os.MkdirAll(config.DrawingsPath, 0770)
		if err != nil {
			panic(err)
		}
		mtestsudmu.Lock()
		defer mtestsudmu.Unlock()
		sudfi, err := os.Open(filepath.Join("testfiles", "sudoku.db"))
		if err != nil {
			panic(err)
		}
		defer sudfi.Close()
		destfi, err := os.Create(config.SudokuDbPath)
		if err != nil {
			panic(err)
		}
		defer destfi.Close()
		_, err = io.Copy(destfi, sudfi)
		if err != nil {
			panic(err)
		}
	}
	return &config
}

func newTestContext(name string) *MakaiContext {
	context, err := NewMakaiContext(reasonableConfig(name))
	if err != nil {
		panic(err)
	}
	return context
}
