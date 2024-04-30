package main

import (
	"context"
	"net/http"
	"sync"
)

type WebService interface {
	RunBackground(context.Context, *sync.WaitGroup)
	GetHandler() (http.Handler, error)
}
