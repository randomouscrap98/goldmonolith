package makai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
	//"bytes"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

type ChatlogSearch struct {
	Search     string `schema:"search"`
	FileFilter string `schema:"filefilter"`
	Before     int    `schema:"before"`
	After      int    `schema:"after"`
}

func (mctx *MakaiContext) SearchChatlogs(search *ChatlogSearch, cancel context.Context) (string, string, error) {

	// Args for grep
	args := []string{"-InE", "-e", search.Search} //make([]string, 0, 20)

	if search.FileFilter != "" {
		if !mctx.chatlogIncludeRegex.Match([]byte(search.FileFilter)) {
			return "", "", fmt.Errorf("Bad characters in include, must be: %s", mctx.config.ChatlogIncludeRegex)
		}
		args = append(args, "--include="+search.FileFilter)
	}

	if search.After > 0 {
		args = append(args, "-A", fmt.Sprintf("%d", search.After))
	}
	if search.Before > 0 {
		args = append(args, "-B", fmt.Sprintf("%d", search.Before))
	}

	// Need to glob for files
	files, err := filepath.Glob(mctx.config.ChatlogFileGlob)
	if err != nil {
		return "", "", err
	}

	//var buffer bytes.Buffer
	//var errbuffer bytes.Buffer
	var result strings.Builder
	var errout strings.Builder

	for i := 0; i < len(files); i += mctx.config.ChatlogGrepChunk {
		fslice := files[i:min(len(files), i+mctx.config.ChatlogGrepChunk)]
		thisargs := slices.Concat(args, fslice)
		// thisargs := make([]string, len(args), len(args) + len(fslice))
		// copy(thisargs, args)
		// thisargs = append(thisargs, fslice...)
		cmd := exec.CommandContext(cancel, "grep", thisargs...)
		cmd.Stdout = &result
		cmd.Stderr = &errout
		if mctx.config.ChatlogLogging {
			log.Printf("Running grep: " + cmd.String())
		}
		err := cmd.Run()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				if ee.ExitCode() == 1 { // Grep is just funny like that...
					continue
				}
			}
			return "", "", err
		}
	}

	//var command = $"ls *.txt | xargs grep -InE -e {EscapeShellArg(search)} {incl}";

	return result.String(), errout.String(), nil
}

func (mctx *MakaiContext) WebSearchChatlogs(w http.ResponseWriter, r *http.Request) {
	query := ChatlogSearch{}
	err := mctx.decoder.Decode(&query, r.URL.Query())
	if err != nil {
		log.Printf("Error parsing query: %s", err)
		http.Error(w, "Can't parse query", http.StatusBadRequest)
		return
	}
	data := mctx.GetDefaultData(r)
	data["get"] = query
	data["oroot"] = mctx.config.RootPath + "/chatlog"
	data["chatlogurl"] = mctx.config.ChatlogUrl
	data["searchglob"] = mctx.config.ChatlogFileGlob
	data["grepchunk"] = mctx.config.ChatlogGrepChunk
	data["greptimeout"] = time.Duration(mctx.config.ChatlogMaxRuntime)
	if query.Search != "" {
		start := time.Now()
		cancel, cfunc := context.WithTimeout(r.Context(), time.Duration(mctx.config.ChatlogMaxRuntime))
		defer cfunc()
		output, errout, err := mctx.SearchChatlogs(&query, cancel)
		if err != nil {
			log.Printf("Error running chatlog search: %s", err)
			http.Error(w, "Some kind of command error", http.StatusInternalServerError)
			return
		}
		data["result"] = output
		data["error"] = errout
		data["time"] = time.Since(start).Seconds()
	}
	mctx.RunTemplate("chatlog_index.tmpl", w, data)
}
