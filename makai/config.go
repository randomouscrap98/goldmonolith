package makai

import (
	"crypto/rand"
	//"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	//"github.com/randomouscrap98/goldmonolith/utils"

	_ "github.com/mattn/go-sqlite3"
)

const (
	BusyTimeout    = 5000
	ArtistJsonFile = "data.json"
)

type Config struct {
	RootPath            string // The root path to makai (the url path)
	AdminId             string // Admin key
	DrawingsPath        string // Path to drawings for simple makai drawer
	TemplatePath        string // path to all makai templates
	StaticFilePath      string // path to all static files
	KlandUrl            string // URL
	DrawSafetyRegex     string // General regex for checking user-input strings
	MaxDrawingData      int64  // Maximum amount of data drawings are allowed to take up
	MaxDrawingFiles     int64  // Maximum amount of total files stored in the drawing system
	MaxFormMemory       int    // Maximum form size to load entirely into memory for makai
	ChatlogIncludeRegex string // Regex for allowed include format in chatlogs
	ChatlogFileGlob     string // Glob for finding chatlog files
	ChatlogGrepChunk    int    // Amount of files to pass to grep all at once
	ChatlogUrl          string // Full url for where to find actual chatlogs
	ChatlogLogging      bool   // Whether to log chatlog search commands
}

func GetDefaultConfig_Toml() string {
	randomUser := make([]byte, 16)
	_, err := rand.Read(randomUser)
	if err != nil {
		log.Printf("WARN: couldn't generate random user")
	}
	randomHex := hex.EncodeToString(randomUser)
	return fmt.Sprintf(`# Config auto-generated on %s
RootPath="/makai"                     # Root path for makai (if at root, leave BLANK)
AdminId="%s"                          # Admin key (randomly generated)
DrawingsPath="data/makai/drawings"    # Drawings path for simple makai drawer
TemplatePath="static/makai/templates" # Path to all template files
StaticFilePath="static/makai"         # Path to static files (currently only valid in monolith)
KlandUrl="/kland"                     # Full url to the root page of kland
DrawSafetyRegex="^[a-zA-Z0-9_-]+$"    # General regex for checking user-input strings
MaxDrawingData=500_000_000            # Maximum amount of data drawings are allowed to take up total
MaxDrawingFiles=50_000                # Maximum amount of total files in the drawing system
MaxFormMemory=500_000                 # Maximum form size to load entirely into memory for makai
ChatlogIncludeRegex="^[a-zA-z0-9*-]+$"       # Regex for allowed include format in chatlogs
ChatlogFileGlob="data/makai/chatlogs/*.txt"  # Glob for finding chatlog files
ChatlogGrepChunk=100                  # Amount of files to pass to grep all at once
ChatlogUrl=""                         # Full url for where to find actual chatlogs
ChatlogLogging=false                  # Whether to log chatlog search commands
`, time.Now().Format(time.RFC3339), randomHex)
}

// func (c *Config) OpenDb() (*sql.DB, error) {
// 	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath, BusyTimeout))
// }
