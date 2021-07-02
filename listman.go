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
		for _, update := range updates {
			//s += fmt.Sprintf("%d: %v", i, update)
			handleRedraw(update)
		}
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

//func handleRedraw(updates [][]interface{}) {
func handleRedraw(update []interface{}) {
	//s := w.screen
	//for _, update := range updates {
	event := update[0].(string)
	args := update[1:]
	//editor.putLog("start   ", event)
	switch event {
	case "win_viewport":
		s, _ := args[0].([]interface{})
		//s := args[0]
		z := [4]int64{}
		z[0] = s[2].(int64)
		z[1] = s[3].(int64)
		z[2] = s[4].(int64)
		z[3] = s[5].(int64)
		sess.showOrgMessage("%d %v %v %v %T", z, s[3], s[4], s[5], s[2])
		sess.showOrgMessage("%v", z)
	}
	//}
}

/*
func handleRedraw(updates [][]interface{}) {
	//s := w.screen
	for _, update := range updates {
		event := update[0].(string)
		args := update[1:]
		//editor.putLog("start   ", event)
		switch event {
		// Global Events
		case "set_title":
			titleStr := (update[1].([]interface{}))[0].(string)
			sess.showEdMessage(title)
			//editor.window.SetupTitle(titleStr)
			//if runtime.GOOS == "linux" {
			//	editor.window.SetWindowTitle(titleStr)
			//}

		case "set_icon":
		case "mode_info_set":
			w.modeInfoSet(args)
			w.cursor.modeIdx = 0
		case "option_set":
			w.setOption(update)
		case "mode_change":
			arg := update[len(update)-1].([]interface{})
			w.mode = arg[0].(string)
			w.modeIdx = util.ReflectToInt(arg[1])
			if w.cursor.modeIdx != w.modeIdx {
				w.cursor.modeIdx = w.modeIdx
			}
			w.disableImeInNormal()
		case "mouse_on":
		case "mouse_off":
		case "busy_start":
		case "busy_stop":
		case "suspend":
		case "update_menu":
		case "bell":
		case "visual_bell":
		case "flush":
			w.flush()

		// Grid Events
		case "grid_resize":
			s.gridResize(args)
		case "default_colors_set":
			for _, u := range update[1:] {
				w.setColorsSet(u.([]interface{}))
			}
			// Show a window when connecting to the remote nvim.
			// The reason for handling the process here is that
			// in some cases, VimEnter will not occur if an error occurs in the remote nvim.
			if !editor.window.IsVisible() {
				if editor.opts.Ssh != "" {
					editor.window.Show()
				}
			}

		case "hl_attr_define":
			s.setHlAttrDef(args)
			// if goneovim own statusline is visible
			if w.drawStatusline {
				w.statusline.getColor()
			}
		case "hl_group_set":
			s.setHighlightGroup(args)
		case "grid_line":
			s.gridLine(args)
		case "grid_clear":
			s.gridClear(args)
		case "grid_destroy":
			s.gridDestroy(args)
		case "grid_cursor_goto":
			s.gridCursorGoto(args)
		case "grid_scroll":
			s.gridScroll(args)

		// Multigrid Events
		case "win_pos":
			s.windowPosition(args)
		case "win_float_pos":
			s.windowFloatPosition(args)
		case "win_external_pos":
			s.windowExternalPosition(args)
		case "win_hide":
			s.windowHide(args)
		case "win_scroll_over_start":
			// old impl
			// s.windowScrollOverStart()
		case "win_scroll_over_reset":
			// old impl
			// s.windowScrollOverReset()
		case "win_close":
			s.windowClose()
		case "msg_set_pos":
			s.msgSetPos(args)
		case "win_viewport":
			w.windowViewport(args)

		// Popupmenu Events
		case "popupmenu_show":
			if w.cmdline != nil {
				if w.cmdline.shown {
					w.cmdline.cmdWildmenuShow(args)
				}
			}
			if w.popup != nil {
				if w.cmdline != nil {
					if !w.cmdline.shown {
						w.popup.showItems(args)
					}
				} else {
					w.popup.showItems(args)
				}
			}
		case "popupmenu_select":
			if w.cmdline != nil {
				if w.cmdline.shown {
					w.cmdline.cmdWildmenuSelect(args)
				}
			}
			if w.popup != nil {
				if w.cmdline != nil {
					if !w.cmdline.shown {
						w.popup.selectItem(args)
					}
				} else {
					w.popup.selectItem(args)
				}
			}
		case "popupmenu_hide":
			if w.cmdline != nil {
				if w.cmdline.shown {
					w.cmdline.cmdWildmenuHide()
				}
			}
			if w.popup != nil {
				if w.cmdline != nil {
					if !w.cmdline.shown {
						w.popup.hide()
					}
				} else {
					w.popup.hide()
				}
			}
		// Tabline Events
		case "tabline_update":
			if w.tabline != nil {
				w.tabline.handle(args)
			}

		// Cmdline Events
		case "cmdline_show":
			if w.cmdline != nil {
				w.cmdline.show(args)
			}

		case "cmdline_pos":
			if w.cmdline != nil {
				w.cmdline.changePos(args)
			}

		case "cmdline_special_char":

		case "cmdline_char":
			if w.cmdline != nil {
				w.cmdline.putChar(args)
			}
		case "cmdline_hide":
			if w.cmdline != nil {
				w.cmdline.hide()
			}
		case "cmdline_function_show":
			if w.cmdline != nil {
				w.cmdline.functionShow()
			}
		case "cmdline_function_hide":
			if w.cmdline != nil {
				w.cmdline.functionHide()
			}
		case "cmdline_block_show":
		case "cmdline_block_append":
		case "cmdline_block_hide":

		// // -- deprecated events
		// case "wildmenu_show":
		// 	w.cmdline.wildmenuShow(args)
		// case "wildmenu_select":
		// 	w.cmdline.wildmenuSelect(args)
		// case "wildmenu_hide":
		// 	w.cmdline.wildmenuHide()

		// Message/Dialog Events
		case "msg_show":
			w.message.msgShow(args)
		case "msg_clear":
			w.message.msgClear()
		case "msg_showmode":
		case "msg_showcmd":
		case "msg_ruler":
		case "msg_history_show":
			w.message.msgHistoryShow(args)

		default:

		}
		editor.putLog("finished", event)
	}
}
*/
