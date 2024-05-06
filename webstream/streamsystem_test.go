package webstream

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
	//"log"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	DefaultCapacity = 1000
	GoroutineWait   = time.Millisecond
)

func reasonableConfig(name string) *Config {
	return &Config{
		StreamFolder:    utils.RandomTestFolder(name, false),
		TotalRoomLimit:  10,
		SingleDataLimit: 500,
		StreamDataLimit: 1000,
		ActiveRoomLimit: 10,
		RoomRegex:       "^[a-zA-Z0-9-]{3,256}$",
	}
}

func getSystem(t *testing.T, name string) (*testBacker, *Config, *WebStreamSystem) {
	config := reasonableConfig(name)
	backer, system := getSystemCustom(t, config)
	return backer, config, system
}

func getSystemCustom(t *testing.T, config *Config) (*testBacker, *WebStreamSystem) {
	backer := NewTestBacker()
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error while initializing new system: %s", err)
	}
	return backer, system
}

// ---- Now the actual tests ------

func TestNewSystem(t *testing.T) {
	_, _, system := getSystem(t, "newsys")
	count := system.RoomCount()
	if count != 0 {
		t.Fatalf("Expected no rooms in new system, got %d", count)
	}
}

func TestRoomRegexEmptyNonblockingRead(t *testing.T) {
	config := reasonableConfig("newsys")
	config.RoomRegex = "^[a-zA-Z0-9-]{3,256}$"
	_, system := getSystemCustom(t, config)
	_, err := system.ReadData("i", 0, -1, nil)
	_, is := err.(*RoomNameError)
	if !is {
		t.Errorf("Expected RoomNameError for too-short string, got %s", err)
	}
	_, err = system.ReadData("abcdef*", 0, -1, nil)
	_, is = err.(*RoomNameError)
	if !is {
		t.Errorf("Expected RoomNameError for special char string, got %s", err)
	}
	data, err := system.ReadData("abcdef", 0, -1, nil)
	if err != nil {
		t.Errorf("Expected no error for normal room string, got %s", err)
	}
	if len(data) != 0 {
		t.Errorf("Expected empty data for non-blocking read from empty room, got %d", len(data))
	}
}

func TestTotalRoomLimit(t *testing.T) {
	config := reasonableConfig("totallimit")
	config.TotalRoomLimit = 10
	config.ActiveRoomLimit = 100 // Must be larger so we don't run into it
	_, system := getSystemCustom(t, config)
	writedata := []byte("Just some data or whatever")
	for r := range 5 {
		for i := range 10 {
			room := fmt.Sprintf("heck%d", i)
			err := system.AppendData(room, writedata)
			if err != nil {
				t.Fatalf("Didn't expect any errors while appending rooms, got %s", err)
			}
			// Check to see if the data length matches what we expect
			info, err := system.RoomInfo(room)
			if err != nil {
				t.Fatalf("Got error while querying room info: %s", err)
			}
			expectedLength := (r + 1) * len(writedata)
			if info.Length != expectedLength {
				t.Fatalf("Unexpected data length: %d vs %d", info.Length, expectedLength)
			}
		}
	}
	// Now this room should fail
	err := system.AppendData("finalroom", []byte("Death"))
	_, is := err.(*RoomLimitError)
	if !is {
		t.Fatalf("Did not fail on too many rooms!")
	}
}

