package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getTestFilePath(parts ...string) string {
	pathparts := make([]string, len(parts)+1)
	pathparts[0] = "testfiles"
	copy(pathparts[1:], parts)
	return filepath.Join(pathparts...)
}

func TestAnyPathExists(t *testing.T) {
	whateverpath := getTestFilePath("whatever.txt")
	notpath := "veryNotPath"
	missingfilepath := getTestFilePath("whateverNOT.txt")
	exists, err := CheckAnyPathExists([]string{whateverpath})
	if err != nil {
		t.Fatalf("Error checking path exists: %s", err)
	}
	if !exists {
		t.Fatalf("Expected %s to exist", whateverpath)
	}
	exists, err = CheckAnyPathExists([]string{whateverpath, notpath})
	if err != nil {
		t.Fatalf("Error checking path exists: %s", err)
	}
	if !exists {
		t.Fatalf("Expected %s to exist in a group", whateverpath)
	}
	exists, err = CheckAnyPathExists([]string{whateverpath, missingfilepath})
	if err != nil {
		t.Fatalf("Error checking path exists: %s", err)
	}
	if !exists {
		t.Fatalf("Expected %s to exist in a group", whateverpath)
	}
	exists, err = CheckAnyPathExists([]string{notpath, missingfilepath})
	if err != nil {
		t.Fatalf("Error checking path exists: %s", err)
	}
	if exists {
		t.Fatalf("Expected nothing to exist in a group")
	}
}

func TestDetectTextfile(t *testing.T) {
	filepath := getTestFilePath("whatever.txt")
	f, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Couldn't open text file for testing: %s", err)
	}
	defer f.Close()
	typ, err := DetectContentType(f)
	if err != nil {
		t.Fatalf("Couldn't detect filetype: %s", err)
	}
	if strings.Index(typ, "text/plain") != 0 {
		t.Fatalf("Bad detection; expected text/plain, got: %s", typ)
	}
}

func TestDetectImagefile(t *testing.T) {
	filepath := getTestFilePath("testimage.png")
	f, err := os.Open(filepath)
	if err != nil {
		t.Fatalf("Couldn't open text file for testing: %s", err)
	}
	defer f.Close()
	typ, err := DetectContentType(f)
	if err != nil {
		t.Fatalf("Couldn't detect filetype: %s", err)
	}
	if strings.Index(typ, "image/png") != 0 {
		t.Fatalf("Bad detection; expected image/png, got: %s", typ)
	}
}
