package kland

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func getTestFilePath(parts ...string) string {
	pathparts := make([]string, len(parts)+1)
	pathparts[0] = "testfiles"
	copy(pathparts[1:], parts)
	return filepath.Join(pathparts...)
}

func TestConvertAnimation(t *testing.T) {
	fp := getTestFilePath("basicanim.txt")
	rawanim, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Error reading %s: %s", fp, err)
	}
	op := filepath.Join(utils.RandomTestFolder("convertanimation", true), "out.gif")
	outfile, err := os.Create(op)
	if err != nil {
		t.Fatalf("Error opening outfile %s: %s", op, err)
	}
	defer outfile.Close()
	err = ConvertAnimation(string(rawanim), outfile)
	if err != nil {
		t.Fatalf("Error creating animation: %s", err)
	}
	log.Printf("Wrote file to %s", op)
}
