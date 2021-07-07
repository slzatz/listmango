package main

import (
	"fmt"
	"strings"

	"github.com/neovim/go-client/nvim"
)

var note string

//["win_viewport", grid, win, topline, botline, curline, curcol]
type win_viewport struct {
	grid    int
	win     nvim.Window
	topline int
	botline int
	curline int
	curcol  int
}

var wvs []win_viewport

//["grid_cursor_goto", grid, row, column]
type grid_cursor_goto struct {
	grid   int
	row    int
	column int
}

var gcgs []grid_cursor_goto

//["mode_change", mode, mode_idx]
type mode_change struct {
	mode     string
	mode_idx int
}

var mcs []mode_change

type grid_line struct {
	grid      int
	row       int
	col_start int
	cells     []interface{}
}

var gls []grid_line

type update_line struct {
	line      string
	highlight []int
}

var uls []update_line

func ReflectToInt(iface interface{}) int {
	i, ok := iface.(int64)
	if ok {
		return int(i)
	}
	j, ok := iface.(uint64)
	if ok {
		return int(j)
	}
	k, ok := iface.(int)
	if ok {
		return int(k)
	}
	l, ok := iface.(uint)
	if ok {
		return int(l)
	}
	return 0
}

//func handleRedraw(updates [][]interface{}) {
// it gonevim it takes in a construced slice of the slice of empty interfaces
func handleRedraw(update []interface{}) {

	if p == nil {
		return
	}
	//s := w.screen
	//for _, update := range updates {
	event := update[0].(string)
	args := update[1:]
	//editor.putLog("start   ", event)
	var s string
	switch event {
	//["win_viewport", grid, win, topline, botline, curline, curcol]
	case "win_viewport":
		a, _ := args[0].([]interface{})

		var wv win_viewport

		wv.grid = ReflectToInt(a[0])
		wv.win = a[1].(nvim.Window)
		wv.topline = ReflectToInt(a[2])
		wv.botline = ReflectToInt(a[3])
		wv.curline = ReflectToInt(a[4])
		wv.curcol = ReflectToInt(a[5])

		wvs = append(wvs, wv)

		//s += fmt.Sprintf("win_viewport: %v", wv)
	case "grid_cursor_gotoz":
		a, _ := args[0].([]interface{})

		var gcg grid_cursor_goto
		gcg.grid = ReflectToInt(a[0])
		gcg.row = ReflectToInt(a[1])
		gcg.column = ReflectToInt(a[2])

		gcgs = append(gcgs, gcg)

		//s += fmt.Sprintf("  grid_cursor_goto: %v", z)
	case "mode_change":
		arg := update[len(update)-1].([]interface{})
		var mc mode_change
		mc.mode = arg[0].(string)
		mc.mode_idx = ReflectToInt(arg[1])
		//w.modeIdx = util.ReflectToInt(arg[1])
		mcs = append(mcs, mc)
		//s += fmt.Sprintf("  mode_change: %s; modeIdx: %d", mode, modeIdx)
	case "grid_line":
		for _, arg := range args {
			var gl grid_line
			gl.grid = ReflectToInt(arg.([]interface{})[0])
			//gl.row = ReflectToInt(arg.([]interface{})[1])
			//gl.col_start = ReflectToInt(arg.([]interface{})[2])
			//gl.cells = arg.([]interface{})[3].([]interface{})
			//gls = append(gls, gl)
			if gl.grid == 2 {
				gl.row = ReflectToInt(arg.([]interface{})[1])
				gl.col_start = ReflectToInt(arg.([]interface{})[2])
				gl.cells = arg.([]interface{})[3].([]interface{})
				gls = append(gls, gl)
				l, h := updateLine(gl.col_start, gl.row, gl.cells)
				s += fmt.Sprintf("row: %d; col %d; line: %s; highlight: %v", gl.row, gl.col_start, l, h)
				var ul update_line
				ul.line = l
				ul.highlight = h
				uls = append(uls, ul)
			}
		}
	case "flush":
		if (len(wvs) + len(gcgs) + len(mcs) + len(gls) + len(uls)) == 0 {
			return
		}
		n := "---------------------------------------\n"
		if len(wvs) > 0 {
			n += fmt.Sprintf("win_viewport: %v\n", wvs)
		}

		if len(gcgs) > 0 {
			n += fmt.Sprintf("grid_cursor_goto: %v\n", gcgs)
		}

		if len(mcs) > 0 {
			n += fmt.Sprintf("mode_change: %v\n", mcs)
		}

		if len(gls) > 0 {
			n += fmt.Sprintf("grid_line: %v\n", gls)
		}

		if len(uls) > 0 {
			n += fmt.Sprintf("update_line: %v\n", uls)
		}
		n += "---------------------------------------\n"

		//n := fmt.Sprintf("win_viewport: %v\ngrid_cursor_goto: %v\nmode_change: %v\ngrid_line: %v\nupdate_line: %v\n-------------------------------------------------\n", wvs, gcgs, mcs, gls, uls)
		//n += s
		n = generateWWString(n, org.totaleditorcols)
		note += n

		// use output window to look at nvim api messages
		if p == nil {
			return
		}
		op := p.output
		//op.rowOffset = 0 // specifically do not want this because it resets rowOffset
		op.rows = strings.Split(note, "\n")
		op.drawText()

		wvs = nil
		gcgs = nil
		mcs = nil
		//gls = nil
		//uls = nil
	}
}

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

func attachUIOption() map[string]interface{} {
	o := make(map[string]interface{})
	o["rgb"] = true
	// o["ext_multigrid"] = editor.config.Editor.ExtMultigrid
	o["ext_multigrid"] = true //////
	o["ext_hlstate"] = false  //// should revisit

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
