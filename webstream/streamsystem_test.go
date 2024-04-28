package webstream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	//"log"
)

const (
	DefaultCapacity = 1000
	GoroutineWait   = time.Millisecond
)

func randomFolder(name string, create bool) string {
	folder := filepath.Join("ignore", fmt.Sprintf("%s_%s",
		strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "", -1),
		name,
	))
	if create {
		os.MkdirAll(folder, 0750)
	}
	return folder
}

func reasonableConfig(name string) *Config {
	return &Config{
		StreamFolder:    randomFolder(name, false),
		TotalRoomLimit:  10,
		SingleDataLimit: 500,
		StreamDataLimit: 1000,
		ActiveRoomLimit: 10,
		RoomRegex:       "^[a-zA-Z0-9-]{3,256}$",
	}
}

// ---- Now the actual tests ------

func TestNewSystem(t *testing.T) {
	backer := NewTestBacker()
	config := reasonableConfig("newsys")
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error while initializing new system: %s", err)
	}
	count := system.RoomCount()
	if count != 0 {
		t.Fatalf("Expected no rooms in new system, got %d", count)
	}
}

func basicStreamTest(t *testing.T, room string, wsys *WebStreamSystem) []byte {
	sendData := []byte("Yes indeed!")
	err := wsys.AppendData(room, sendData)
	if err != nil {
		t.Fatalf("Couldn't write %s: %s\n", room, err)
	}
	info, err := wsys.RoomInfo(room)
	if err != nil {
		t.Fatalf("Couldn't get info for %s: %s", room, err)
	}
	if info.Length != len(sendData) {
		t.Fatalf("Length in %s unexpected: %d vs %d\n", room, info.Length, len(sendData))
	}
	data, err := wsys.ReadData(room, 0, -1, context.Background())
	if err != nil {
		t.Fatalf("Got error during read in %s: %s\n", room, err)
	}
	if !bytes.Equal(sendData, data) {
		t.Fatalf("Read and write in %s not equivalent! %s vs %s\n", room, string(data), string(sendData))
	}
	exists, err := Exists(wsys.backer, room)
	if err != nil {
		t.Fatalf("Got error during '%s' exists check: %s", room, err)
	}
	// Make sure it hasn't written it back yet
	if exists {
		t.Fatalf("Backing data for %s written before requested\n", room)
	}
	dumped := wsys.DumpStreams(true)
	if len(dumped) < 1 || dumped[0] != room {
		t.Fatalf("Stream %s was supposed to be dumped, it was not (see: %s)", room, dumped)
	}
	info, err = wsys.RoomInfo(room)
	if err != nil {
		t.Fatalf("Couldn't get after write info on room %s: %s", room, err)
	}
	if info.Length == 0 {
		t.Fatalf("Stream %s reset on dump incorrectly!\n", room)
	}
	if info.Capacity != 0 {
		t.Fatalf("Stream %s not cleared on dump!\n", room)
	}
	backdat, exists, err := wsys.backer.Read(room, DefaultCapacity)
	if err != nil {
		t.Fatalf("Error reading backing for room %s: %s\n", room, err)
	}
	if !exists {
		t.Fatalf("Expected room %s to exist in backing after dump\n", room)
	}
	if !bytes.Equal(backdat, sendData) {
		t.Fatalf("Backing and send not equivalent! %s vs %s\n", string(backdat), string(sendData))
	}
	return sendData
}

func TestWebstreamSimple(t *testing.T) {
	backer := NewTestBacker()
	config := reasonableConfig("simple")
	system, err := NewWebStreamSystem(config, backer)
	//ws := NewWebStream("junk", backer)
	sendData := basicStreamTest(t, "sim", system)
	eventLength := len(backer.Events)
	// Reading should still get us the data, it'll be refreshed
	data, err := system.ReadData("sim", 0, -1, context.Background())
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
	config := reasonableConfig("simplefile")
	backer, err := NewFileBacker(config.StreamFolder)
	if err != nil {
		t.Fatalf("Error when creating file backer: %s\n", err)
	}
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error creating webstream system: %s\n", err)
	}
	_ = basicStreamTest(t, "simfile", system)
}

func basicReadRoutine(t *testing.T, room string, wsys *WebStreamSystem, count int, offset int) {
	sendData := []byte("Yes indeed!")
	threadRead := make([][]byte, count)
	threadLock := make([]sync.Mutex, count)
	threadErr := make([]error, count)
	//roomname := func(i int) string { return fmt.Sprintf("read_%d", i) }
	for i := range count {
		go func(index int) {
			tempRead, err := wsys.ReadData(room, offset, -1, context.Background())
			threadLock[index].Lock()
			defer threadLock[index].Unlock()
			threadRead[index] = tempRead
			threadErr[index] = err
		}(i)
		time.Sleep(GoroutineWait)
	}
	// So, after a short bit, the reader should still be sitting around
	info, err := wsys.RoomInfo(room)
	if err != nil {
		t.Fatalf("Error getting room info for %s: %s", room, err)
	}
	if info.ListenerCount != count {
		t.Fatalf("Listeners not registered! Expected %d, got %d\n", count, info.ListenerCount)
	}
	for i := range count {
		threadLock[i].Lock()
		if threadRead[i] != nil {
			t.Fatalf("The read thread %d didn't block on empty in %s!\n", i, room)
		}
		threadLock[i].Unlock()
	}
	// Now, we send data. The reader should get unblocked
	err = wsys.AppendData(room, sendData)
	if err != nil {
		t.Fatalf("Couldn't append data int %s: %s\n", room, err)
	}
	time.Sleep(GoroutineWait * time.Duration(count))
	// It should now be over
	info, err = wsys.RoomInfo(room)
	if err != nil {
		t.Fatalf("Error getting room info for %s after write: %s", room, err)
	}
	if info.ListenerCount != 0 {
		t.Fatalf("Listener still registered! Expected 0, got %d\n", info.ListenerCount)
	}
	for i := range count {
		threadLock[i].Lock()
		if threadErr[i] != nil {
			t.Fatalf("Error while reading: %s\n", threadErr[i])
		}
		if !bytes.Equal(threadRead[i], sendData) {
			t.Fatalf("Bytes read not the same as bytes written: %s vs %s\n", string(threadRead[i]), string(sendData))
		}
		threadLock[i].Unlock()
	}
}

func TestWebstreamReadEmptyWait(t *testing.T) {
	backer := NewTestBacker()
	config := reasonableConfig("readempty")
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error creating webstream system: %s\n", err)
	}
	doRun := func(count int) {
		basicReadRoutine(t, fmt.Sprintf("junk%d", count), system, count, 0)
	}
	doRun(1)
	doRun(2)
	doRun(3)
	doRun(10)
}

func TestWebstreamReadFilledWait(t *testing.T) {
	backer := NewTestBacker()
	config := reasonableConfig("readfilled")
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error creating webstream system: %s\n", err)
	}
	doRun := func(count int) {
		room := fmt.Sprintf("junk%d", count)
		// Write some data
		junk := make([]byte, 55)
		err := system.AppendData(room, junk)
		if err != nil {
			t.Fatalf("Write junk data failed: %s\n", err)
		}
		basicReadRoutine(t, room, system, count, 55)
	}
	doRun(1)
	doRun(2)
	doRun(5)
}
