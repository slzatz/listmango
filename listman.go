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

/*
type editorState int

const (
	stateEditing editorState = iota
	stateSavePrompt
	stateQuitPrompt
	stateFindPrompt
	stateFindNav
)

const kiloVersion = "0.0.2"
*/
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

type Session struct {
	editorMode  bool
	editors     []editor
	screenLines int
	screenCols  int
	divider     int
	margins     int
	needSync    bool
}

var s = Session{}

func main_() {

	// parse config flags & parameters
	//flag.Parse()
	//filename := flag.Arg(0)

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}
	s.OrigTermCfg = origCfg

	s.editorMode = false

	// get the screen dimensions and create a view
	rows, cols, err := rawmode.GetWindowSize()
	if err != nil {
		SafeExit(fmt.Errorf("couldn't get window size: %v", err))
	}
	s.screenLines = rows
	s.screenCols = cols

	s.setStatusMessage(startMsg)
	s.State = stateEditing

	for {
		//s.View.RefreshScreen(s.Editor, s.StatusMessage, s.Prompt)

		// read key
		k, err := terminal.ReadKey()
		if err != nil {
			SafeExit(fmt.Errorf("Error reading from terminal: %s", err))
		}

		//s.Dispatch(k)
		if s.editorMode {
			editorProcessKey(&e)
		} else {
			organizerProcessKey(&o)
		}

		// if it's been 5 secs since the last status message, reset
		if time.Now().Sub(s.StatusMessageTime) > time.Second*5 && s.State == stateEditing {
			s.setStatusMessage("")
		}

	}
}

func editorProcessKey(e *editor) {
	switch e.mode {

	case insert:
	case normal:
	case commandLine:

	}
}

func organizerProcessKey(o *organizer) {
	switch o.mode {

	case insert:
	case normal:
	case commandLine:

	}
}
