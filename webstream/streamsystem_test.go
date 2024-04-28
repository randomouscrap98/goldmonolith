package webstream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	//"sync"
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
	// Now actually dump it
	// dumped, err = ws.DumpStream(true)
	// if err != nil {
	// 	t.Fatalf("Error dumping back to backing (clear): %s\n", err)
	// }
	// if !dumped {
	// 	t.Fatalf("Stream was supposed to be dumped, it was not")
	// }
	// if len(ws.stream) != 0 || cap(ws.stream) != 0 {
	// 	t.Fatalf("Stream not reset on dump\n")
	// }
	// length = ws.GetLength()
	// if length != len(sendData) {
	// 	t.Fatalf("Length cleared incorrectly")
	// }
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

/*func TestWebstreamInitial(t *testing.T) {
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
	dumped, err := ws.DumpStream(false)
	if err != nil {
		t.Fatalf("Error dumping back to backing: %s\n", err)
	}
	if !dumped {
		t.Fatalf("Stream was supposed to be dumped, it was not")
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
	dumped, err = ws.DumpStream(true)
	if err != nil {
		t.Fatalf("Error dumping back to backing (clear): %s\n", err)
	}
	if !dumped {
		t.Fatalf("Stream was supposed to be dumped, it was not")
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
	backer, err := NewFileBacker(reasonableConfig("testfilesimple")) //&WebStreamBacker_File{Config: reasonableConfig("testfilesimple")}
	if err != nil {
		t.Fatalf("Error when creating file backer: %s\n", err)
	}
	ws := NewWebStream("junk", backer)
	_ = basicStreamTest(t, ws)
}

func basicReadRoutine(t *testing.T, ws *WebStream, count int, offset int) {
	sendData := []byte("Yes indeed!")
	threadRead := make([][]byte, count)
	threadLock := make([]sync.Mutex, count)
	threadErr := make([]error, count)
	for i := range count {
		go func(index int) {
			tempRead, err := ws.ReadData(offset, -1, context.Background())
			threadLock[index].Lock()
			defer threadLock[index].Unlock()
			threadRead[index] = tempRead
			threadErr[index] = err
		}(i)
		time.Sleep(GoroutineWait)
	}
	// So, after a short bit, the reader should still be sitting around
	listenCount := ws.GetListenerCount()
	if listenCount != count {
		t.Fatalf("Listeners not registered! Expected %d, got %d\n", count, listenCount)
	}
	for i := range count {
		threadLock[i].Lock()
		if threadRead[i] != nil {
			t.Fatalf("The read thread didn't block on empty!\n")
		}
		threadLock[i].Unlock()
	}
	// Now, we send data. The reader should get unblocked
	err := ws.AppendData(sendData)
	if err != nil {
		t.Fatalf("Couldn't append data: %s\n", err)
	}
	time.Sleep(GoroutineWait * time.Duration(count))
	// It should now be over
	listenCount = ws.GetListenerCount()
	if listenCount != 0 {
		t.Fatalf("Listener still registered! Expected 0, got %d\n", listenCount)
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
	doRun := func(count int) {
		ws := NewWebStream(fmt.Sprintf("junk%d", count), backer)
		basicReadRoutine(t, ws, count, 0)
	}
	doRun(1)
	doRun(2)
	doRun(3)
	doRun(10)
}

func TestWebstreamReadFilledWait(t *testing.T) {
	backer := NewTestBacker()
	doRun := func(count int) {
		ws := NewWebStream(fmt.Sprintf("junk%d", count), backer)
		// Write some data
		junk := make([]byte, 55)
		err := ws.AppendData(junk)
		if err != nil {
			t.Fatalf("Write junk data failed: %s\n", err)
		}
		basicReadRoutine(t, ws, count, 55)
	}
	doRun(1)
	doRun(2)
	doRun(5)
}*/
