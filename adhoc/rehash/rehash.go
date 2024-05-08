package main

import (
	"os"
	"slices"
	//"github.com/randomouscrap98/goldmonolith/utils"
	"github.com/randomouscrap98/goldmonolith/kland"

	_ "github.com/mattn/go-sqlite3"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("Must pass the path to the kland data folder")
	}
	config := kland.Config{
		DataPath: os.Args[1],
	}
	db, err := config.OpenDb()
	must(err)

	ignoredBuckets := []string{
		kland.BucketSubject(""),
		kland.BucketSubject("chatDrawAnimations"),
	}

	// Go get the list of threads
	threads, err := kland.GetAllThreads(db)
	must(err)

	for _, t := range threads {
		if slices.Index(ignoredBuckets, t.Subject) >= 0 {
			continue
		}
	}
	// Need to load the data and the
}
