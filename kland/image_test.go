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

func runConvertAnimation(filename string, outname string, t *testing.T) {
	fp := getTestFilePath(filename)
	rawanim, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Error reading %s: %s", fp, err)
	}
	op := filepath.Join(utils.RandomTestFolder("convertanimation", true), outname)
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

func TestConvertAnimation(t *testing.T) {
	runConvertAnimation("basicanim.txt", "out.gif", t)
	runConvertAnimation("basicanim_noloop.txt", "out_noloop.gif", t)
	runConvertAnimation("basicanim_weirdframe.txt", "out_weirdframe.gif", t)
}
