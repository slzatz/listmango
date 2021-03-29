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
	//"strings"

	"github.com/neovim/go-client/nvim"
	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

/*
func ctrlKey(b byte) rune {
  return rune(b & 0x1f)
}
*/

//var insert_cmds = map[string]struct{}{"I": z0, "i": z0, "A": z0, "a": z0, "o": z0, "O": z0, "s": z0, "cw": z0, "caw": z0}
//var file_cmds = map[string]struct{}{"savefile": z0, "save": z0, "readfile": z0, "read": z0}

//var move_only = map[string]struct{}{"w": z0, "e": z0, "b": z0, "0": z0, "$": z0, ":": z0, "*": z0, "n": z0, "[s": z0, "]s": z0, "z=": z0, "gg": z0, "G": z0, "yy": z0} //could put 'u' ctrl-r here

var sess Session
var org = Organizer{Session: &sess}
var p *Editor
var editors []*Editor

var v *nvim.Nvim
var w nvim.Window
var messageBuf nvim.Buffer

func highlightInfo_(v *nvim.Nvim) [2][4]int {
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

func showMessage_(v *nvim.Nvim, buf nvim.Buffer) {
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
	v.SetCurrentBuffer(p.vbuf)
	currentBuf, _ := v.CurrentBuffer()
	if message != "" {
		//sess.showOrgMessage("len bb: %v; i: %v; message: %v", len(bb), i, message)
		sess.showEdMessage("len bb: %v; i: %v; message: %v", len(bb), i, message)
	} else {
		//sess.showOrgMessage("No message, %v %v %v", sess.p.vbuf, buf, currentBuf)
		//sess.showOrgMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
		sess.showEdMessage("No message: len bb %v; Current Buf %v", len(bb), currentBuf)
	}
}

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
	messageBuf, _ = v.CreateBuffer(true, true)

	////////////////////////////////////////////////
	bufLinesChan := make(chan *BufLinesEvent)
	v.RegisterHandler("nvim_buf_lines_event", func(bufLinesEvent ...interface{}) {
		ev := &BufLinesEvent{
			Buffer: bufLinesEvent[0].(nvim.Buffer),
			//Changetick:  bufLinesEvent[1].(int64),
			Changetick:  bufLinesEvent[1], // .(int64)
			FirstLine:   bufLinesEvent[2], // .(int64)
			LastLine:    bufLinesEvent[3], // .(int64)
			LineData:    fmt.Sprint(bufLinesEvent[4]),
			IsMultipart: bufLinesEvent[5].(bool),
		}
		bufLinesChan <- ev
	})

	// records changes that do not involve actual text changes
	changedtickChan := make(chan *ChangedtickEvent)
	v.RegisterHandler("nvim_buf_changedtick_event", func(changedtickEvent ...interface{}) {
		ev := &ChangedtickEvent{
			Buffer: changedtickEvent[0].(nvim.Buffer),
			//Changetick: changedtickEvent[1].(int64),
			Changetick: changedtickEvent[1],
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
				for _, e := range editors {
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
	org.idToContext = make(map[int]string)
	org.folder_map = make(map[string]int)
	org.idToFolder = make(map[int]string)
	org.marked_entries = make(map[int]struct{})

	org.fts_titles = make(map[int]string)
	// ? where this should be.  Also in signal.
	sess.textLines = sess.screenLines - 2 - TOP_MARGIN // -2 for status bar and message bar
	//sess.divider = sess.screencols - sess.cfg.ed_pct * sess.screencols/100
	sess.divider = sess.screenCols - (60 * sess.screenCols / 100)
	sess.totaleditorcols = sess.screenCols - sess.divider - 1 // was 2

	generateContextMap()
	generateFolderMap()
	sess.eraseScreenRedrawLines()
	getItems(MAX)

	org.refreshScreen()
	org.drawStatusBar()
	sess.showOrgMessage("rows: %d  columns: %d", sess.screenLines, sess.screenCols)
	sess.returnCursor()
	sess.run = true

	for sess.run {

		key, err := terminal.ReadKey()
		if err != nil {
			sess.showOrgMessage("Readkey problem %w", err)
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

			scroll := p.scroll()
			redraw := textChange || scroll || p.redraw
			p.refreshScreen(redraw)
		} else {
			organizerProcessKey(k)
			org.scroll()
			org.refreshScreen()
			if sess.divider > 10 {
				org.drawStatusBar()
			}
		}
		sess.returnCursor()

		// if it's been 5 secs since the last status message, reset
		//if time.Now().Sub(sess.StatusMessageTime) > time.Second*5 && sess.State == stateEditing {
		//	sess.setStatusMessage("")
		//}
	}
	sess.quitApp()
}
