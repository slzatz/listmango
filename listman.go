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

func highlightInfo(v *nvim.Nvim) [2][4]int {
	var bufnum, lnum, col, off int
	var z [2][4]int
	v.Input("\x1bgv") //I need to send this but may be a problem

	err := v.Eval("getpos(\"'<\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		fmt.Printf("getpos error: %v", err)
	}
	//fmt.Printf("beginning: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[0] = [4]int{bufnum, lnum, col, off}

	err = v.Eval("getpos(\"'>\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		fmt.Printf("getpos error: %v\n", err)
	}
	//fmt.Printf("end: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[1] = [4]int{bufnum, lnum, col, off}

	return z
}

func showMessage(v *nvim.Nvim, buf nvim.Buffer) {
	//currentBuf, _ := v.CurrentBuffer()
	_ = v.SetCurrentBuffer(buf)
	//_ = v.FeedKeys("\x1bgg\"apqaq", "t", false)
	//_ = v.FeedKeys("\x1b\"apqaq\x1bi\r\x1b", "t", false)

	_ = v.SetBufferLines(buf, 0, -1, true, [][]byte{})
	//_ = v.FeedKeys("\x1bG\"apqaq", "t", false)
	_ = v.FeedKeys("\x1b\"apqaq", "t", false)
	bb, _ := v.BufferLines(buf, 0, -1, true)
	var message string
	var i int
	for i = len(bb) - 1; i >= 0; i-- {
		message = string(bb[i])
		if message != "" {
			break
		}
	}
	//_ = v.SetBufferLines(buf, 0, 0, true, [][]byte{})
	v.SetCurrentBuffer(sess.p.vbuf)
	currentBuf, _ := v.CurrentBuffer()
	if message != "" {
		//sess.showOrgMessage("len bb: %v; i: %v; message: %v", len(bb), i, message)
		sess.p.showMessage("len bb: %v; i: %v; message: %v", len(bb), i, message)
	} else {
		//sess.showOrgMessage("No message, %v %v %v", sess.p.vbuf, buf, currentBuf)
		//sess.showOrgMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
		sess.p.showMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
	}
}

// this doesn't work
func redirectMessages(v *nvim.Nvim) {
	//_, err := v.Input("\x1b:redir >> listman_messages.txt")
	//err := v.FeedKeys("\x1b:redir >> listman_messages.txt\r", "t", false)
	err := v.FeedKeys("\x1b:redir @a\r", "t", false)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	/*
		out, err := v.Exec("redir >> listman_messages.txt", true)
		if err != nil {
			fmt.Printf("messages error: %v", err)
		}
		return out
	*/
}

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

	redirectMessages(v)
	//messageBuf, _ := v.CurrentBuffer()
	messageBuf, _ := v.CreateBuffer(true, true)

	////////////////////////////////////////////////
	bufLinesChan := make(chan *BufLinesEvent)
	v.RegisterHandler("nvim_buf_lines_event", func(bufLinesEvent ...interface{}) {
		ev := &BufLinesEvent{
			Buffer:      bufLinesEvent[0].(nvim.Buffer),
			Changetick:  bufLinesEvent[1].(int64),
			FirstLine:   bufLinesEvent[2].(int64),
			LastLine:    bufLinesEvent[3].(int64),
			LineData:    fmt.Sprint(bufLinesEvent[4]),
			IsMultipart: bufLinesEvent[5].(bool),
		}
		bufLinesChan <- ev
	})

	// records changes that do not involve actual text changes
	changedtickChan := make(chan *ChangedtickEvent)
	v.RegisterHandler("nvim_buf_changedtick_event", func(changedtickEvent ...interface{}) {
		ev := &ChangedtickEvent{
			Buffer:     changedtickEvent[0].(nvim.Buffer),
			Changetick: changedtickEvent[1].(int64),
		}
		changedtickChan <- ev
	})

	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-changedtickChan:
			//case c := <-changedtickChan:
			//do nothing - these are not text changes
			/*
				for _, e := range sess.editors {
					if c.Buffer == e.vbuf {
						e.dirty++
						break
					}
				}
			*/
			case b := <-bufLinesChan:
				for _, e := range sess.editors {
					if b.Buffer == e.vbuf {
						e.dirty++
						break
					}
				}
			case <-quit:
				return

			}
		}
	}()

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
			textChange := editorProcessKey(k, messageBuf)

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

