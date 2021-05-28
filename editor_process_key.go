package main

import (
	"fmt"
	"strings"

	//"sync/atomic"

	"github.com/neovim/go-client/nvim"
)

var termcodes = map[int]string{
	ARROW_UP:    "\x80ku",
	ARROW_DOWN:  "\x80kd",
	ARROW_RIGHT: "\x80kr",
	ARROW_LEFT:  "\x80kl",
	BACKSPACE:   "\x80kb", //? also works "\x08"
	HOME_KEY:    "\x80kh",
	DEL_KEY:     "\x80kD",
	PAGE_UP:     "\x80kP",
	PAGE_DOWN:   "\x80kN",
}

var quit_cmds = map[string]struct{}{"quit": z0, "q": z0, "quit!": z0, "q!": z0, "x": z0}

func highlightInfo(v *nvim.Nvim) [2][4]int {
	var bufnum, lnum, col, off int
	var z [2][4]int
	v.Input("\x1bgv") //I need to send this but may be a problem

	err := v.Eval("getpos(\"'<\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		sess.showOrgMessage("getpos error: %v", err)
	}
	//fmt.Printf("beginning: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[0] = [4]int{bufnum, lnum, col, off}

	err = v.Eval("getpos(\"'>\")", []*int{&bufnum, &lnum, &col, &off})
	if err != nil {
		sess.showOrgMessage("getpos error: %v", err)
	}
	//fmt.Printf("end: bufnum = %v; lnum = %v; col = %v; off = %v\n", bufnum, lnum, col, off)
	z[1] = [4]int{bufnum, lnum, col, off}

	return z
}

