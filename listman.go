package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

func ctrlKey(b byte) rune {
  return rune(b & 0x1f)
}

// SafeExit restores terminal using the original terminal config stored
// in the global session variable
func SafeExit(err error) {
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	if err1 := rawmode.Restore(s.OrigTermCfg); err1 != nil {
		fmt.Fprintf(os.Stderr, "Error: disabling raw mode: %s\r\n", err)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

var s = Session{}

func main_() {

	// parse config flags & parameters
	flag.Parse()
	filename := flag.Arg(0)

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}
	s.OrigTermCfg = origCfg

	s.editorMode = false

	// get the screen dimensions and create a view
	s.screenLines, s.screenCols, err := rawmode.GetWindowSize()
	if err != nil {
		SafeExit(fmt.Errorf("couldn't get window size: %v", err))
	}

	s.setStatusMessage("hello")

	for {
		//s.View.RefreshScreen(s.Editor, s.StatusMessage, s.Prompt)

		// read key
		k, err := terminal.ReadKey()
		if err != nil {
			SafeExit(fmt.Errorf("Error reading from terminal: %s", err))
		}

		if s.editorMode {
			editorProcessKey(k)
		} else {
			organizerProcessKey(k)
		}

		// if it's been 5 secs since the last status message, reset
		if time.Now().Sub(s.StatusMessageTime) > time.Second*5 && s.State == stateEditing {
			s.setStatusMessage("")
		}
	}
}

func organizerProcessKey(o *Organizer) {
	switch o.mode {

	case insert:
	case normal:
	case commandLine:

	}

func editorProcessKey(o *Editor) {
	switch o.mode {

	case insert:
	case normal:
	case commandLine:

	}
}
