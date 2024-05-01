package kland

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
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
	TemplatePath        string         // path to all kland templates
	UploadPerInterval   int            // Amount of uploads (any) allowed per timespan
	UploadLimitInterval utils.Duration // interval for upload limits
	AdminId             string         // Admin key
	CookieExpire        utils.Duration // Expiration of cookie (admin cookie?)
}

func GetDefaultConfig_Toml() string {
	randomUser := make([]byte, 16)
	_, err := rand.Read(randomUser)
	if err != nil {
		log.Printf("WARN: couldn't generate random user")
	}
	randomHex := hex.EncodeToString(randomUser)
	return fmt.Sprintf(`# Config auto-generated on %s
DatabasePath="data/kland/kland.db"    # Path to database (just data, not images)
ImagePath="data/kland/images"         # Path to image folder
StringPath="data/kland/text"          # Path to text folder (animations?)
StaticFilePath="static/kland"         # Path to static files (currently only valid in monolith)
TemplatePath="static/kland/templates" # Path to all template files
UploadPerInterval=20                  # Amount of uploads (any) per interval
UploadLimitInterval="1m"              # Interval for upload limit
AdminId="%s"                          # Admin key (randomly generated)
`, time.Now().Format(time.RFC3339), randomHex)
}

func (c *Config) OpenDb() (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath, BusyTimeout))
}
