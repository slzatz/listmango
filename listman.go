package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	//	"time"
	"strings"

	"github.com/neovim/go-client/nvim"
	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

/*
func ctrlKey(b byte) rune {
  return rune(b & 0x1f)
}
*/
var z0 = struct{}{}
var navigation = map[int]struct{}{
	ARROW_UP:    z0,
	ARROW_DOWN:  z0,
	ARROW_LEFT:  z0,
	ARROW_RIGHT: z0,
	'h':         z0,
	'j':         z0,
	'k':         z0,
	'l':         z0,
}

var insert_cmds = map[string]struct{}{"I": z0, "i": z0, "A": z0, "a": z0, "o": z0, "O": z0, "s": z0, "cw": z0, "caw": z0}
var quit_cmds = map[string]struct{}{"quit": z0, "q": z0, "quit!": z0, "q!": z0, "x": z0}
var file_cmds = map[string]struct{}{"savefile": z0, "save": z0, "readfile": z0, "read": z0}
var move_only = map[string]struct{}{"w": z0, "e": z0, "b": z0, "0": z0, "$": z0, ":": z0, "*": z0, "n": z0, "[s": z0, "]s": z0, "z=": z0, "gg": z0, "G": z0, "yy": z0} //could put 'u' ctrl-r here

var sess Session
var org Organizer

var v *nvim.Nvim
var w nvim.Window
var vimb [][]byte

func main() {

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, syscall.SIGWINCH)

	go func() {
		for {
			_ = <-signal_chan
			sess.signalHandler()
		}
	}()
	// parse config flags & parameters
	flag.Parse()

	// initialize neovim server
	ctx := context.Background()
	opts := []nvim.ChildProcessOption{

		// -u NONE is no vimrc and -n is no swap file
		nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed", "--headless", "--noplugin"),

		//without headless nothing happens but should be OK once ui attached.
		//nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed", "--noplugin"),

		nvim.ChildProcessContext(ctx),
		nvim.ChildProcessLogf(log.Printf),
	}

	/*
		if runtime.GOOS == "windows" {
			opts = append(opts, nvim.ChildProcessCommand("nvim.exe"))
		}
	*/

	var err error
	v, err = nvim.NewChildProcess(opts...)
	if err != nil {
		log.Fatal(err)
	}

	// Cleanup on return.
	defer v.Close()

	wins, err := v.Windows()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	w = wins[0]

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}
	sess.origTermCfg = origCfg

	sess.editorMode = false

	// get the screen dimensions and create a view
	sess.screenLines, sess.screenCols, err = rawmode.GetWindowSize()
	if err != nil {
		//SafeExit(fmt.Errorf("couldn't get window size: %v", err))
		os.Exit(1)
	}

	sess.showOrgMessage("hello")
	//filename := flag.Arg(0)

	org.cx = 0               //cursor x position
	org.cy = 0               //cursor y position
	org.fc = 0               //file x position
	org.fr = 0               //file y position
	org.rowoff = 0           //number of rows scrolled off the screen
	org.coloff = 0           //col the user is currently scrolled to
	org.sort = "modified"    //Entry sort column
	org.show_deleted = false //not treating these separately right now
	org.show_completed = true
	org.message = "" //displayed at the bottom of screen; ex. -- INSERT --
	org.highlight[0], org.highlight[1] = -1, -1
	org.mode = NORMAL
	org.last_mode = NORMAL
	org.command = ""
	org.command_line = ""
	org.repeat = 0 //number of times to repeat commands like x,s,yy also used for visual line mode x,y

	org.view = TASK
	org.taskview = BY_FOLDER
	org.folder = "todo"
	org.context = "No Context"
	org.keyword = ""

	org.context_map = make(map[string]int)
	org.folder_map = make(map[string]int)

	// ? where this should be.  Also in signal.
	sess.textLines = sess.screenLines - 2 - TOP_MARGIN // -2 for status bar and message bar
	//sess.divider = sess.screencols - sess.cfg.ed_pct * sess.screencols/100
	sess.divider = sess.screenCols - (60 * sess.screenCols / 100)
	sess.totaleditorcols = sess.screenCols - sess.divider - 1 // was 2

	generateContextMap()
	generateFolderMap()
	sess.eraseScreenRedrawLines()
	getItems(MAX)

	sess.refreshOrgScreen()
	sess.drawOrgStatusBar()
	sess.showOrgMessage("rows: %d  columns: %d", sess.screenLines, sess.screenCols)
	sess.returnCursor()
	sess.run = true

	for sess.run {

		// read key
		key, err := terminal.ReadKey()
		if err != nil {
			//SafeExit(fmt.Errorf("Error reading from terminal: %s", err))
			os.Exit(1)
		}

		var k int
		if key.Regular != 0 {
			k = int(key.Regular)
		} else {
			k = key.Special
		}

		if sess.editorMode {
			textChange := editorProcessKey(k)

			if !sess.editorMode {
				continue
			}
			scroll := sess.p.scroll()
			redraw := textChange || scroll || sess.p.redraw
			sess.p.refreshScreen(redraw)
		} else {
			organizerProcessKey(k)
			org.scroll()
			sess.refreshOrgScreen()
			sess.drawOrgStatusBar()
		}

		if sess.divider > 10 {
			sess.drawOrgStatusBar()
		}
		sess.returnCursor()

		// if it's been 5 secs since the last status message, reset
		//if time.Now().Sub(sess.StatusMessageTime) > time.Second*5 && sess.State == stateEditing {
		//	sess.setStatusMessage("")
		//}
	}
	sess.quitApp()
}

