package webstream

import (
	"sync"
)

// A collection of webstreams, representing essentially the runtime
// for the webstream website
type WebstreamCollection struct {
	webstreams map[string]*WebStream
	lock       sync.Mutex
	backer     WebStreamBacker
}

func NewWebstreamCollection(backing WebStreamBacker) *WebstreamCollection {
	return &WebstreamCollection{
		webstreams: make(map[string]*WebStream),
		backer:     backing,
	}
}

// Safely retrieve a webstream. The backing is automatic, you just read
// and write anytime you want.
func (wc *WebstreamCollection) GetStream(name string) *WebStream {
	wc.lock.Lock()
	defer wc.lock.Unlock()
	ws, ok := wc.webstreams[name]
	if !ok {
		ws = NewWebStream(name, wc.backer)
		wc.webstreams[name] = ws
	}
	return ws
}
