package kland

import (
	"log"
	"os"
	"testing"
)

const (
	BigTestFolder = "ignore/manyfiles"
)

func TestGenerateRandomFile(t *testing.T) {
	log.Printf("NOTE: FOR THIS TEST TO WORK, you must have the folder %s", BigTestFolder)
	_, err := os.Stat(BigTestFolder)
	if err != nil {
		t.Fatalf("RandomFileTest: %s", err)
	}
	for range 1000 {
		_, err := GenerateRandomUniqueFilename(BigTestFolder, ".png")
		if err != nil {
			t.Fatalf("ERROR WHILE GENERATING UNIQUE FILENAME: %s", err)
		}
	}
}
