package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/neovim/go-client/nvim"
	"github.com/slzatz/listmango/rawmode"
	"github.com/slzatz/listmango/terminal"
)

type Window interface {
	drawText()
	drawFrame()
	drawStatusBar()
}

var sess Session
var org = Organizer{Session: &sess}
var p *Editor
var editors []*Editor

//var windows []interface{}
var windows []Window

var v *nvim.Nvim
var w nvim.Window
var messageBuf nvim.Buffer

//var redrawUpdates chan [][]interface{}

var config *dbConfig
var db *sql.DB
var fts_db *sql.DB

func redirectMessages(v *nvim.Nvim) {
	err := v.FeedKeys("\x1b:redir @a\r", "t", false)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

// FromFile returns a dbConfig struct parsed from a file.
func FromFile(path string) (*dbConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg dbConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	//redrawUpdates := make(chan [][]interface{}, 1000)
	var err error
	config, err = FromFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	db, _ = sql.Open("sqlite3", config.Sqlite3.DB)
	fts_db, _ = sql.Open("sqlite3", config.Sqlite3.FTS_DB)

	sess.style = [7]string{"dracula", "fruity", "monokai", "native", "paraiso-dark", "rrt", "solarized-dark256"} //vim is dark but unusable
	sess.styleIndex = 2
	sess.imagePreview = false //image preview
	sess.imgSizeY = 800

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

	err = sess.GetWindowSize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting window size: %v", err)
		os.Exit(1)
	}
	// -2 for status bar and message bar
	sess.textLines = sess.screenLines - 2 - TOP_MARGIN
	//sess.divider = sess.screencols - sess.cfg.ed_pct * sess.screencols/100
	sess.divider = sess.screenCols - (60 * sess.screenCols / 100)
	sess.totaleditorcols = sess.screenCols - sess.divider - 1

	// initialize neovim server
	ctx := context.Background()
	opts := []nvim.ChildProcessOption{

		// -u NONE is no vimrc and -n is no swap file
		//nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed", "--headless", "--noplugin"),
		nvim.ChildProcessArgs("-u", "NONE", "-n", "--embed"),

		//without headless nothing happens but should be OK once ui attached.

		nvim.ChildProcessContext(ctx),
		nvim.ChildProcessLogf(log.Printf),
	}

	os.Setenv("VIMRUNTIME", "/home/slzatz/neovim/runtime")
	opts = append(opts, nvim.ChildProcessCommand("/home/slzatz/neovim/build/bin/nvim"))

	//var err error
	v, err = nvim.NewChildProcess(opts...)
	if err != nil {
		log.Fatal(err)
	}

	defer v.Close()

	v.RegisterHandler("Gui", func(updates ...interface{}) {
		sess.showEdMessage("len(updates) = %d", len(updates))
		for _, update := range updates {
			//          // handle update
			sess.showOrgMessage("Gui: %v", update)
		}
	})

	// probably map[string]interface{}
	/*
		[msg_showcmd [[]]]
		[grid_line [1 54 108 [[1 64]]]]
		grid_line [2 0 1 [[n 0] [o] [ ] [p] [e] [r] [a] [i] [n] [o] [ ]
		                                [p] [e] [r] [a] [i] [n] [o] [  0 1]]]]
		grid_scroll [2 1 54 0 126 1 0]]
		[win_viewport [2 Window:1000 0 5 0 0]]
		[grid_cursor_goto [2 0 0]]
		[mode_change [normal 0]]
		[flush []]
	*/

	v.RegisterHandler("redraw", func(updates ...[]interface{}) {
		//s := ""
		sess.showEdMessage("len(updates) = %d", len(updates))
		for _, update := range updates {
			//s += fmt.Sprintf("%d: %v", i, update)
			handleRedraw(update)
		}

		//redrawUpdates <- updates
		//sess.showOrgMessage("redraw: %v", s)
	})

	err = v.AttachUI(sess.totaleditorcols, sess.textLines, attachUIOption())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching UI: %v", err)
		os.Exit(1)
	}

	wins, err := v.Windows()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	w = wins[0]

	redirectMessages(v)
	messageBuf, _ = v.CreateBuffer(true, true)

	// enable raw mode
	origCfg, err := rawmode.Enable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v", err)
		os.Exit(1)
	}

	sess.origTermCfg = origCfg

	sess.editorMode = false

	/*
		err = sess.GetWindowSize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting window size: %v", err)
			os.Exit(1)
		}
	*/

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
	org.repeat = 0 //number of times to repeat commands like x,s,yy ? also used for visual line mode x,y

	org.view = TASK
	org.taskview = BY_FOLDER
	//org.filter = "todo"
	org.filter = "No Folder"
	org.context_map = make(map[string]int)
	org.idToContext = make(map[int]string)
	org.folder_map = make(map[string]int)
	org.idToFolder = make(map[int]string)
	org.marked_entries = make(map[int]struct{})
	org.keywordMap = make(map[string]int)

	// ? where this should be.  Also in signal.
	sess.textLines = sess.screenLines - 2 - TOP_MARGIN // -2 for status bar and message bar
	//sess.divider = sess.screencols - sess.cfg.ed_pct * sess.screencols/100
	sess.divider = sess.screenCols - (60 * sess.screenCols / 100)
	sess.totaleditorcols = sess.screenCols - sess.divider - 1 // was 2

	generateContextMap()
	generateFolderMap()
	generateKeywordMap()
	sess.eraseScreenRedrawLines()
	org.rows = filterEntries(org.taskview, org.filter, org.show_deleted, org.sort, MAX)
	if len(org.rows) == 0 {
		sess.showOrgMessage("No results were returned")
		org.mode = NO_ROWS
	}
	org.drawPreview()
	org.refreshScreen()
	org.drawStatusBar()
	sess.showOrgMessage("rows: %d  columns: %d", sess.screenLines, sess.screenCols)
	sess.returnCursor()
	sess.run = true

	err = os.RemoveAll("temp")
	if err != nil {
		sess.showOrgMessage("Error deleting temp directory: %v", err)
	}
	err = os.Mkdir("temp", 0700)
	if err != nil {
		sess.showOrgMessage("Error creating temp directory: %v", err)
	}

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
			textChange := editorProcessKey(k)

			if !sess.editorMode {
				continue
			}

			if textChange {
				p.scroll()
				p.drawText()
				p.drawStatusBar()
			}
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

