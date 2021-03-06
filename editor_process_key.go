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

var quit_cmds = map[string]struct{}{"quit": z0, "q": z0, "quit!": z0, "q!": z0, "x": z0, "qa": z0}

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

		if p.mode == PREVIEW {
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
	if p.mode == PREVIEW {
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

	if p.mode == SPELLING || p.mode == VIEW_LOG {
		switch c {
		case PAGE_DOWN, ARROW_DOWN, 'j':
			p.previewLineOffset++
			p.drawOverlay()
			return false
		case PAGE_UP, ARROW_UP, 'k':
			if p.previewLineOffset > 0 {
				p.previewLineOffset--
				p.drawOverlay()
				return false
			}
		}
		// enter a number and that's the selected replacement for a mispelling
		if c == '\r' && p.mode == SPELLING {
			v.Input("z=" + p.command_line + "\r") //don't need a check nvim is handling
			//
			p.bb, _ = v.BufferLines(p.vbuf, 0, -1, true) //reading updated buffer
			pos, _ := v.WindowCursor(w)                  //screen cx and cy set from pos
			p.fr = pos[0] - 1
			p.fc = pos[1]
			//
			p.mode = NORMAL
			sess.showOrgMessage(p.command_line)
			return true
		}
		if c == DEL_KEY || c == BACKSPACE {
			if len(p.command_line) > 0 {
				p.command_line = p.command_line[:len(p.command_line)-1]
			}
		} else {
			p.command_line += string(c)
		}
		return false
	}

	if p.mode == NORMAL {
		if len(p.command) == 0 {
			if strings.IndexAny(string(c), "\x17\x08\x0c\x02\x05\x09\x06\x0a\x0b z") != -1 {
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
				//if p.command != "z=" {
				_, err := v.Input("\x1b")
				if err != nil {
					sess.showEdMessage("%v", err)
				}
				//}
				if p.command == " m" || p.command == " l" || p.command == " c" || p.command == "z=" {
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
		// if c is a control character we don't want to send to nvim 07012021
		// except we do want to send carriage return (13), ctrl-v (22) and escape (27)
		// escape is dealt with first thing
	} else if c < 32 && !(c == 13 || c == 22) {
		return false
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

	if (c == 'u' || c == '\x12') && p.mode == NORMAL {
		showLastVimMessage()
	}

	if p.mode == PENDING { // -> operator pending (eg. typed 'd')
		return false
	} else {
		return true
	}
}
