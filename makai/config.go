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
	BusyTimeout = 5000
)

type Config struct {
	RootPath       string // The root path to makai (the url path)
	AdminId        string // Admin key
	DrawingsPath   string // Path to drawings for simple makai drawer
	TemplatePath   string // path to all makai templates
	StaticFilePath string // path to all static files
	KlandUrl       string // URL
	// MaxImageSize        int            // Maximum image upload size. It's a hard cutoff
	// DatabasePath        string         // path to database
	// ImagePath           string         // path to images on local filesystem
	// TextPath            string         // path to text data (animations?) on local filesystem
	// TempPath            string         // Place to put all temporary files
	// UploadPerInterval   int            // Amount of uploads (any) allowed per timespan
	// UploadLimitInterval utils.Duration // interval for upload limits
	// VisitPerInterval    int            // Amount of visits (any) allowed per timespan
	// VisitLimitInterval  utils.Duration // interval for visit limits
	// CookieExpire        utils.Duration // Expiration of cookie (admin cookie?)
	// IpHeader            string         // The header field for the user's IP
	// ShortUrl            string         // The endpoint for the short url
	// FullUrl             string         // The url for the "real" endpoint (where kland is hosted)
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
# MaxImageSize=10_000_000               # Maximum image upload size
# DatabasePath="data/kland/kland.db"    # Path to database (just data, not images)
# ImagePath="data/kland/images"         # Path to image folder
# TextPath="data/kland/text"            # Path to text folder (animations?)
# TempPath="data/tmp"                   # Path to put all temporary files
# UploadPerInterval=20                  # Amount of uploads (any) per interval
# UploadLimitInterval="1m"              # Interval for upload limit
# VisitPerInterval=100                  # Amount of visits (any) allowed per timespan
# VisitLimitInterval="1m"               # interval for visit limits
# CookieExpire="8760h"                  # Cookie expiration (for settings/etc)
# IPHeader="X-Real-IP"                  # Header field for user IP (assumes reverse proxy)
# ShortUrl="http://localhost:5020"      # The short domain 
# FullUrl="http://127.0.0.1:5020"       # The full domain 
`, time.Now().Format(time.RFC3339), randomHex)
}

// func (c *Config) OpenDb() (*sql.DB, error) {
// 	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath, BusyTimeout))
// }
