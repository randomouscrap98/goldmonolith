package kland

import (
	"io"
	"log"
	"os"
)

// Close a temporary upload file. Also deletes the backing file if it exists
func CloseDeleteUploadFile(outfile io.ReadSeekCloser) {
	err := outfile.Close()
	if err != nil {
		log.Printf("CAN'T CLOSE UPLOAD TEMP FILE: %s", err)
		return
	}
	realfile, ok := outfile.(*os.File)
	if ok {
		err := os.Remove(realfile.Name())
		if err != nil {
			log.Printf("CAN'T DELETE UPLOAD TEMP FILE: %s", err)
		} else {
			log.Printf("Deleted underlying temp file: %s", realfile.Name())
		}
	} else {
		log.Printf("Temp file is in memory, no file delete")
	}
}