func organizerProcessKey(c int) {

	switch org.mode {

	case NO_ROWS:
		switch c {
		case ':':
			colon_N()
			return
		case '\x1b':
			org.command = ""
			org.repeat = 0
			return
		case 'i', 'I', 'a', 'A', 's':
			org.insertRow(0, "", true, false, false, BASE_DATE)
			org.mode = INSERT
			org.command = ""
			org.repeat = 0
			return
		}
		return

	case INSERT:
		switch c {
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
			org.moveCursor(c)
			return
		case '\x1b':
			org.command = ""
			org.mode = NORMAL
			if org.fc > 0 {
				org.fc--
			}
			sess.showOrgMessage("")
			return
		default:
			org.insertChar(c)
			return
		}

	case NORMAL:

		if c == '\x1b' {
			if org.view == TASK {
				sess.drawPreviewWindow(org.rows[org.fr].id)
			}
			sess.showOrgMessage("")
			org.command = ""
			org.repeat = 0
			return
		}

		/*leading digit is a multiplier*/
		//if (isdigit(c))  //equiv to if (c > 47 && c < 58)

		if (c > 47 && c < 58) && len(org.command) == 0 {

			if org.repeat == 0 && c == 48 {
			} else if org.repeat == 0 {
				org.repeat = c - 48
				return
			} else {
				org.repeat = org.repeat*10 + c - 48
				return
			}
		}

		if org.repeat == 0 {
			org.repeat = 1
		}

		org.command += string(c)

		if cmd, found := n_lookup[org.command]; found {
			cmd()
			org.command = ""
			org.repeat = 0
			return
		}

		//also means that any key sequence ending in something
		//that matches below will perform command

		// needs to be here because needs to pick up repeat
		//Arrows + h,j,k,l
		if _, found := navigation[c]; found {
			for j := 0; j < org.repeat; j++ {
				org.moveCursor(c)
			}
			org.command = ""
			org.repeat = 0
			return
		}

		return // end of case NORMAL

	case COMMAND_LINE:
		if c == '\x1b' {
			org.mode = NORMAL
			sess.showOrgMessage("")
			return
		}

		if c == '\r' {
			pos := strings.Index(org.command_line, " ")
			var s string
			if pos != -1 {
				s = org.command_line[:pos]
			} else {
				pos = 0
				s = org.command_line
			}
			if cmd, found := cmd_lookup[s]; found {
				cmd(pos)
				return
			}

			sess.showOrgMessage("\x1b[41mNot an outline command: %s\x1b[0m", s)
			org.mode = NORMAL
			return
		}

		if c == DEL_KEY || c == BACKSPACE {
			length := len(org.command_line)
			if length > 0 {
				org.command_line = org.command_line[:length-1]
			}
		} else {
			org.command_line += string(c)
		}

		sess.showOrgMessage(":%s", org.command_line)
		return //end of case COMMAND_LINE

	} // end switch o.mode
} // end func organizerProcessKey(c int)

