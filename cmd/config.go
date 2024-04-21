package main

import (
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/webstream"
)

type Config struct {
	Port      int
	Webstream *webstream.Config
}

func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
Port=5020               # Which port to run the server on

[Webstream]
%s
`, time.Now().Format(time.RFC3339),
		webstream.GetDefaultConfig_Toml())
}
