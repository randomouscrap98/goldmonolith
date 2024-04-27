package webstream

import (
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

// Configuration for the entirety of webstream
type Config struct {
	RoomRegex       string
	StreamFolder    string
	SingleDataLimit int            // Allowed amount of data to write at once
	StreamDataLimit int            // Allowed amount of data per room
	TotalRoomLimit  int            // Total amount of rooms allowed to be stored on the filesystem.
	IdleRoomTime    utils.Duration // Time since last write = dump if greater
	//TotalDataLimit  int64 // Total amount of data
}

// Retrieve a default configuration in TOML which should parse to
// a Config object
func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
RoomRegex="^[a-zA-Z0-9_-]{5,256}$"  # Allowed room names
StreamFolder="streams"              # Where to store the data streams on the filesystem
SingleDataLimit=50000               # Allowed amount of data in one write
StreamDataLimit=5000000             # Allowed amount of data for total room
# TotalDataLimit=1_000_000_000      # Total amount of data in all rooms
TotalRoomLimit=400                  # Total amount of rooms allowed to be created.
IdleRoomTime="1m"                   # ow long a room can have no writes in before dumping it to fs (AGGRESSIVE)

# NOTE: the upper limit of storage will be the TotalRoomLimit * StreamDataLimit.
# This config targets a 2GB general limit. It is difficult to enforce a global
# limit, as each write would require querying the filesystem, which in some cases
# may be prohibitively slow
`, time.Now().Format(time.RFC3339))
}
