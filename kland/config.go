package kland

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	BusyTimeout = 5000
)

type Config struct {
	DatabasePath   string // path to database
	ImagePath      string // path to images
	StaticFilePath string // path to all static files
}

func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
DatabasePath="data/kland/kland.db"   # Path to database (just data, not images)
ImagePath="data/kland/images"        # Path to image folder
StaticFilePath="static/kland"   # Path to static files (currently only valid in monolith)
`, time.Now().Format(time.RFC3339))
}

func (c *Config) OpenDb() (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath, BusyTimeout))
}
