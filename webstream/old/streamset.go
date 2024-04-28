/* package webstream

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

type StreamSet struct {
	roomRegex *regexp.Regexp
	//obfuscator *utils.ObfuscatedKeys
	backer      WebStreamBacker
	webstreams  map[string]*WebStream
	wslock      sync.Mutex
	IdleTimeout time.Duration
	//config     *Config
}

func NewStreamSet(config *Config, backer WebStreamBacker) (*StreamSet, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	return &StreamSet{
		//config:     config,
		backer:      backer,
		roomRegex:   roomRegex,
		webstreams:  make(map[string]*WebStream),
		IdleTimeout: time.Duration(config.IdleRoomTime),
	}, nil
}

// Get a ready-to-use instance of of a webstream, usable with reads
// and writes immediately, auto-backed by whatever backer is there
func (ss *StreamSet) GetStream(name string) (*WebStream, error) {
	if !ss.roomRegex.MatchString(name) {
		return nil, fmt.Errorf("Room name has invalid characters! Try something simpler!")
	}
	ss.wslock.Lock()
	defer ss.wslock.Unlock()
	ws, ok := ss.webstreams[name]
	if !ok {
		ws = NewWebStream(name, ss.backer)
		refreshed, err := ws.RefreshStream()
		if err != nil {
			log.Printf("ERROR: Couldn't load webstream %s: %s", name, err)
		}
		if refreshed {
			log.Printf("First load of stream %s from persistent backing", name)
		}
		ss.webstreams[name] = ws
	}
	return ws, nil
}

func (ss *StreamSet) DumpStreams(force bool) {
	ss.wslock.Lock()
	defer ss.wslock.Unlock()
	for k, v := range ss.webstreams {
		if force || time.Now().Sub(v.GetLastWrite()) > ss.IdleTimeout {
			if force {
				log.Printf("FORCE DUMPING STREAM: %s", k)
			}
			ok, err := v.DumpStream(true)
			if err != nil {
				// A warning is about all we can do...
				log.Printf("WARN: Error saving webstream %s: %s\n", k, err)
			}
			if ok {
				log.Printf("Dumped room %s to filesystem\n", k)
			}
		}
	}
}*/
