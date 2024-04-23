package webstream

import (
	"testing"
	//"time"
	//"fmt"
)

// The beginning is all the test rigging and whatever. Don't
// want to write streams to the filesystem, so we do an in-memory thing.

type BackerEvent struct {
	Type int    // Read 0 write 1
	Data []byte // This is wasteful but like whatever
}

type TestBacker struct {
	Rooms  map[string][]byte
	Events []BackerEvent
}

func NewTestBacker() *TestBacker {
	return &TestBacker{
		Rooms:  make(map[string][]byte),
		Events: make([]BackerEvent, 0),
	}
}

func (tb *TestBacker) Write(name string, data []byte) error {
	tb.Rooms[name] = data
	tb.Events = append(tb.Events, BackerEvent{
		Type: 1,
		Data: data,
	})
	return nil
}

func (tb *TestBacker) Read(name string) ([]byte, error) {
	data, ok := tb.Rooms[name]
	tb.Events = append(tb.Events, BackerEvent{
		Type: 0,
		Data: data,
	})
	if !ok {
		return nil, nil
	}
	return data, nil
}

func TestWebstreamInitial(t *testing.T) {
	// Just want to create a webstream. Need a backer...
	backer := NewTestBacker()
	ws := NewWebStream("junk", backer)
	if ws.Name != "junk" {
		t.Errorf("Name not set!")
	}
	if ws.GetListenerCount() != 0 {
		t.Errorf("Nonzero listener count!")
	}
	if !ws.GetLastWrite().IsZero() {
		t.Errorf("Nonzero last write!")
	}
}
