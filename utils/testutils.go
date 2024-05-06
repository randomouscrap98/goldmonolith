package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RandomTestFolder(name string, create bool) string {
	folder := filepath.Join("ignore", fmt.Sprintf("%s_%s",
		strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "", -1),
		name,
	))
	if create {
		os.MkdirAll(folder, 0750)
	}
	return folder
}