//note that bool returned is whether to redraw which will freeze program
//in BufferLines if mode is blocking
func editorProcessKey(c int) bool { //bool returned is whether to redraw

	//No matter what mode you are in an escape puts you in NORMAL mode
	if c == '\x1b' {
		_, err := v.Input("\x1b")
		if err != nil {
			sess.showEdMessage("Error input escape: %v", err)
			return false
		}
		p.command = ""
		p.command_line = ""

		if p.mode == PREVIEW_MARKDOWN {
			// don't need to check WindowCursor - no change in pos
			fmt.Print("\x1b_Ga=d\x1b\\") //delete any images
			sess.showEdMessage("")
			p.mode = NORMAL
			return true
		}

		p.mode = NORMAL

		/*
			if previously in visual mode some text may be highlighted so need to return true
			 also need the cursor position because for example going from INSERT -> NORMAL causes cursor to move back
			 note you could fall through to getting pos but that recalcs rows which is unnecessary
		*/

		pos, _ := v.WindowCursor(w) //set screen cx and cy from pos
		p.fr = pos[0] - 1
		p.fc = pos[1]
		sess.showEdMessage("")
		return true
	}
	/*
		 there are a set of commands like ctrl-w that we are intercepting
		note any command that changes the UI like splits or tabs doesn't make sense
		Also note that the if below falls through if p.command is "" and the character isn't
		one of the one that starts a command
	*/
	if p.mode == PREVIEW_MARKDOWN {
		switch c {
		case PAGE_DOWN, ARROW_DOWN, 'j':
			p.previewLineOffset++
		case PAGE_UP, ARROW_UP, 'k':
			if p.previewLineOffset > 0 {
				p.previewLineOffset--
			}
		}
		p.drawPreview()
		return false
	}

	if p.mode == NORMAL {
		if len(p.command) == 0 {
			if strings.IndexAny(string(c), "\x17\x08\x0c\x02\x05\x09\x06 ") != -1 {
				p.command = string(c)
			}
		} else {
			p.command += string(c)
		}

		if len(p.command) > 0 {
			if cmd, found := e_lookup2[p.command]; found {
				switch cmd := cmd.(type) {
				case func(*Editor):
					cmd(p)
				case func():
					cmd()
				case func(*Editor, int):
					cmd(p, c)
				case func(*Editor) bool:
					cmd(p)
				}
				// seems to be necessary at least for certain commands
				_, err := v.Input("\x1b")
				if err != nil {
					sess.showEdMessage("%v", err)
				}
				if p.command == " m" {
					p.command = ""
					return false
				}
				p.command = ""
				p.bb, _ = v.BufferLines(p.vbuf, 0, -1, true) //reading updated buffer
				pos, _ := v.WindowCursor(w)                  //screen cx and cy set from pos
				p.fr = pos[0] - 1
				p.fc = pos[1]
				return true
			} else {
				return false
			}
		}
	}

	if p.mode == EX_COMMAND {
		//don't send keys to nvim - don't want it processing them
		//sess.showEdMessage("NOP or COMMAND_LINE or SEARCH - %q", p.mode)
		if c == '\r' {
			pos := strings.Index(p.command_line, " ")
			var cmd string
			if pos != -1 {
				cmd = p.command_line[:pos]
			} else {
				pos = 0
				cmd = p.command_line
			}

			// note that right now we are not calling editor commands like E_write_close_C
			// and E_quit_C and E_quit0_C
			//sess.showOrgMessage("You hit return and command is %v", cmd) //debugging
			if _, found := quit_cmds[cmd]; found {
				if cmd == "x" {
					if p.is_subeditor {
						p.mode = NORMAL
						p.command = ""
						p.command_line = ""
						sess.showEdMessage("You can't save the contents of the Output Window")
						return false
					}
					updateNote(p)
					/*
						deleteBufferOpts := map[string]bool{
							"force":  true,
							"unload": false,
						}
						//err := v.DeleteBuffer(0, map[string]bool{})
						err := v.DeleteBuffer(0, deleteBufferOpts)
						if err != nil {
							sess.showOrgMessage("DeleteBuffer error %v", err)
						} else {
							sess.showOrgMessage("DeleteBuffer successful")
						}
					*/

				} else if cmd == "q!" || cmd == "quit!" {
					// do nothing = allow editor to be closed

				} else if p.isModified() {
					p.mode = NORMAL
					p.command = ""
					p.command_line = ""
					sess.showEdMessage("No write since last change")
					return false
				}
				deleteBufferOpts := map[string]bool{
					"force":  true,
					"unload": false,
				}
				//err = v.DeleteBuffer(0, map[string]bool{})
				err := v.DeleteBuffer(p.vbuf, deleteBufferOpts)
				if err != nil {
					sess.showOrgMessage("DeleteBuffer error %v", err)
				} else {
					sess.showOrgMessage("DeleteBuffer successful")
				}

				index := -1
				for i := range editors {
					if editors[i] == p {
						index = i
						break
					}
				}
				copy(editors[index:], editors[index+1:])
				editors = editors[:len(editors)-1]

				if p.linked_editor != nil {
					index := -1
					for i := range editors {
						if editors[i] == p.linked_editor {
							index = i
							break
						}
					}
					copy(editors[index:], editors[index+1:])
					editors = editors[:len(editors)-1]
				}

				if len(editors) > 0 {

					p = editors[0] //kluge should move in some logical fashion
					err = v.SetCurrentBuffer(p.vbuf)
					if err != nil {
						sess.showOrgMessage("Error setting current buffer: %v", err)
					}
					sess.positionEditors()
					sess.eraseRightScreen()
					sess.drawEditors()

				} else { // we've quit the last remaining editor(s)
					// unless commented out earlier sess.p.quit <- causes panic
					//sess.p = nil
					sess.editorMode = false
					sess.eraseRightScreen()

					if sess.divider < 10 {
						sess.cfg.ed_pct = 80
						moveDivider(80)
					}

					//org.drawPreviewWindow()
					org.drawPreview()
					sess.returnCursor() //because main while loop if started in editor_mode -- need this 09302020
				}

				return false
			} //end quit_cmds

			// for testing looking at message buffer
			if cmd == "s" { //switch buffer
				bufs, _ := v.Buffers()
				if int(p.vbuf) == 2 {
					_ = v.SetCurrentBuffer(bufs[len(bufs)-1])
					p.vbuf = bufs[len(bufs)-1]
				} else {
					_ = v.SetCurrentBuffer(bufs[1])
					p.vbuf = bufs[1]
				}
				p.command_line = ""
				p.mode = NORMAL
				p.refreshScreen()
				return true
			}

			if cmd0, found := e_lookup_C[cmd]; found {
				cmd0(p)
				p.command_line = ""
				p.mode = NORMAL
				return false
			}

			sess.showEdMessage("\x1b[41mNot an editor command: %s\x1b[0m", cmd)
			p.mode = NORMAL
			p.command_line = ""
			return false
		} //end 'r'

		if c == DEL_KEY || c == BACKSPACE {
			if len(p.command_line) > 0 {
				p.command_line = p.command_line[:len(p.command_line)-1]
			}
		} else {
			p.command_line += string(c)
		}

		sess.showEdMessage(":%s", p.command_line)
		return false //end EX_COMMAND
	}

	/////////////////below everything besides EX_COMMAND///////////////////////////////////

	if z, found := termcodes[c]; found {
		v.FeedKeys(z, "t", true)
	} else {
		_, err := v.Input(string(c))
		if err != nil {
			sess.showEdMessage("Error in nvim.Input: %v", err)
		}
	}

	mode, _ := v.Mode()

	/* debugging
	sess.showOrgMessage("blocking: %t; mode: %s", mode.Blocking, mode.Mode)
	Example of input that blocks is entering a number (eg, 4x) in NORMAL mode
	If blocked = true you can't retrieve buffer with v.BufferLines -
	app just locks up
	*/

	if mode.Blocking {
		return false // don't draw rows - which calls v.BufferLines
	}
	// the only way to get into EX_COMMAND or SEARCH
	if mode.Mode == "c" && p.mode != SEARCH {
		p.command_line = ""
		p.command = ""
		if c == ':' {
			p.mode = EX_COMMAND
			/*
			 below will put nvim back in NORMAL mode but listmango will be
			 in COMMAND_LINE mode, ie 'park' nvim in NORMAL mode
			 and don't feed it any keys while in listmango COMMAND_LINE mode
			*/
			_, err := v.Input("\x1b")
			if err != nil {
				sess.showEdMessage("Error input escape: %v", err)
			}
			sess.showEdMessage(":")
		} else {
			p.mode = SEARCH
			p.searchPrefix = string(c)
			sess.showEdMessage(p.searchPrefix)
		}

		return false
	} else if mode.Mode == "i" && p.mode != INSERT {
		sess.showEdMessage("\x1b[1m-- INSERT --\x1b[0m")
	}

	p.mode = modeMap[mode.Mode] //note that "c" => SEARCH

	switch p.mode {
	//case INSERT, REPLACE, NORMAL:
	case VISUAL, VISUAL_LINE, VISUAL_BLOCK:
		p.vb_highlight = highlightInfo(v)
	case SEARCH:
		// return puts nvim into normal mode so don't need to catch return
		if c == DEL_KEY || c == BACKSPACE {
			if len(p.command_line) > 0 {
				p.command_line = p.command_line[:len(p.command_line)-1]
			}
		} else {
			p.command_line += string(c)
		}

		sess.showEdMessage("%s%s", p.searchPrefix, p.command_line)
		return false
	} // end switch p.mode

	//below is done for everything except SEARCH and EX_COMMAND
	p.bb, _ = v.BufferLines(p.vbuf, 0, -1, true) //reading updated buffer
	pos, _ := v.WindowCursor(w)                  //set screen cx and cy from pos
	p.fr = pos[0] - 1
	p.fc = pos[1]

	if c == 'u' && p.mode == NORMAL {
		showVimMessage()
	}

	if p.mode == PENDING { // -> operator pending (eg. typed 'd')
		return false
	} else {
		return true
	}
}
