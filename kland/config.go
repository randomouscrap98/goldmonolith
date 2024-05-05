package kland

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"

	_ "github.com/mattn/go-sqlite3"
)

const (
	BusyTimeout = 5000
)

type Config struct {
	RootPath            string         // The root path to kland
	AdminId             string         // Admin key
	MaxImageSize        int            // Maximum image upload size. It's a hard cutoff
	DataPath            string         // Base path to all data (everything else relative to this)
	TempPath            string         // Place to put all temporary files
	StaticFilePath      string         // path to all static files
	TemplatePath        string         // path to all kland templates
	UploadPerInterval   int            // Amount of uploads (any) allowed per timespan
	UploadLimitInterval utils.Duration // interval for upload limits
	VisitPerInterval    int            // Amount of visits (any) allowed per timespan
	VisitLimitInterval  utils.Duration // interval for visit limits
	CookieExpire        utils.Duration // Expiration of cookie (admin cookie?)
	IpHeader            string         // The header field for the user's IP
	ShortUrl            string         // The endpoint for the short url
	FullUrl             string         // The url for the "real" endpoint (where kland is hosted)
	DefaultIPP          int            // Default number of images per page
	MaxMultipartMemory  int64          // Maximum image upload form size before dumping to disk
	MaxTotalDataSize    int64          // Limit the total amount of data that the system stores
	MaxTotalFileCount   int64          // Limit the total amount of files the system stores
}

func GetDefaultConfig_Toml() string {
	randomUser := make([]byte, 16)
	_, err := rand.Read(randomUser)
	if err != nil {
		log.Printf("WARN: couldn't generate random user")
	}
	randomHex := hex.EncodeToString(randomUser)
	return fmt.Sprintf(`# Config auto-generated on %s
RootPath="/kland"                     # Root path for kland (if at root, leave BLANK)
AdminId="%s"                          # Admin key (randomly generated)
MaxImageSize=10_000_000               # Maximum image upload size
DataPath="data/kland"                 # Base path to data (all other data relative to this)
TempPath="/tmp/kland"                 # Path to put all temporary files
StaticFilePath="static/kland"         # Path to static files (currently only valid in monolith)
TemplatePath="static/kland/templates" # Path to all template files
UploadPerInterval=20                  # Amount of uploads (any) per interval
UploadLimitInterval="1m"              # Interval for upload limit
VisitPerInterval=100                  # Amount of visits (any) allowed per timespan
VisitLimitInterval="1m"               # interval for visit limits
CookieExpire="8760h"                  # Cookie expiration (for settings/etc)
IPHeader="X-Real-IP"                  # Header field for user IP (assumes reverse proxy)
ShortUrl="http://localhost:5020"      # The short domain 
FullUrl="http://127.0.0.1:5020"       # The full domain 
DefaultIpp=20                         # Default number of images per page
MaxMultipartMemory=256_00             # Maximum image upload form size before dumping to disk
MaxTotalDataSize=6_000_000_000        # Max total size of kland data on filesystem.
MaxTotalFileCount=50_000              # Max amount of total files kland will support. Set both this and MaxTotalDataSize to 0 to disable (can be slow)
`, time.Now().Format(time.RFC3339), randomHex)
}

func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataPath, "kland.db")
}

func (c *Config) ImagePath() string {
	return filepath.Join(c.DataPath, "images")
}

func (c *Config) TextPath() string {
	return filepath.Join(c.DataPath, "text")
}

func (c *Config) OpenDb() (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d", c.DatabasePath(), BusyTimeout))
}
