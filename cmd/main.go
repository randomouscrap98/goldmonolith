package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	//"time"

	"github.com/chi-middleware/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pelletier/go-toml/v2"
	//"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/webstream"
)

const (
	ConfigFile = "config.toml"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func initConfig(allowRecreate bool) *Config {
	var config Config
	// Read the config. It's OK if it doesn't exist
	configData, err := os.ReadFile(ConfigFile)
	if err != nil {
		if allowRecreate {
			configRaw := GetDefaultConfig_Toml()
			err = os.WriteFile(ConfigFile, []byte(configRaw), 0600)
			if err != nil {
				log.Printf("ERROR: Couldn't write default config: %s\n", err)
			} else {
				log.Printf("Generated default config at %s\n", ConfigFile)
				return initConfig(false)
			}
		} else {
			log.Fatalf("WARN: Couldn't read config file %s: %s", ConfigFile, err)
		}
	} else {
		// If the config exists, it MUST be parsable.
		err = toml.Unmarshal(configData, &config)
		must(err)
	}
	return &config
}

func main() {
	log.Printf("Gold monolith server started\n")
	config := initConfig(true)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	//r.Use(middleware.Timeout(time.Duration(config.Timeout)))
	r.Use(proxy.ForwardedHeaders())
	//r.Use(httprate.LimitByIP(config.RateLimitCount, time.Duration(config.RateLimitInterval)))

	r.Mount("/webstream", webstream.GetHandler())

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: r,
		//MaxHeaderBytes: config.HeaderLimit,
	}

	log.Fatal(s.ListenAndServe())
}
