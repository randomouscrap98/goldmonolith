package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/chi-middleware/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/pelletier/go-toml/v2"

	//"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/kland"
	"github.com/randomouscrap98/goldmonolith/makai"
	"github.com/randomouscrap98/goldmonolith/utils"
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

func initConfig() *Config {
	var config Config
	results, err := utils.ReadConfigStack(ConfigFile, func(raw []byte) error {
		return toml.Unmarshal(raw, &config)
	}, 10)
	must(err)
	// This means the original config wasn't loaded; let's load it and try again
	if slices.Index(results, ConfigFile) < 0 {
		configRaw := GetDefaultConfig_Toml()
		must(os.WriteFile(ConfigFile, []byte(configRaw), 0600))
		log.Printf("Generated default config at %s\n", ConfigFile)
		return initConfig() // Redo the configs (it's fine...?)
	}
	log.Printf("Loaded %d config(s)", len(results))
	return &config
}

func initRouter(_ *Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(cors.AllowAll().Handler)
	//r.Use(middleware.Timeout(time.Duration(config.Timeout)))
	r.Use(proxy.ForwardedHeaders())
	//r.Use(httprate.LimitByIP(config.RateLimitCount, time.Duration(config.RateLimitInterval)))

	return r
}

// Initialize and spawn the http server for the given handler and with the given config
func runServer(handler http.Handler, config *Config) *http.Server {
	s := &http.Server{
		Addr:    config.Address,
		Handler: handler,
		//MaxHeaderBytes: config.HeaderLimit,
	}

	go func() {
		log.Printf("Running server in goroutine at %s", s.Addr)
		if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	return s
}

// Great readup: https://dev.to/mokiat/proper-http-shutdown-in-go-3fji
func waitForShutdown() {
	// Create a channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Block until a signal is received
	<-sigChan
}

func main() {
	log.Printf("Gold monolith server started\n")
	config := initConfig()

	// Context is something we'll cancel to cancel any and all background tasks
	// when the server gets a shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := initRouter(config)

	var wg sync.WaitGroup
	mounts := make(map[string]func() (WebService, error))

	// --- Which services to host ----
	mounts["/stream"] = func() (WebService, error) { return webstream.NewWebstreamContext(config.Webstream) }
	mounts["/kland"] = func() (WebService, error) { return kland.NewKlandContext(config.Kland) }
	mounts["/makai"] = func() (WebService, error) { return makai.NewMakaiContext(config.Makai) }

	// --- Host all services ---
	for k, f := range mounts {
		service, err := f() //webstream.NewWebstreamContext(config.Webstream)
		if err != nil {
			panic(err)
		}
		handler, err := service.GetHandler()
		if err != nil {
			panic(err)
		}
		r.Mount(k, handler)
		wg.Add(1)
		service.RunBackground(ctx, &wg)
		log.Printf("Mounted service at %s", k)
	}

	// --- Static files -----
	err := utils.FileServer(r, "/static", config.StaticFiles)
	if err != nil {
		panic(err)
	}
	log.Printf("Hosting static files at %s\n", config.StaticFiles)

	// --- Server ---
	s := runServer(r, config)
	waitForShutdown()

	log.Println("Shutting down...")
	cancel() // Cancel the context to signal goroutines to stop
	wg.Wait()
	log.Println("All background services stopped")

	// Create a context with a timeout to allow for graceful shutdown
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), time.Duration(config.ShutdownTime))
	defer cancelShutdown()

	// Shut down the server gracefully
	if err := s.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}
