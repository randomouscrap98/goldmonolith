package main

import (
	"fmt"
	"time"
)

type Config struct {
	Port int
}

func GetDefaultConfig_Toml() string {
	return fmt.Sprintf(`# Config auto-generated on %s
Port=5020               # Which port to run the server on
`, time.Now().Format(time.RFC3339))
}
