package kland

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func GenerateRandomUniqueFilename(folder string, extension string) (string, error) {
	if len(extension) > 0 && extension[0] == '.' {
		extension = extension[1:]
	}
	if len(extension) == 0 {
		return "", fmt.Errorf("you must provide an extension")
	}
	// Maybe change this to generate more...
	lowerExt := strings.ToLower(extension)
	upperExt := strings.ToUpper(extension)
	// generate a valid file name (one that is not currently used)
	retries := 0
	var name string
	for {
		name = utils.RandomAsciiName(HashBaseCount + retries/HashIncreaseFactor)
		found, err := utils.CheckAnyPathExists([]string{
			filepath.Join(folder, fmt.Sprintf("%s.%s", name, lowerExt)),
			filepath.Join(folder, fmt.Sprintf("%s.%s", name, upperExt)),
		})
		if err != nil {
			return "", err
		}
		// Nothing found, that's good, it's a usable file
		if !found {
			break
		} else {
			log.Printf("Collision: %s (retries: %d)", name, retries)
		}
		retries += 1
	}
	// now we move the file and we're done
	filename := fmt.Sprintf("%s.%s", name, extension)
	return filename, nil
}
