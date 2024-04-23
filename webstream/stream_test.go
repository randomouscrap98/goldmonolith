package webstream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	//"log"
)

const (
	DefaultCapacity = 1000
)

// The beginning is all the test rigging and whatever. Don't
// want to write streams to the filesystem, so we do an in-memory thing.

type BackerEvent struct {
	Type int    // Read 0 write 1
	Data []byte // This is wasteful but like whatever
}

type TestBacker struct {
	Capacity int
	Rooms    map[string][]byte
	Events   []BackerEvent
}

func NewTestBacker() *TestBacker {
	return &TestBacker{
		Capacity: DefaultCapacity,
		Rooms:    make(map[string][]byte),
		Events:   make([]BackerEvent, 0),
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
	if !ok { // This is a "new" room, so give it something...
		return make([]byte, 0, tb.Capacity), nil
	}
	return data, nil
}

func (tb *TestBacker) Exists(name string) bool {
	_, ok := tb.Rooms[name]
	return ok
}

func randomFolder(name string) string {
	folder := filepath.Join("ignore", fmt.Sprintf("%s_%s",
		strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "", -1),
		name,
	))
	os.MkdirAll(folder, 0750)
	return folder
}

func reasonableConfig(name string) *Config {
	return &Config{
		StreamFolder:    randomFolder(name),
		TotalRoomLimit:  10,
		SingleDataLimit: 500,
		StreamDataLimit: 1000,
		RoomRegex:       "^[a-zA-Z0-9-]{5,256}$",
	}
}

// ---- Now the actual tests ------

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
	if ws.GetLength() != 0 {
		t.Errorf("Nonzero length!")
	}
	if !ws.GetLastWrite().IsZero() {
		t.Errorf("Nonzero last write!")
	}
}

func basicStreamTest(t *testing.T, ws *WebStream) []byte {
	sendData := []byte("Yes indeed!")
	err := ws.AppendData(sendData)
	if err != nil {
		t.Fatalf("Couldn't write: %s\n", err)
	}
	length := ws.GetLength()
	if length != len(sendData) {
		t.Fatalf("Length unexpected: %d vs %d\n", length, len(sendData))
	}
	data, err := ws.ReadData(0, -1, context.Background())
	if err != nil {
		t.Fatalf("Got error during read: %s\n", err)
	}
	if !bytes.Equal(sendData, data) {
		t.Fatalf("Read and write not equivalent! %s vs %s\n", string(data), string(sendData))
	}
	// Make sure it hasn't written it back yet
	if ws.Backer.Exists("junk") {
		t.Fatalf("Backing data written before requested\n")
	}
	err = ws.DumpStream(false)
	if err != nil {
		t.Fatalf("Error dumping back to backing: %s\n", err)
	}
	if len(ws.stream) == 0 || cap(ws.stream) == 0 {
		t.Fatalf("Stream reset on dump incorrectly!\n")
	}
	backdat, err := ws.Backer.Read("junk") //backer.Rooms["junk"]
	if err != nil {
		t.Fatalf("Error reading backing: %s\n", err)
	}
	if !bytes.Equal(backdat, sendData) {
		t.Fatalf("Backing and send not equivalent! %s vs %s\n", string(backdat), string(sendData))
	}
	// Now actually dump it
	err = ws.DumpStream(true)
	if err != nil {
		t.Fatalf("Error dumping back to backing (clear): %s\n", err)
	}
	if len(ws.stream) != 0 || cap(ws.stream) != 0 {
		t.Fatalf("Stream not reset on dump\n")
	}
	length = ws.GetLength()
	if length != len(sendData) {
		t.Fatalf("Length cleared incorrectly")
	}
	return sendData
}

func TestWebstreamSimple(t *testing.T) {
	backer := NewTestBacker()
	ws := NewWebStream("junk", backer)
	sendData := basicStreamTest(t, ws)
	eventLength := len(backer.Events)
	// Reading should still get us the data, it'll be refreshed
	data, err := ws.ReadData(0, -1, context.Background())
	if err != nil {
		t.Fatalf("Got error during read after dump: %s\n", err)
	}
	if !bytes.Equal(sendData, data) {
		t.Fatalf("Read not equivalent after dump! %s vs %s\n", string(data), string(sendData))
	}
	newEventLength := len(backer.Events)
	if newEventLength != eventLength+1 {
		t.Fatalf("Somehow, read event not performed\n")
	}
}

func TestWebstreamFileSimple(t *testing.T) {
	backer := &WebStreamBacker_File{Config: reasonableConfig("testfilesimple")}
	ws := NewWebStream("junk", backer)
	_ = basicStreamTest(t, ws)
}