func TestActiveRoomLimit(t *testing.T) {
	config := reasonableConfig("activelimit")
	config.ActiveRoomLimit = 10
	config.TotalRoomLimit = 100 // Must be larger so we don't run into it
	config.IdleRoomTime = utils.Duration(time.Duration(time.Minute))
	_, system := getSystemCustom(t, config)
	for range 5 {
		for i := range 10 {
			err := system.AppendData(fmt.Sprintf("heck%d", i), []byte("Just some data or whatever"))
			if err != nil {
				t.Fatalf("Didn't expect any errors while appending rooms, got %s", err)
			}
		}
	}
	// Now this room should fail
	err := system.AppendData("finalroom", []byte("Death"))
	_, is := err.(*ActiveRoomLimitError)
	if !is {
		t.Fatalf("Did not fail on too many active rooms!")
	}
	// Go dump nothing (not forced)
	dumps := system.DumpStreams(false)
	if len(dumps) != 0 {
		t.Fatalf("Expected no dumps, got %d", len(dumps))
	}
	err = system.AppendData("finalroom", []byte("Death"))
	_, is = err.(*ActiveRoomLimitError)
	if !is {
		t.Fatalf("Did not fail on too many active rooms (2)!")
	}
	// Go dump everything(forced)
	dumps = system.DumpStreams(true)
	if len(dumps) != 10 {
		t.Fatalf("Expected 10 dumps, got %d", len(dumps))
	}
	err = system.AppendData("finalroom", []byte("Death"))
	if err != nil {
		t.Fatalf("Expected no failure on finalroom, got %s", err)
	}
	data, err := system.ReadData("finalroom", 0, -1, nil)
	if err != nil {
		t.Fatalf("Expected no failure on finalroom read, got %s", err)
	}
	if string(data) != "Death" {
		t.Fatalf("Data written not identical: expected Death, got %s", string(data))
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
	backer, _, system := getSystem(t, "simple")
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
	_, _, system := getSystem(t, "readempty")
	doRun := func(count int) {
		basicReadRoutine(t, fmt.Sprintf("junk%d", count), system, count, 0)
	}
	doRun(1)
	doRun(2)
	doRun(3)
	doRun(10)
}

func TestWebstreamReadFilledWait(t *testing.T) {
	_, _, system := getSystem(t, "readempty")
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

func TestExistingRooms(t *testing.T) {
	config := reasonableConfig("existingrooms")
	backer := NewTestBacker()
	// Add some crap to the backer. There was a regression where
	// we didn't iterate over every room, so we add a HUGE amount
	// (go uses random hashtables UGH)
	backer.Rooms["abc"] = []byte("It's easy or something")
	backer.Rooms["123"] = []byte("But instead it's horrible")
	for i := range 5000 {
		backer.Rooms[fmt.Sprintf("filler%d", i)] = []byte("nice")
	}
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error while initializing new system: %s", err)
	}
	// now, we should be able to immediately read from the room
	ctx, cancel := context.WithTimeout(context.Background(), GoroutineWait)
	defer cancel()
	data, err := system.ReadData("abc", 0, -1, ctx)
	if err != nil {
		t.Fatalf("Error while reading existing room: %s", err)
	}
	if !bytes.Equal(backer.Rooms["abc"], data) {
		t.Fatalf("Read data not the same as existing room, %s vs %s", string(data), string(backer.Rooms["abc"]))
	}
}

func TestDeadlockRegression(t *testing.T) {
	// There used to be a very simple way to deadlock: nonblocking read on empty room
	_, _, system := getSystem(t, "deadlockregression")
	data, err := system.ReadData("junk", 0, -1, nil)
	if err != nil {
		t.Fatalf("Not supposed to error out: %s", err)
	}
	if len(data) != 0 {
		t.Fatalf("Not supposed to get data")
	}
	// Now just do anything with the same room. In the website, we pull the room info
	done := make(chan struct{})
	var goerr error
	var info *WebStreamInfo
	go func() {
		info, goerr = system.RoomInfo("junk")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("Deadlock on empty nonblocking read")
	}
	if goerr != nil {
		t.Fatalf("Not supposed to error out: %s", err)
	}
	if info.Length != 0 || info.Capacity != 0 {
		t.Fatalf("Something weird happened with info")
	}
}

// if you only read from a room, it used to be that a dump would never occur,
// so it would always stay in memory
func TestNoDumpOnReadOnlyRegression(t *testing.T) {
	config := reasonableConfig("nodumpreadonly")
	backer := NewTestBacker()
	// Add some crap to the backer. There was a regression where
	// we didn't iterate over every room, so we add a HUGE amount
	// (go uses random hashtables UGH)
	backer.Rooms["abc"] = []byte("It's easy or something")
	system, err := NewWebStreamSystem(config, backer)
	if err != nil {
		t.Fatalf("Error while initializing new system: %s", err)
	}
	for range 3 {
		data, err := system.ReadData("abc", 0, -1, nil)
		if err != nil {
			t.Fatalf("Not supposed to error out: %s", err)
		}
		if !bytes.Equal(backer.Rooms["abc"], data) {
			t.Fatalf("Expected read %s, got %s", string(backer.Rooms["abc"]), string(data))
		}
		dumped := system.DumpStreams(true)
		if len(dumped) != 1 {
			t.Fatalf("Needed to dump exactly one room. See: %s", dumped)
		}
	}
}
