package main

import (
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/kland"
	"github.com/randomouscrap98/goldmonolith/utils"
	"github.com/randomouscrap98/goldmonolith/webstream"
)

type Config struct {
	Address      string            // Full address to host on (includes IP to limit to localhost/etc)
	ShutdownTime utils.Duration    // Time to wait for server to shutdown
	Webstream    *webstream.Config // Webstream specific config
	Kland        *kland.Config     // Kland specific config
	StaticFiles  string            // Where the static files for ALL endpoints go
}

func GetDefaultConfig_Toml() string {
	baseConfig := `# Config auto-generated on %s
Address=":5020"               # Where to run the server
ShutdownTime="10s"            # How long to wait for the server to shutdown
StaticFiles="static"          # Where ALL static files go

[Webstream]
%s

[Kland]
%s
`
	return fmt.Sprintf(
		baseConfig,
		time.Now().Format(time.RFC3339),
		webstream.GetDefaultConfig_Toml(),
		kland.GetDefaultConfig_Toml(),
	)
}