func editorProcessKey(c int) bool {

	switch sess.p.mode {

	case NO_ROWS:
		switch c {
		case '\x1b':
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		case ':':
			sess.p.mode = COMMAND_LINE
			sess.p.command_line = ""
			sess.p.command = ""
			sess.p.showMessage(":")
			return false
		case 'i', 'I', 'a', 'A', 's', 'o', 'O':
			//p.editorInsertRow(0, std::string())
			sess.p.mode = INSERT
			sess.p.last_command = "i" //all the commands equiv to i
			sess.p.prev_fr = 0
			sess.p.prev_fc = 0
			sess.p.last_repeat = 1
			sess.p.snapshot = nil
			sess.p.snapshot = append(sess.p.snapshot, "")
			sess.p.showMessage("\x1b[1m-- INSERT --\x1b[0m")
			//p.command[0] = '\0'
			//p.repeat = 0
			// ? p.redraw = true
			return true
		}

	case INSERT:
		switch c {

		case '\r':
			sess.p.insertReturn()
			sess.p.last_typed += string(c)
			return true

		case HOME_KEY:
			sess.p.moveCursorBOL()
			return false

		case END_KEY:
			sess.p.moveCursorEOL()
			sess.p.moveCursor(ARROW_RIGHT)
			return false

		case BACKSPACE:
			sess.p.backspace()

			//not handling backspace correctly
			//when backspacing deletes more than currently entered text
			//A common case would be to enter insert mode  and then just start backspacing
			//because then dotting would actually delete characters
			//I could record a \b and then handle similar to handling \r
			length := len(sess.p.last_typed)
			if length > 0 {
				sess.p.last_typed = sess.p.last_typed[:length-1]
			}
			return true

		case DEL_KEY:
			sess.p.delChar()
			return true

		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
			sess.p.moveCursor(c)
			return false

		case ctrlKey('b'), ctrlKey('e'):
			//sess.p.push_current() //p.editorCreateSnapshot()
			//sess.p.editorDecorateWord(c)
			return true

		case '\x1b':

			/*Escape whatever else happens falls through to here*/
			sess.p.mode = NORMAL
			sess.p.repeat = 0

			if sess.p.fc > 0 {
				sess.p.fc--
			}

			sess.p.showMessage("")
			return false //end case x1b:

		// deal with tab in insert mode - was causing segfault
		case '\t':
			for i := 0; i < 4; i++ {
				sess.p.insertChar(' ')
			}
			return true

		default:
			sess.p.insertChar(c)
			sess.p.last_typed += string(c)
			return true

		} //end inner switch for outer case INSERT

		return true // end of case INSERT: - should not be executed

	case NORMAL:
		_, err := v.Input(string(c))
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		mode, _ := v.Mode() //status msg and branch if v
		sess.showOrgMessage("char = %v => mode = %v; blocking = %v", string(c), mode.Mode, mode.Blocking)
		if mode.Blocking == false {
			pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
			sess.p.showMessage(" => position = %v", pos)
		}

		z, _ := v.Bufferlines(vimb, 0, -1, true)
		for _, vv := range z {
			sess.p.rows[i] = string(vv)
		}

		switch c {

		case '\x1b':
			sess.p.command = ""
			sess.p.repeat = 0
			return false

		case ':':
			sess.p.mode = COMMAND_LINE
			sess.p.command_line = ""
			sess.p.command = ""
			sess.p.showMessage(":")
			return false

		case '/':
			sess.p.mode = SEARCH
			sess.p.command_line = ""
			sess.p.command = ""
			sess.p.showMessage("/")
			return false

			/*
			   case 'u':
			     sess.p.command = ""
			     sess.p.undo()
			     return true

			   case ctrlKey('r'):
			     sess.p.command = ""
			     sess.p.redo()
			     return true

			*/

		case ctrlKey('h'):
			sess.p.command = ""
			if len(sess.editors) == 1 {

				if sess.divider < 10 {
					sess.cfg.ed_pct = 80
					sess.moveDivider(80)
				}

				sess.editorMode = false //needs to be here

				sess.drawPreviewWindow(org.rows[org.fr].id)
				org.mode = NORMAL
				sess.returnCursor()
				return false
			}

			if sess.p.is_below {
				sess.p = sess.p.linked_editor
			}

			temp := []*Editor{}
			for _, e := range sess.editors {
				if !e.is_below {
					temp = append(temp, e)
				}
			}

			index := 0
			for i, e := range temp {
				if e == sess.p {
					index = i
					break
				}
			}

			sess.p.showMessage("index: %d; length: %d", index, len(temp))

			if index > 0 {
				sess.p = temp[index-1]
				if len(sess.p.rows) == 0 {
					sess.p.mode = NO_ROWS
				} else {
					sess.p.mode = NORMAL
				}
				return false
			} else {

				if sess.divider < 10 {
					sess.cfg.ed_pct = 80
					sess.moveDivider(80)
				}

				sess.editorMode = false //needs to be here

				sess.drawPreviewWindow(org.rows[org.fr].id)
				org.mode = NORMAL
				sess.returnCursor()
				return false
			}

		case ctrlKey('l'):
			sess.p.command = ""

			if sess.p.is_below {
				sess.p = sess.p.linked_editor
			}

			temp := []*Editor{}
			for _, e := range sess.editors {
				if !e.is_below {
					temp = append(temp, e)
				}
			}

			index := 0
			for i, e := range temp {
				if e == sess.p {
					index = i
					break
				}
			}

			sess.p.showMessage("index: %d; length: %d", index, len(temp))

			if index < len(temp)-1 {
				sess.p = temp[index+1]
				if len(sess.p.rows) == 0 {
					sess.p.mode = NO_ROWS
				} else {
					sess.p.mode = NORMAL
				}
			}

			return false

		case ctrlKey('j'):
			if sess.p.linked_editor.is_below {
				sess.p = sess.p.linked_editor
			}
			if len(sess.p.rows) == 0 {
				sess.p.mode = NO_ROWS
			} else {
				sess.p.mode = NORMAL
			}
			sess.p.command = ""
			return false

		case ctrlKey('k'):
			if sess.p.is_below {
				sess.p = sess.p.linked_editor
			}
			if len(sess.p.rows) == 0 {
				sess.p.mode = NO_ROWS
			} else {
				sess.p.mode = NORMAL
			}
			sess.p.command = ""
			return false

		} //end switch in NORMAL

		/*leading digit is a multiplier*/

		if (c > 47 && c < 58) && sess.p.command == "" {

			if sess.p.repeat == 0 && c == 48 {

			} else if sess.p.repeat == 0 {
				sess.p.repeat = c - 48
				// return false because command not complete
				return false
			} else {
				sess.p.repeat = sess.p.repeat*10 + c - 48
				// return false because command not complete
				return false
			}
		}

		if sess.p.repeat == 0 {
			sess.p.repeat = 1
		}
		sess.p.command += string(c)

		/* this and next if should probably be dropped
		 * and just use CTRL_KEY('w') to toggle
		 * size of windows and right now can't reach
		 * them given CTRL('w') above
		 */

		//if (std::string_view(p->command) == std::string({0x17,'='}))
		//if (p->command == std::string({0x17,'='}))
		if sess.p.command == "\x17=" {
			sess.p.resize('=')
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		}

		//if (std::string_view(p->command) == std::string({0x17,'_'}))
		//if (p->command == std::string({0x17,'_'}))
		if sess.p.command == "\x17_" {
			sess.p.resize('_')
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		}

		if cmd, found := e_lookup[sess.p.command]; found {

			sess.p.prev_fr = sess.p.fr
			sess.p.prev_fc = sess.p.fc

			sess.p.snapshot = sess.p.rows ////////////////////////////////////////////09182020

			cmd(sess.p, sess.p.repeat) //money shot

			if _, found := insert_cmds[sess.p.command]; found {
				sess.p.mode = INSERT
				sess.p.showMessage("\x1b[1m-- INSERT --\x1b[0m")
				sess.p.last_repeat = sess.p.repeat
				sess.p.last_command = sess.p.command //p->last_command must be a string
				sess.p.command = ""
				sess.p.repeat = 0
				return true
			} else if _, found := move_only[sess.p.command]; found {
				sess.p.command = ""
				sess.p.repeat = 0
				return false //note text did not change
			} else if sess.p.command != "." {
				sess.p.last_repeat = sess.p.repeat
				sess.p.last_command = sess.p.command
				//sess.p.push_current();
				sess.p.command = ""
				sess.p.repeat = 0
			} else { //if dot
				//if dot then just repeast last command at new location
				//sess.p->push_previous();
			}
		}

		// needs to be here because needs to pick up repeat
		//Arrows + h,j,k,l
		if _, found := navigation[c]; found {
			for i := 0; i < sess.p.repeat; i++ {
				sess.p.moveCursor(c)
			}
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		}

		return true // end of case NORMAL - there are breaks that can get to code above

	case COMMAND_LINE:

		if c == '\x1b' {
			sess.p.mode = NORMAL
			sess.p.command = ""
			sess.p.repeat, sess.p.last_repeat = 0, 0
			sess.p.showMessage("")
			return false
		}

		if c == '\r' {
			pos := strings.Index(sess.p.command_line, " ")
			var cmd string
			if pos != -1 {
				cmd = sess.p.command_line[:pos]
			} else {
				pos = 0
				cmd = sess.p.command_line
			}

			// note that right now we are not calling editor commands like E_write_close_C
			// and E_quit_C and E_quit0_C
			sess.showOrgMessage("You hit return and command is %v", cmd)
			if _, found := quit_cmds[cmd]; found {
				if cmd == "x" {
					if sess.p.is_subeditor {
						sess.p.mode = NORMAL
						sess.p.command = ""
						sess.p.command_line = ""
						sess.p.showMessage("You can't save the contents of the Output Window")
						return false
					}
					//update_note(false, true); //should be p->E_write_C(); closing_editor = true;
					updateNote() //should be p->E_write_C(); closing_editor = true;
				} else if cmd == "q!" || cmd == "quit!" {
					// do nothing = allow editor to be closed
				} else if sess.p.dirty > 0 {
					sess.p.mode = NORMAL
					sess.p.command = ""
					sess.p.command_line = ""
					sess.p.showMessage("No write since last change")
					return false
				}

				index := -1
				for i := range sess.editors {
					if sess.editors[i] == sess.p {
						index = i
						break
					}
				}
				copy(sess.editors[index:], sess.editors[index+1:])
				sess.editors = sess.editors[:len(sess.editors)-1]

				if sess.p.linked_editor != nil {
					index := -1
					for i := range sess.editors {
						if sess.editors[i] == sess.p.linked_editor {
							index = i
							break
						}
					}
					copy(sess.editors[index:], sess.editors[index+1:])
					sess.editors = sess.editors[:len(sess.editors)-1]
				}

				if len(sess.editors) > 0 {

					sess.p = sess.editors[0] //kluge should move in some logical fashion
					sess.positionEditors()
					sess.eraseRightScreen() //moved down here on 10-24-2020
					sess.drawEditors()

				} else { // we've quit the last remaining editor(s)
					sess.p = nil
					sess.editorMode = false
					sess.eraseRightScreen()

					if sess.divider < 10 {
						sess.cfg.ed_pct = 80
						sess.moveDivider(80)
					}

					sess.drawPreviewWindow(org.rows[org.fr].id)
					sess.returnCursor() //because main while loop if started in editor_mode -- need this 09302020
				}

				//sess.p.command_line = ""
				//sess.p.mode = NORMAL
				return false
			} //end quit_cmds

			if cmd0, found := e_lookup_C[cmd]; found {
				cmd0(sess.p)
				sess.p.command_line = ""
				sess.p.mode = NORMAL
				return false
			}

			sess.p.showMessage("\x1b[41mNot an editor command: %s\x1b[0m", cmd)
			sess.p.mode = NORMAL
			sess.p.command_line = ""
			return false
		}

		if c == DEL_KEY || c == BACKSPACE {
			if len(sess.p.command_line) > 0 {
				sess.p.command_line = sess.p.command_line[:len(sess.p.command_line)-1]
			}
		} else {
			sess.p.command_line += string(c)
		}

		sess.p.showMessage(":%s", sess.p.command_line)
		//sess.p.showMessage(":%s", "hello")
		//sess.showOrgMessage(":%s", sess.p.command_line)
		return false //end of case COMMAND_LINE
	}
	return false
}
