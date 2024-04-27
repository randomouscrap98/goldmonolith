package main

import (
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
	"github.com/randomouscrap98/goldmonolith/webstream"
)

type Config struct {
	Address      string            // Full address to host on (includes IP to limit to localhost/etc)
	ShutdownTime utils.Duration    // Time to wait for server to shutdown
	Webstream    *webstream.Config // Webstream specific config
}

func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
Address=":5020"               # Where to run the server
ShutdownTime="10s"            # How long to wait for the server to shutdown

[Webstream]
%s
`, time.Now().Format(time.RFC3339),
		webstream.GetDefaultConfig_Toml())
}
