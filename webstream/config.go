package webstream

import (
	"fmt"
	"time"
)

// Configuration for the entirety of webstream
type Config struct {
	RoomRegex       string
	SingleDataLimit int
	StreamDataLimit int
	TotalDataLimit  int64
	TotalRoomLimit  int
}

// Retrieve a default configuration in TOML which should parse to
// a Config object
func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
RoomRegex="^[a-zA-Z0-9\\-_]+$"    # Allowed room names
SingleDataLimit=50000             # Allowed amount of data in one write
StreamDataLimit=5000000           # Allowed amount of data for total room
TotalDataLimit=1_000_000_000      # Total amount of data in all rooms
TotalRoomLimit=4096               # Total amount of rooms allowed to be created
`, time.Now().Format(time.RFC3339))
}
