package kland

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func reasonableConfig(name string) *Config {
	config := Config{}
	// Get a baseline config from toml
	rawconfig := GetDefaultConfig_Toml()
	err := toml.Unmarshal([]byte(rawconfig), &config)
	if err != nil {
		panic(err)
	}
	// Set some fields to test values
	config.DataPath = utils.RandomTestFolder(name, false)
	config.TemplatePath = filepath.Join("..", "cmd", config.TemplatePath)
	// WARN: You will need to change the above if the structure of the project changes
	return &config
}

func newTestContext(name string) *KlandContext {
	context, err := NewKlandContext(reasonableConfig(name))
	if err != nil {
		panic(err)
	}
	return context
}

func registerUpload(context *KlandContext, data []byte, t *testing.T) string {
	reader := utils.NewMemBuffer(data)
	name, err := context.RegisterUpload(&reader, ".png")
	if err != nil {
		t.Fatalf("Couldn't register upload: %s", err)
	}
	ext := filepath.Ext(name)
	if ext != ".png" {
		t.Fatalf("Incorrect extension on generated name: %s vs .png", ext)
	}
	// While we're at it, might as well make sure the file actually gets written
	fp := filepath.Join(context.config.ImagePath(), name)
	writedata, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Couldn't read writen file: %s", err)
	}
	if !bytes.Equal(writedata, data) {
		t.Fatalf("Written data differs from data given to Registerupload")
	}
	return fp
}

func registerUploadGenerate(context *KlandContext, length int, t *testing.T) ([]byte, string) {
	data := make([]byte, length)
	_, err := rand.Read(data)
	if err != nil {
		panic(err) // how does this even fail??
	}
	fp := registerUpload(context, data, t)
	return data, fp
}

func TestReasonableConfig(t *testing.T) {
	config := reasonableConfig("reasonableconfig")
	if config == nil {
		t.Fatalf("Didn't create config properly (nil)")
	}
}

func TestCreateContext(t *testing.T) {
	context := newTestContext("createcontext")
	checks := []string{
		context.config.TemplatePath,
		context.config.DatabasePath(),
		context.config.ImagePath(),
		context.config.TextPath(),
	}
	// Go check to see if various directories exist
	for _, check := range checks {
		fs, err := os.Stat(check)
		if err != nil {
			t.Fatalf("Missing expected filesystem entry: %s", check)
		}
		if fs.IsDir() && check != context.config.TemplatePath {
			// Check if dir empty
			entries, err := os.ReadDir(check)
			if err != nil {
				t.Fatalf("Error reading expected filesystem entry: %s", err)
			}
			if len(entries) != 0 {
				t.Fatalf("Data directory not empty: %s (%d)", check, len(entries))
			}
		}
	}
}

func TestFileUploadLimits(t *testing.T) {
	context := newTestContext("uploadlimits")
	context.config.MaxTotalDataSize = 0
	context.config.MaxTotalFileCount = 0
	// Write 4 files each of 1024 bytes. None of them should fail
	for range 4 {
		_, _ = registerUploadGenerate(context, 1024, t)
	}
	// Now change the requirements to be extremely restrictive
	context.config.MaxTotalDataSize = 5000
	reader := utils.NewMemBuffer(make([]byte, 1024))
	_, err := context.RegisterUpload(&reader, ".png")
	ooserr, is := err.(*utils.OutOfSpaceError)
	if !is {
		t.Fatalf("Expected error to be OutOfSpaceError")
	}
	if ooserr.Allowed != 5000 {
		t.Fatalf("Error didn't report correct allowed amount: %d", ooserr.Allowed)
	}
	if ooserr.Current < 4096 {
		t.Fatalf("Error didn't report correct current amount: %d", ooserr.Current)
	}
	context.config.MaxTotalDataSize = 0
	context.config.MaxTotalFileCount = 4
	reader = utils.NewMemBuffer(make([]byte, 1024))
	_, err = context.RegisterUpload(&reader, ".png")
	ooserr, is = err.(*utils.OutOfSpaceError)
	if !is {
		t.Fatalf("Expected error to be OutOfSpaceError")
	}
	if ooserr.Allowed != 4 {
		t.Fatalf("Error didn't report correct allowed amount: %d", ooserr.Allowed)
	}
	if ooserr.Current < 4 {
		t.Fatalf("Error didn't report correct current amount: %d", ooserr.Current)
	}
}