func attachUIOption() map[string]interface{} {
	o := make(map[string]interface{})
	o["rgb"] = true
	// o["ext_multigrid"] = editor.config.Editor.ExtMultigrid
	o["ext_multigrid"] = true
	o["ext_hlstate"] = true

	apiInfo, err := v.APIInfo()
	if err == nil {
		for _, item := range apiInfo {
			i, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// k => string v => interface
			for k, v := range i {
				if k != "ui_events" {
					continue
				}
				events, ok := v.([]interface{})
				if !ok {
					continue
				}
				for _, event := range events {
					function, ok := event.(map[string]interface{})
					if !ok {
						continue
					}
					name, ok := function["name"]
					if !ok {
						continue
					}

					switch name {
					// case "wildmenu_show" :
					// 	o["ext_wildmenu"] = editor.config.Editor.ExtCmdline
					case "cmdline_show":
						//o["ext_cmdline"] = editor.config.Editor.ExtCmdline
						o["ext_cmdline"] = true
					case "msg_show":
						//o["ext_messages"] = editor.config.Editor.ExtMessages
						o["ext_messages"] = true
					case "popupmenu_show":
						//o["ext_popupmenu"] = editor.config.Editor.ExtPopupmenu
						o["ext_popupmenu"] = false
					case "tabline_update":
						//o["ext_tabline"] = editor.config.Editor.ExtTabline
						o["ext_tabline"] = false
					case "win_viewport":
						//w.api5 = true
					}
				}
			}
		}
	}

	return o
}

