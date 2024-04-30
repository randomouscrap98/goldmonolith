package kland

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"

	_ "github.com/mattn/go-sqlite3"
)

const (
	BusyTimeout = 5000
)

type Config struct {
	DatabasePath        string         // path to database
	ImagePath           string         // path to images
	TextPath            string         // path to text data (animations?)
	StaticFilePath      string         // path to all static files
	UploadPerInterval   int            // Amount of uploads (any) allowed per timespan
	UploadLimitInterval utils.Duration // interval for upload limits
}

func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
DatabasePath="data/kland/kland.db"   # Path to database (just data, not images)
ImagePath="data/kland/images"        # Path to image folder
StringPath="data/kland/text"         # Path to text folder (animations?)
StaticFilePath="static/kland"        # Path to static files (currently only valid in monolith)
UploadPerInterval=20                 # Amount of uploads (any) per interval
UploadLimitInterval="1m"             # Interval for upload limit
`, time.Now().Format(time.RFC3339))
}

func (c *Config) OpenDb() (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath, BusyTimeout))
}
