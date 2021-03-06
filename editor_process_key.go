package main

import (
	"fmt"
	"github.com/neovim/go-client/nvim"
	"strings"
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

func editorProcessKey(c int, messageBuf nvim.Buffer) bool {
	// need to check that every editor is instantiated with sess.p.mode == NORMAL

	nop := false
	sess.p.command += string(c)
	if strings.IndexAny(sess.p.command[0:1], "\x17\x08\x0c\x02\x05\x09\x06 ") == -1 {
		sess.p.command = ""
	} else {
		nop = true
	}
	if sess.p.mode == NORMAL && c == int(leader[0]) {
		sess.p.command = leader
		nop = true
	}

	/*
		if c == '+' {
			showMessage(v, messageBuf)
			return false
		}
	*/

	sess.showOrgMessage("char = %d", c) //debugging
	var mode *nvim.Mode                 //may not be needed if not debugging

	if nop || sess.p.mode == COMMAND_LINE {
		//do nothing
		sess.showEdMessage("NOP or COMMAND_LINE")
	} else {

		if z, found := termcodes[c]; found {
			v.FeedKeys(z, "t", true)
		} else {
			_, err := v.Input(string(c))
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}

		mode, _ = v.Mode()
		sess.p.mode = modeMap[mode.Mode]
		sess.p.showMessage("blocking = %t; mode = %v", mode.Blocking, mode.Mode) //debugging
	}

	//mode, _ = v.Mode()
	//sess.p.showMessage("blocking = %t; mode = %v", mode.Blocking, mode.Mode) //debugging

	if mode != nil && mode.Blocking == true {
		return false
	}

	/*
		if mode.Mode == "v" || mode.Mode == "V" || mode.Mode == string('\x16') {
			sess.showOrgMessage("mode: %v -> h0=%v; h1= %v", mode.Mode, highlightInfo(v)[0], highlightInfo(v)[1])
		}
	*/

	//sess.p.mode = modeMap[mode.Mode]
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

		//sess.p.showMessage("blocking = %t; mode = %v; command = %v", mode.Blocking, mode.Mode, sess.p.command) //debugging
		if cmd, found := e_lookup2[sess.p.command]; found {
			sess.showEdMessage("Got here")
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
		showVimMessage()
	}
	return true
}