func editorProcessKey(c int, messageBuf nvim.Buffer) bool {

	sess.p.command += string(c)
	if strings.IndexAny(sess.p.command[0:1], "\x17\x08\x0c\x02\x05\x09\x06") == -1 {
		sess.p.command = ""
	} else {
		c = NOP
	}

	/*
		if c == '+' {
			showMessage(v, messageBuf)
			return false
		}
	*/

	sess.showOrgMessage("char = %d", c) //debugging
	var mode *nvim.Mode                 //may not be needed if not debugging
	// below are vim-specific maps
	/*****************************/
	if sess.p.mode != COMMAND_LINE {

		switch c {
		case ARROW_UP:
			v.FeedKeys("\x80ku", "t", true)
		case ARROW_DOWN:
			v.FeedKeys("\x80kd", "t", true)
		case ARROW_RIGHT:
			v.FeedKeys("\x80kr", "t", true)
		case ARROW_LEFT:
			v.FeedKeys("\x80kl", "t", true)
		case BACKSPACE:
			//v.FeedKeys("\x08", "t", true)
			v.FeedKeys("\x80kb", "t", true)
		case HOME_KEY:
			v.FeedKeys("\x80kh", "t", true)
		case DEL_KEY:
			v.FeedKeys("\x80kD", "t", true)
		case PAGE_UP:
			v.FeedKeys("\x80kP", "t", true)
		case PAGE_DOWN:
			v.FeedKeys("\x80kN", "t", true)
		case NOP:
			// this means we are intercepting
			// the key to run our own command
			// do nothing
		default:
			_, err := v.Input(string(c))
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}

		mode, _ = v.Mode()
		sess.p.showMessage("blocking = %t; mode = %v", mode.Blocking, mode.Mode) //debugging

		if mode.Blocking == true {
			// note that ctrl w blocks
			sess.p.command += string(c)
			return false
		}

		if mode.Mode == "v" || mode.Mode == "V" || mode.Mode == string('\x16') {
			sess.showOrgMessage("mode: %v -> h0=%v; h1= %v", mode.Mode, highlightInfo(v)[0], highlightInfo(v)[1])
		}

		sess.p.mode = modeMap[mode.Mode]
	}
	switch sess.p.mode {
	case INSERT:
		sess.p.showMessage("--INSERT--")
	case NORMAL:
		if c == ':' {
			sess.p.mode = COMMAND_LINE
			sess.p.command_line = ""
			sess.p.command = ""
			sess.p.showMessage(":")
			_, err := v.Input("\x1b")
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			return false
		}
		if c == '\x1b' {
			sess.p.command = ""
			//if previously in visual mode some text may be highlighted so need to return true
			// also need the cursor position because for example going from INSERT -> NORMAL causes cursor to move back
			// note you could fall through to getting pos but that recalcs rows which is unnecessary
			pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
			sess.p.fr = pos[0] - 1
			sess.p.fc = pos[1]
			return true
		}

		/*
			sess.p.command += string(c)
			if strings.IndexAny(sess.p.command[0:1], "\x17\x08\x0c\x02\x05\x09\x06") == -1 {
				sess.p.command = ""
			}
		*/

		sess.p.showMessage("blocking = %t; mode = %v; command = %v", mode.Blocking, mode.Mode, sess.p.command) //debugging
		if cmd, found := e_lookup2[sess.p.command]; found {
			switch cmd := cmd.(type) {
			case func(*Editor):
				cmd(sess.p)
			case func():
				cmd()
			case func(*Editor, int):
				cmd(sess.p, c)
			case func(*Editor) bool:
				cmd(sess.p)
			}

			_, err := v.Input("\x1b")
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			sess.p.command = ""
			return true
		}
	case VISUAL, VISUAL_LINE, VISUAL_BLOCK:
		sess.p.vb_highlight = highlightInfo(v)

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
					updateNote() //should be p->E_write_C(); closing_editor = true;

					/*
						ok, err := v.DetachBuffer(0)
						if err != nil {
							log.Fatal(err)
						}
						if !ok {
							log.Fatal()
						}
					*/

					//sess.p.quit <- struct{}{}

					// this seems like a kluge but I can't delete buffer
					// without generating an error
					err := v.SetBufferLines(0, 0, -1, true, [][]byte{})
					if err != nil {
						sess.showOrgMessage("SetBufferLines to []  error %v", err)
					}

				} else if cmd == "q!" || cmd == "quit!" {

					err := v.SetBufferLines(0, 0, -1, true, [][]byte{})
					if err != nil {
						sess.showOrgMessage("SetBufferLines to []  error %v", err)
					}
					/*
							deleteBufferOpts := map[string]bool{
								"force":  true,
								"unload": false,
							}
						//err = v.DeleteBuffer(0, deleteBufferOpts)
						//zero is the current buffer
						err = v.DeleteBuffer(0, map[string]bool{})
						if err != nil {
							sess.showOrgMessage("DeleteBuffer error %v", err)
						}
					*/

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
					// unless commented out earlier sess.p.quiet <- causes panic
					//sess.p = nil
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

			if cmd == "s" { //switch bufferd
				bufs, _ := v.Buffers()
				if int(sess.p.vbuf) == 2 {
					_ = v.SetCurrentBuffer(bufs[len(bufs)-1])
					sess.p.vbuf = bufs[len(bufs)-1]
				} else {
					_ = v.SetCurrentBuffer(bufs[1])
					sess.p.vbuf = bufs[1]
				}
				sess.p.command_line = ""
				sess.p.mode = NORMAL
				sess.p.refreshScreen(true)
				return true
			}

			if cmd == "m" {
				sess.p.showMessage("buffer %v has been modified %v times", sess.p.vbuf, sess.p.dirty)
				sess.p.command_line = ""
				sess.p.mode = NORMAL
				return false
			}

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
		} //end 'r'

		if c == DEL_KEY || c == BACKSPACE {
			if len(sess.p.command_line) > 0 {
				sess.p.command_line = sess.p.command_line[:len(sess.p.command_line)-1]
			}
		} else {
			sess.p.command_line += string(c)
		}

		sess.p.showMessage(":%s", sess.p.command_line)
		return false //end of case COMMAND_LINE
	} //end switch

	// may or may not be in middle of a command like caw or daw
	sess.p.rows = nil
	bb, _ := v.BufferLines(sess.p.vbuf, 0, -1, true)
	for _, b := range bb {
		sess.p.rows = append(sess.p.rows, string(b))
	}
	pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
	sess.p.fr = pos[0] - 1
	sess.p.fc = pos[1]
	if c == 'u' && sess.p.mode == NORMAL {
		showMessage(v, messageBuf)
	}
	return true
}
