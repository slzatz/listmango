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
			case c := <-changedtickChan:
				for _, e := range sess.editors {
					if c.Buffer == e.vbuf {
						e.dirty++
						break
					}
				}
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

	switch sess.p.mode {

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

	case NORMAL, VISUAL, VISUAL_LINE, VISUAL_BLOCK: //actually handling NORMAL and INSERT
		switch c {

		case '\x1b':
			sess.p.command = ""
			//sess.p.repeat = 0
			//sess.p.mode = NORMAL

		case ':':
			sess.p.mode = COMMAND_LINE
			sess.p.command_line = ""
			sess.p.command = ""
			sess.p.showMessage(":")
			return false

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
				err := v.SetCurrentBuffer(sess.p.vbuf)
				if err != nil {
					sess.p.showMessage("Problem setting current buffer")
				}
				sess.p.mode = NORMAL
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
				sess.p.mode = NORMAL
				err := v.SetCurrentBuffer(sess.p.vbuf)
				if err != nil {
					sess.p.showMessage("Problem setting current buffer")
				}
			}

			return false

		} // end of switch in NORMAL

		/*
			if c == '+' {
				showMessage(v, messageBuf)
				return false
			}
		*/

		sess.p.showMessage("char = %d", c) //debugging
		sess.showOrgMessage("char = %d", c) //debugging
		// note below that arrow maps seem vim-specific
		/*****************************/
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
			v.FeedKeys("\x08", "t", true)
		default:
			_, err := v.Input(string(c))
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}

		mode, _ := v.Mode()
		sess.p.showMessage("blocking = %t; mode = %v", mode.Blocking, mode.Mode) //debugging

		if mode.Blocking == true {
      //ctrl w blocks
			sess.p.command += string(c)
      return false
    }

		if mode.Mode == "v" || mode.Mode == "V" || mode.Mode == string('\x16') {
			sess.showOrgMessage("mode: %v -> h0=%v; h1= %v", mode.Mode, highlightInfo(v)[0], highlightInfo(v)[1])
		}

    if mode.Mode == "i" {
    } else if mode.Mode == "n" {
      sess.p.mode = NORMAL
      if c != '\x1b' {
			  sess.p.command += string(c)
		    sess.p.showMessage("blocking = %t; mode = %v; command = %v", mode.Blocking, mode.Mode, sess.p.command) //debugging
		    if cmd, found := e_lookup2[sess.p.command]; found {
          switch cmd := cmd.(type) {
          case func(*Editor):
            cmd(sess.p)
          case func():
            cmd()
       }
			    //cmd(sess.p) 
			    _, err := v.Input("\x1b")
			    if err != nil {
			  	  fmt.Printf("%v\n", err)
			    }
       sess.p.command = ""   
        return false
      }
    } else {
      sess.p.command = ""
    }
    } else {
		switch mode.Mode {
		case "v":
			sess.p.mode = VISUAL
		case "V":
			sess.p.mode = VISUAL_LINE
		//case string('\x16'):
		case "\x16": //ctrl-v
			sess.p.mode = VISUAL_BLOCK
		}
		sess.p.vb_highlight = highlightInfo(v)
    }

    // may or may not be in middle of a command like caw or daw
		sess.p.rows = nil
		bb, _ := v.BufferLines(sess.p.vbuf, 0, -1, true)
		for _, b := range bb {
			sess.p.rows = append(sess.p.rows, string(b))
		}
		pos, _ := v.WindowCursor(w) //set screen cx and cy from pos

			//sess.p.showMessage(" => position = %v", pos) //debug
		sess.p.fr = pos[0] - 1
		sess.p.fc = pos[1]
		if c == 'u' && sess.p.mode == NORMAL {
			showMessage(v, messageBuf)
		}

		return true

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

		if sess.p.command == "\x17=" { //ctrl-w =
			sess.p.resize('=')
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		}

		if sess.p.command == "\x17_" { //ctrl-w _
			sess.p.resize('_')
			sess.p.command = ""
			sess.p.repeat = 0
			return false
		}

		if cmd, found := e_lookup[sess.p.command]; found {

			sess.p.prev_fr = sess.p.fr
			sess.p.prev_fc = sess.p.fc

			//sess.p.snapshot = sess.p.rows ////////////////////////////////////////////09182020

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

					/*
						}
						deleteBufferOpts := map[string]bool{
							"force":  true,
							"unload": false,
						}

						//err = v.DeleteBuffer(sess.p.vbuf, deleteBufferOpts)
						err = v.DeleteBuffer(0, deleteBufferOpts)
						if err != nil {
							sess.showOrgMessage("DeleteBuffer error %v", err)
						}
					*/

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
				} else if sess.p.dirty > 1 {
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
