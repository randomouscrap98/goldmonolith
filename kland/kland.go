package kland

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version = "0.1.0"
)

type KlandContext struct {
	config *Config
}

func NewKlandContext(config *Config) (*KlandContext, error) {
	// MUST have database exist and in good standing...
	dir, _ := filepath.Split(config.DatabasePath)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return nil, err
	}
	err = CreateTables(config)
	if err != nil {
		return nil, err
	}
	err = utils.VerifyVersionedDb(config, DatabaseVersion)
	if err != nil {
		return nil, err
	}
	// MUST have image folder existing...
	err = os.MkdirAll(config.ImagePath, 0750)
	if err != nil {
		return nil, err
	}
	// Now we're good to go
	return &KlandContext{
		config: config,
	}, nil
}

func (wc *KlandContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	// A stub, do nothing. But you HAVE to exit the wait group!!
	log.Printf("No background tasks for kland")
	wg.Done()
}

func (kctx *KlandContext) GetHandler() http.Handler {
	r := chi.NewRouter()
	return r
}