/*
//func handleRedraw(updates [][]interface{}) {
func handleRedraw(update []interface{}) {
	//s := w.screen
	//for _, update := range updates {
	event := update[0].(string)
	args := update[1:]
	//editor.putLog("start   ", event)
	var s string
	switch event {
	//["win_viewport", grid, win, topline, botline, curline, curcol]
	case "win_viewportz":
		a, _ := args[0].([]interface{})
		//s := args[0]
		z := [6]int64{}
		z[0] = a[0].(int64)
		//z[1] = a[1].(nvim.Window)
		z[2] = a[2].(int64)
		z[3] = a[3].(int64)
		z[4] = a[4].(int64) //curline
		z[5] = a[5].(int64) //curcol
		//sess.showOrgMessage("%d %v %v %v %T", z, s[3], s[4], s[5], s[2])
		//sess.showOrgMessage("win_viewport: %v", z)
		s += fmt.Sprintf("win_viewport: %v", z)
	//["grid_cursor_goto", grid, row, column]
	case "grid_cursor_gotoz":
		a, _ := args[0].([]interface{})
		z := [3]int64{}
		z[0] = a[0].(int64)
		z[1] = a[1].(int64)
		z[2] = a[2].(int64)
		s += fmt.Sprintf("  grid_cursor_goto: %v", z)
	case "mode_changez":
		arg := update[len(update)-1].([]interface{})
		mode := arg[0].(string)
		modeIdx := arg[1].(int64)
		//w.modeIdx = util.ReflectToInt(arg[1])
		s += fmt.Sprintf("  mode_change: %s; modeIdx: %d", mode, modeIdx)
	case "grid_line":
		for _, arg := range args {
			gridid := ReflectToInt(arg.([]interface{})[0])
			row := ReflectToInt(arg.([]interface{})[1])
			colStart := ReflectToInt(arg.([]interface{})[2])
			cells := arg.([]interface{})[3].([]interface{})
			if gridid == 2 {
				l, h := updateLine(colStart, row, cells)
				//updateGridContent(row, colStart, arg.([]interface{})[3].([]interface{}))
				//s += fmt.Sprintf("gridid: %d; row: %d; col_start: %d; cells: %v", gridid, row, colStart, cells)
				s += fmt.Sprintf("row: %d; col %d; line: %s; highlight: %v", row, colStart, l, h)
			}
		}
	}
	if s != "" {
		sess.showOrgMessage(s)
	}
	//}
}
*/

/*
func (win *Window) updateGridContent(row, colStart int, cells []interface{}) {
	if colStart < 0 {
		return
	}

	//if row >= win.rows {
	if row >= sess.textLines {
		return
	}

	// Suppresses flickering during smooth scrolling
	//if win.scrollPixels[1] != 0 {
	//	win.scrollPixels[1] = 0
//	}

	// We should control to draw statusline, vsplitter
	if editor.config.Editor.DrawWindowSeparator && win.grid == 1 {

		isSkipDraw := true
		if win.s.name != "minimap" {

			// Draw  bottom statusline
			if row == win.rows-2 {
				isSkipDraw = false
			}
			// Draw tabline
			if row == 0 {
				isSkipDraw = false
			}

			// // Do not Draw statusline of splitted window
			// win.s.windows.Range(func(_, winITF interface{}) bool {
			// 	w := winITF.(*Window)
			// 	if w == nil {
			// 		return true
			// 	}
			// 	if !w.isShown() {
			// 		return true
			// 	}
			// 	if row == w.pos[1]-1 {
			// 		isDraw = true
			// 		return false
			// 	}
			// 	return true
			// })
		} else {
			isSkipDraw = false
		}

		if isSkipDraw {
			return
		}
	}

	win.updateLine(colStart, row, cells)
	win.countContent(row)
	win.makeUpdateMask(row)
	if !win.isShown() {
		win.show()
	}

	if win.isMsgGrid {
		return
	}
	if win.grid == 1 {
		return
	}
	if win.maxLenContent < win.lenContent[row] {
		win.maxLenContent = win.lenContent[row]
	}
}
*/

/*
func updateLine(col, row int, cells []interface{}) (line string, highlight []int) {
	//line := ""
	//highlight := []int64{}
	//colStart := col
	for _, arg := range cells {
		cell := arg.([]interface{})

		var hl, repeat int

		hl = -1
		text := cell[0]
		if len(cell) >= 2 {
			hl = ReflectToInt(cell[1])
		}

		if len(cell) == 3 {
			repeat = ReflectToInt(cell[2])
		}

		// If `repeat` is present, the cell should be
		// repeated `repeat` times (including the first time), otherwise just
		// once.
		r := 1
		if repeat == 0 {
			repeat = 1
		}
		for r <= repeat {

			line += text.(string)

			// If `hl_id` is not present the most recently seen `hl_id` in
			//	the same call should be used (it is always sent for the first
			//	cell in the event).
			switch col {
			case 0:
				//line[col].highlight = w.s.hlAttrDef[hl]
				highlight = append(highlight, hl)
			default:
				if hl == -1 {
					//line[col].highlight = line[col-1].highlight
					highlight = append(highlight, highlight[len(highlight)-1])
				} else {
					//line[col].highlight = w.s.hlAttrDef[hl]
					highlight = append(highlight, hl)
				}
			}
			col++
			r++
		}
	}
	//w.updateMutex.Unlock()

	//w.queueRedraw(colStart, row, col-colStart+1, 1)
	return
}
*/
